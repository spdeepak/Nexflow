//go:build integration

package mcpstore

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
)

type testStoreFixture struct {
	store    TokenStore
	serverID uuid.UUID
	conn     *sql.DB
}

func newTestStore(t *testing.T) *testStoreFixture {
	t.Helper()
	dbConfig := cfg.AppConfig{
		DBConfig: cfg.DBConfig{
			Host:             "localhost",
			Port:             "5432",
			DBName:           "app.db",
			UserName:         "admin",
			Password:         "admin",
			SSLMode:          "disable",
			Timeout:          10 * time.Second,
			MaxRetry:         3,
			ConnectTimeout:   5 * time.Second,
			StatementTimeout: 30 * time.Second,
			MaxOpenConns:     1,
			MaxIdleConns:     1,
			ConnMaxLifetime:  1 * time.Hour,
			ConnMaxIdleTime:  30 * time.Minute,
		},
	}.DBConfig

	require.NoError(t, db.RunMigrations(dbConfig))
	t.Cleanup(func() {
		_ = os.Remove("app.db")
		_ = os.Remove("app.db-shm")
		_ = os.Remove("app.db-wal")
	})

	conn := db.Connect(dbConfig)
	store := NewTokenStore(New(conn))

	ownerID := insertUser(t, conn)
	serverID := uuid.New()
	_, err := conn.ExecContext(context.Background(),
		`INSERT INTO mcp_server (id, user_id, name, endpoint, transport) VALUES (?, ?, ?, ?, ?)`,
		serverID, ownerID, "test-server", "http://localhost:3000/mcp", enums.McpTransportStreamableHttp)
	require.NoError(t, err)

	return &testStoreFixture{store: store, serverID: serverID, conn: conn}
}

func insertUser(t *testing.T, conn *sql.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := conn.ExecContext(context.Background(), `INSERT INTO users (id, external_id, name) VALUES (?, ?, ?)`, id, id.String(), "owner")
	require.NoError(t, err)
	return id
}

func TestSaveAndLoadClientConfig(t *testing.T) {
	fx := newTestStore(t)
	ctx := context.Background()

	want := OAuthClientConfig{
		AuthURL:      "https://auth.example.com/authorize",
		TokenURL:     "https://auth.example.com/token",
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		Scopes:       []string{"openid", "read"},
		AuthStyle:    oauth2.AuthStyleInHeader,
	}
	require.NoError(t, fx.store.SaveClientConfig(ctx, fx.serverID.String(), want))

	got, err := fx.store.ClientConfig(ctx, fx.serverID.String())
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestSaveClientConfig_UpdatesExisting(t *testing.T) {
	fx := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, fx.store.SaveClientConfig(ctx, fx.serverID.String(), OAuthClientConfig{
		AuthURL:   "https://a.example.com/authorize",
		TokenURL:  "https://a.example.com/token",
		ClientID:  "old-client",
		Scopes:    []string{"read"},
		AuthStyle: oauth2.AuthStyleAutoDetect,
	}))
	require.NoError(t, fx.store.SaveClientConfig(ctx, fx.serverID.String(), OAuthClientConfig{
		AuthURL:   "https://b.example.com/authorize",
		TokenURL:  "https://b.example.com/token",
		ClientID:  "new-client",
		Scopes:    []string{"write"},
		AuthStyle: oauth2.AuthStyleInParams,
	}))

	got, err := fx.store.ClientConfig(ctx, fx.serverID.String())
	require.NoError(t, err)
	require.Equal(t, "new-client", got.ClientID)
	require.Equal(t, "https://b.example.com/authorize", got.AuthURL)
	require.Equal(t, []string{"write"}, got.Scopes)
	require.Equal(t, oauth2.AuthStyleInParams, got.AuthStyle)
}

func TestSaveAndDeleteGrant(t *testing.T) {
	fx := newTestStore(t)
	ctx := context.Background()

	userID := insertUser(t, fx.conn)
	tok := &oauth2.Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, fx.store.SaveGrant(ctx, userID.String(), fx.serverID.String(), tok))
	require.NoError(t, fx.store.SaveClientConfig(ctx, fx.serverID.String(), OAuthClientConfig{
		AuthURL:  "https://auth.example.com/authorize",
		TokenURL: "https://auth.example.com/token",
		ClientID: "client-1",
	}))

	grant, err := fx.store.LoadGrant(ctx, userID.String(), fx.serverID.String())
	require.NoError(t, err)
	require.Equal(t, "access-1", grant.AccessToken)
	require.Equal(t, "refresh-1", grant.RefreshToken)
	require.WithinDuration(t, time.Now().Add(1*time.Hour), grant.Expiry, 5*time.Second)

	mutated := &oauth2.Token{AccessToken: "access-2", RefreshToken: "refresh-2"}
	require.NoError(t, fx.store.SaveGrant(ctx, userID.String(), fx.serverID.String(), mutated))
	grant, err = fx.store.LoadGrant(ctx, userID.String(), fx.serverID.String())
	require.NoError(t, err)
	require.Equal(t, "access-2", grant.AccessToken)
	require.Equal(t, "refresh-2", grant.RefreshToken)

	require.NoError(t, fx.store.DeleteGrant(ctx, userID.String(), fx.serverID.String()))
	_, err = fx.store.LoadGrant(ctx, userID.String(), fx.serverID.String())
	require.Error(t, err)
}

func TestClientConfig_BeforeSave(t *testing.T) {
	fx := newTestStore(t)
	_, err := fx.store.ClientConfig(context.Background(), fx.serverID.String())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no rows")
}

func TestScopesRoundTrip(t *testing.T) {
	fx := newTestStore(t)
	ctx := context.Background()

	scopes := []string{"scope-a", "scope-b", "scope-c"}
	require.NoError(t, fx.store.SaveClientConfig(ctx, fx.serverID.String(), OAuthClientConfig{
		AuthURL:  "https://a.example.com/authorize",
		TokenURL: "https://a.example.com/token",
		ClientID: "c",
		Scopes:   scopes,
	}))

	got, err := fx.store.ClientConfig(ctx, fx.serverID.String())
	require.NoError(t, err)
	require.Equal(t, scopes, got.Scopes)

	row, err := fx.store.(*service).query.GetMCPOAuthClientConfig(ctx, fx.serverID)
	require.NoError(t, err)
	require.JSONEq(t, `["scope-a","scope-b","scope-c"]`, row.Scopes)
}
