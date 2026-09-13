package mcpstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type (
	// OAuthClientConfig is registered once per MCP server — shared across
	// every user who connects to that server. Populate it via discovery/DCR
	// when the server supports it, or a one-time manual setup when it doesn't.
	OAuthClientConfig struct {
		AuthURL      string
		TokenURL     string
		ClientID     string
		ClientSecret string // empty for a public/PKCE-only client
		Scopes       []string
		AuthStyle    oauth2.AuthStyle // AuthStyleAutoDetect is fine until a provider proves otherwise
	}

	// OAuthGrant is the per-(user, connection) token pair — the only thing
	// that's actually unique to a given user.
	OAuthGrant struct {
		AccessToken  string
		RefreshToken string
		Expiry       time.Time
	}

	// service is a TokenStore backed by the database. It persists the
	// per-MCP-server OAuth client config and the per-(user, connection) grant that
	// credentialProviderFor and persistingTokenSource need to rebuild an oauth2
	// config and token for a run.
	service struct {
		query Querier
	}
)

// TokenStore persists OAuth grants per (userID, connectionID). Key on both —
// a user can connect the same MCP twice, or connect several different MCPs.
type TokenStore interface {
	ClientConfig(ctx context.Context, mcpServerID string) (OAuthClientConfig, error)
	SaveClientConfig(ctx context.Context, mcpServerID string, cfg OAuthClientConfig) error
	LoadGrant(ctx context.Context, userID, connectionID string) (OAuthGrant, error)
	SaveGrant(ctx context.Context, userID, connectionID string, tok *oauth2.Token) error
	DeleteGrant(ctx context.Context, userID, connectionID string) error
}

// NewTokenStore returns a TokenStore that reads and writes OAuth grants through the provided mcpstore querier.
func NewTokenStore(query Querier) TokenStore {
	return &service{query: query}
}

func (s *service) ClientConfig(ctx context.Context, mcpServerID string) (OAuthClientConfig, error) {
	serverID, err := uuid.Parse(mcpServerID)
	if err != nil {
		return OAuthClientConfig{}, fmt.Errorf("parsing mcp server id %q: %w", mcpServerID, err)
	}
	row, err := s.query.GetMCPOAuthClientConfig(ctx, serverID)
	if err != nil {
		return OAuthClientConfig{}, err
	}
	var scopes []string
	if err := json.Unmarshal([]byte(row.Scopes), &scopes); err != nil {
		return OAuthClientConfig{}, fmt.Errorf("unmarshaling scopes: %w", err)
	}
	return OAuthClientConfig{
		AuthURL:      row.AuthUrl,
		TokenURL:     row.TokenUrl,
		ClientID:     row.ClientID,
		ClientSecret: row.ClientSecret.String,
		Scopes:       scopes,
		AuthStyle:    row.AuthStyle,
	}, nil
}

func (s *service) SaveClientConfig(ctx context.Context, mcpServerID string, cfg OAuthClientConfig) error {
	serverID, err := uuid.Parse(mcpServerID)
	if err != nil {
		return fmt.Errorf("parsing mcp server id %q: %w", mcpServerID, err)
	}
	scopes, err := json.Marshal(cfg.Scopes)
	if err != nil {
		return fmt.Errorf("marshaling scopes: %w", err)
	}
	_, err = s.query.UpsertMCPOAuthClientConfig(ctx, UpsertMCPOAuthClientConfigParams{
		McpServerID:  serverID,
		AuthUrl:      cfg.AuthURL,
		TokenUrl:     cfg.TokenURL,
		ClientID:     cfg.ClientID,
		ClientSecret: sql.NullString{String: cfg.ClientSecret, Valid: cfg.ClientSecret != ""},
		Scopes:       string(scopes),
		AuthStyle:    cfg.AuthStyle,
	})
	return err
}

func (s *service) LoadGrant(ctx context.Context, userID, connectionID string) (OAuthGrant, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return OAuthGrant{}, fmt.Errorf("parsing user id %q: %w", userID, err)
	}
	connID, err := uuid.Parse(connectionID)
	if err != nil {
		return OAuthGrant{}, fmt.Errorf("parsing connection id %q: %w", connectionID, err)
	}
	row, err := s.query.GetMCPOAuthGrant(ctx, GetMCPOAuthGrantParams{
		UserID:      uid,
		McpServerID: connID,
	})
	if err != nil {
		return OAuthGrant{}, err
	}
	grant := OAuthGrant{
		AccessToken:  row.AccessToken,
		RefreshToken: row.RefreshToken.String,
	}
	if row.Expiry.Valid {
		grant.Expiry = row.Expiry.Time
	}
	return grant, nil
}

func (s *service) SaveGrant(ctx context.Context, userID, connectionID string, tok *oauth2.Token) error {
	if tok == nil {
		return errors.New("cannot save a nil oauth token")
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("parsing user id %q: %w", userID, err)
	}
	connID, err := uuid.Parse(connectionID)
	if err != nil {
		return fmt.Errorf("parsing connection id %q: %w", connectionID, err)
	}
	_, err = s.query.UpsertMCPOAuthGrant(ctx, UpsertMCPOAuthGrantParams{
		UserID:       uid,
		McpServerID:  connID,
		AccessToken:  tok.AccessToken,
		RefreshToken: sql.NullString{String: tok.RefreshToken, Valid: tok.RefreshToken != ""},
		Expiry:       sql.NullTime{Time: tok.Expiry, Valid: !tok.Expiry.IsZero()},
	})
	return err
}

func (s *service) DeleteGrant(ctx context.Context, userID, connectionID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("parsing user id %q: %w", userID, err)
	}
	connID, err := uuid.Parse(connectionID)
	if err != nil {
		return fmt.Errorf("parsing connection id %q: %w", connectionID, err)
	}
	return s.query.DeleteMCPOAuthGrant(ctx, DeleteMCPOAuthGrantParams{
		UserID:      uid,
		McpServerID: connID,
	})
}
