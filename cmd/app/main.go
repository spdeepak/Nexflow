package main

import (
	"context"
	"embed"
	"log/slog"

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
	// Agent
	agentsQuery := agents.New(dbConnection)
	agentService := agents.NewService(agentsQuery, agentSkillService, agentMcpService)
	// Skill
	skillQuery := skills.New(dbConnection)
	skillService := skills.NewService(skillQuery)
	// Model
	modelCredsQuery := modelcredentials.New(dbConnection)
	modelService := modelcredentials.NewService(modelCredsQuery)
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
	// Chat
	sessionQuery := sessions.New(dbConnection)
	eventQuery := events.New(dbConnection)
	runQuery := runs.New(dbConnection)
	stateQuery := sessionstate.New(dbConnection)
	chatService := runner.NewChat(sessionQuery, eventQuery, runQuery, stateQuery, agentService, modelService, deviceID, "nexflow")

	app := application.NewApp(agentService, skillService, modelService, userService, chatService, mcpService, deviceID)

	err := wails.Run(&options.App{
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
