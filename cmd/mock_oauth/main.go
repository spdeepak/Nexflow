package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpserver"
	"github.com/spdeepak/nexflow/internal/mcpstore"
	"github.com/spdeepak/nexflow/internal/oauth"
)

func main() {
	port := flag.Int("port", 18421, "port for mock OAuth server")
	clientID := flag.String("client-id", "mock-client", "OAuth client ID")
	clientSecret := flag.String("client-secret", "mock-secret", "OAuth client secret")
	redirectURI := flag.String("redirect-uri", "", "redirect URI (default: http://127.0.0.1:<port>/callback)")
	createMCP := flag.Bool("create-mcp", true, "auto-create an MCP server entry in the database")
	mcpName := flag.String("mcp-name", "Mock OAuth MCP", "name for the created MCP server")
	flag.Parse()

	if *redirectURI == "" {
		*redirectURI = fmt.Sprintf("http://127.0.0.1:%d/callback", *port)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	cfg := config.NewConfiguration()

	if err := db.RunMigrations(cfg.DBConfig); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	conn := db.Connect(cfg.DBConfig)

	mock := oauth.NewMockServer(*clientID, *clientSecret, *redirectURI, *port)
	baseURL, err := mock.Start()
	if err != nil {
		log.Fatalf("start mock server: %v", err)
	}
	slog.Info("mock OAuth server started", "base_url", baseURL, "client_id", *clientID, "redirect_uri", *redirectURI)

	// Ensure a user exists for the MCP server FK
	userID := uuid.New()
	_, err = conn.ExecContext(context.Background(),
		`INSERT INTO users (id, external_id, name) VALUES (?, ?, ?)`,
		userID, userID.String(), "mock-user")
	if err != nil {
		log.Fatalf("create user: %v", err)
	}

	var mcpID uuid.UUID
	if *createMCP {
		mcpID = uuid.New()
		mcpQ := mcpserver.New(conn)
		_, err = mcpQ.CreateMCPServer(context.Background(), mcpserver.CreateMCPServerParams{
			ID:        mcpID,
			UserID:    userID,
			Name:      *mcpName,
			Endpoint:  baseURL + "/mcp",
			Transport: enums.McpTransportStreamableHttp,
			AuthType:  enums.McpAuthTypeOAUTH,
			AuthConfig: mustMarshal(map[string]any{
				"client_id":     *clientID,
				"client_secret": *clientSecret,
				"auth_url":      baseURL + "/authorize",
				"token_url":     baseURL + "/token",
				"redirect_uri":  *redirectURI,
				"scopes":        []string{"openid", "profile", "email"},
			}),
			Command:             "",
			Args:                "[]",
			AllowedTools:        "[]",
			RequireConfirmation: false,
			ConfirmationRules:   sql.NullString{String: "{}", Valid: true},
			IsActive:            true,
		})
		if err != nil {
			log.Fatalf("create MCP: %v", err)
		}
		slog.Info("created MCP server", "id", mcpID, "name", *mcpName)
	}

	// Also pre-populate the client config in mcpstore for immediate use
	if *createMCP {
		store := mcpstore.NewTokenStore(mcpstore.New(conn))
		if err := store.SaveClientConfig(context.Background(), mcpID.String(), mcpstore.OAuthClientConfig{
			AuthURL:      baseURL + "/authorize",
			TokenURL:     baseURL + "/token",
			ClientID:     *clientID,
			ClientSecret: *clientSecret,
			Scopes:       []string{"openid", "profile", "email"},
			AuthStyle:    0, // auto
		}); err != nil {
			slog.Warn("pre-populate client config", "error", err)
		}
	}

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Printf("Mock OAuth Server Ready\n")
	fmt.Printf("  Base URL:      %s\n", baseURL)
	fmt.Printf("  Client ID:     %s\n", *clientID)
	fmt.Printf("  Client Secret: %s\n", *clientSecret)
	fmt.Printf("  Redirect URI:  %s\n", *redirectURI)
	if *createMCP {
		fmt.Printf("  MCP Server ID: %s\n", mcpID)
		fmt.Printf("  MCP Name:      %s\n", *mcpName)
	}
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("In the Nexflow app:")
	fmt.Println("  1. Open MCP Servers page")
	if *createMCP {
		fmt.Printf("  2. Find \"%s\" (OAuth type) and click Connect\n", *mcpName)
	} else {
		fmt.Println("  2. Add new MCP with Auth Type = oauth, paste the values above")
		fmt.Println("  3. Click Connect")
	}
	fmt.Println("  4. Browser opens -> auto-approves -> returns to app")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutting down")
	_ = mock.Stop()
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
