package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/adk/v2/auth"
)

type persistingTokenSource struct {
	inner          oauth2.TokenSource
	userID, connID string
	store          TokenStore

	mu   sync.Mutex
	last string
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.inner.Token()
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	changed := tok.AccessToken != p.last
	if changed {
		p.last = tok.AccessToken
	}
	p.mu.Unlock()

	if changed {
		if err = p.store.SaveGrant(context.Background(), p.userID, p.connID, tok); err != nil {
			slog.Error("failed to persist refreshed mcp token ", "userId", p.userID, "connectionId", p.connID, "error", err)
		}
	}
	return tok, nil
}

func credentialProviderFor(ctx context.Context, mcpServerID, userID, connID string, store TokenStore) (auth.CredentialProvider, error) {
	cc, err := store.ClientConfig(ctx, mcpServerID)
	if err != nil {
		return nil, fmt.Errorf("loading oauth client config: %w", err)
	}
	grant, err := store.LoadGrant(ctx, userID, connID)
	if err != nil {
		return nil, fmt.Errorf("loading oauth grant: %w", err)
	}

	cfg := &oauth2.Config{
		ClientID:     cc.ClientID,
		ClientSecret: cc.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   cc.AuthURL,
			TokenURL:  cc.TokenURL,
			AuthStyle: cc.AuthStyle,
		},
		Scopes: cc.Scopes,
	}
	base := cfg.TokenSource(context.Background(), &oauth2.Token{
		AccessToken:  grant.AccessToken,
		RefreshToken: grant.RefreshToken,
		Expiry:       grant.Expiry,
	})
	return auth.TokenSourceProvider(&persistingTokenSource{
		inner: base, userID: userID, connID: connID, store: store,
	}), nil
}

// StoredToken is everything needed to rebuild an oauth2.Config + oauth2.Token
// for one (user, MCP connection) OAuth grant.
type StoredToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time

	// Fixed at connect time (from the MCP server's discovered AS metadata,
	// or a manually-registered client for providers without dynamic
	// registration). A refresh never changes these — Save() below only
	// touches the three fields above.
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// OAuthClientConfig is registered once per MCP server — shared across
// every user who connects to that server. Populate it via discovery/DCR
// when the server supports it, or a one-time manual setup when it doesn't.
type OAuthClientConfig struct {
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string // empty for a public/PKCE-only client
	Scopes       []string
	AuthStyle    oauth2.AuthStyle // AuthStyleAutoDetect is fine until a provider proves otherwise
}

// OAuthGrant is the per-(user, connection) token pair — the only thing
// that's actually unique to a given user.
type OAuthGrant struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

// TokenStore persists OAuth grants per (userID, connectionID). Key on both —
// a user can connect the same MCP twice, or connect several different MCPs.
type TokenStore interface {
	ClientConfig(ctx context.Context, mcpServerID string) (OAuthClientConfig, error)
	LoadGrant(ctx context.Context, userID, connectionID string) (OAuthGrant, error)
	SaveGrant(ctx context.Context, userID, connectionID string, tok *oauth2.Token) error
}

var ErrNotConnected = errors.New("mcpconnect: no token stored for this user/connection")
