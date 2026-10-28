package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os/user"
	"path"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/spdeepak/nexflow/internal/agentmcp"
	"github.com/spdeepak/nexflow/internal/agents"
	"github.com/spdeepak/nexflow/internal/agentskills"
	"github.com/spdeepak/nexflow/internal/application"
	"github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/device"
	"github.com/spdeepak/nexflow/internal/events"
	"github.com/spdeepak/nexflow/internal/mcpserver"
	"github.com/spdeepak/nexflow/internal/mcpstore"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/runner"
	"github.com/spdeepak/nexflow/internal/runs"
	"github.com/spdeepak/nexflow/internal/sessions"
	"github.com/spdeepak/nexflow/internal/sessionstate"
	"github.com/spdeepak/nexflow/internal/skills"
	"github.com/spdeepak/nexflow/internal/users"
	"github.com/spdeepak/nexflow/pkg/logging"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	slog.SetDefault(slog.New(logging.NewDefaultHandler()))
	cfg := config.NewConfiguration()

	_ = db.RunMigrations(cfg.DBConfig)
	dbConnection := db.Connect(cfg.DBConfig)

	// Agent Skill
	agentSkillQuery := agentskills.New(dbConnection)
	agentSkillService := agentskills.NewService(agentSkillQuery)
	// Agent MCP
	agentMcpQuery := agentmcp.New(dbConnection)
	agentMcpService := agentmcp.NewService(agentMcpQuery)
	// Skill
	skillQuery := skills.New(dbConnection)
	skillService := skills.NewService(skillQuery)
	// Model
	modelCredsQuery := modelcredentials.New(dbConnection)
	modelService := modelcredentials.NewService(modelCredsQuery)
	// Agent
	agentsQuery := agents.New(dbConnection)
	agentService := agents.NewService(agentsQuery, agentSkillService, agentMcpService, modelService)
	// MCP
	mcpQuery := mcpserver.New(dbConnection)
	mcpService := mcpserver.NewService(mcpQuery)
	// User
	userQuery := users.New(dbConnection)
	userService := users.NewService(userQuery)
	// Device
	deviceQuery := device.New(dbConnection)
	deviceService := device.NewService(deviceQuery)
	deviceID := deviceService.GetDeviceDetail(context.Background())

	// Local app user. The device serial is the external identity; the user row
	// keeps its own UUID key, which is what sessions and owned rows reference.
	name := ""
	if currentUser, err := user.Current(); err == nil {
		name = currentUser.Name
	}
	appUser, err := userService.EnsureAppUser(context.Background(), deviceID, name)
	if err != nil {
		slog.Error("failed to initialise local app user", "deviceID", deviceID, "error", err)
		panic(err)
	}
	userID := appUser.ID

	// Chat
	sessionQuery := sessions.New(dbConnection)
	eventQuery := events.New(dbConnection)
	runQuery := runs.New(dbConnection)
	stateQuery := sessionstate.New(dbConnection)
	mcpStoreQuery := mcpstore.NewTokenStore(mcpstore.New(dbConnection))
	chatService := runner.NewChat(sessionQuery, eventQuery, runQuery, stateQuery, agentService, modelService, userID, "nexflow", mcpStoreQuery)

	app := application.NewApp(agentService, skillService, modelService, userService, chatService, mcpService, deviceID, userID, mcpStoreQuery)

	startEmbeddedServer(18391, app)

	err = wails.Run(&options.App{
		Title:  "Nexflow",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			app.Startup(ctx)
		},
		Bind: []any{
			app,
		},
	})
	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}
}

func startEmbeddedServer(port int, app *application.App) {
	// Serve static frontend build files. The embed contains frontend/ itself,
	// so mount that subtree at / to keep the app at http://localhost:<port>/.
	frontend, err := fs.Sub(assets, "frontend")
	if err != nil {
		panic(fmt.Errorf("failed to mount embedded frontend: %w", err))
	}
	// Client-side views have no file of their own, so any unknown extensionless
	// path falls back to index.html and lets app.js route it (/models, /mcp, ...).
	fileServer := http.FileServer(http.FS(frontend))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(frontend, strings.TrimPrefix(r.URL.Path, "/")); err != nil && path.Ext(r.URL.Path) == "" {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	// Expose HTTP/REST endpoints for remote devices
	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "running from production app!"}`)
	})

	// Same App methods Wails binds in the webview, for the browser frontend.
	http.HandleFunc("/api/rpc/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		// Any web page can POST to localhost, and these methods can launch MCP
		// processes, so only the app's own origin may call them.
		if origin := r.Header.Get("Origin"); origin != "" && !isLocalOrigin(origin, port) {
			http.Error(w, "cross-origin requests are not allowed", http.StatusForbidden)
			return
		}

		var args []json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, "invalid arguments: "+err.Error(), http.StatusBadRequest)
			return
		}

		result, err := app.HandleRPC(strings.TrimPrefix(r.URL.Path, "/api/rpc/"), args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if result == nil {
			fmt.Fprint(w, "null")
			return
		}
		body, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(body))
	})

	// Loopback only: these endpoints act with the local user's full authority.
	go func() {
		err = http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), nil)
		if err != nil {
			slog.Error("HTTP server error", "error", err)
		}
	}()
}

func isLocalOrigin(origin string, port int) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Port() != strconv.Itoa(port) {
		return false
	}
	return u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"
}
