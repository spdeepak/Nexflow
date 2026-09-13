package mcpstore

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

var ErrNotConnected = errors.New("mcpconnect: no token stored for this user/connection")
var ErrReauthRequired = errors.New("mcpconnect: oauth grant is no longer valid, re-authorization required")

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
		var re *oauth2.RetrieveError
		if errors.As(err, &re) && isDeadGrantError(re.ErrorCode) {
			if derr := p.store.DeleteGrant(context.Background(), p.userID, p.connID); derr != nil {
				slog.Error("failed to delete dead oauth grant", "userId", p.userID, "connectionId", p.connID, "error", derr)
			}
			return nil, ErrReauthRequired
		}
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

func CredentialProviderFor(ctx context.Context, mcpServerID, userID, connID string, store TokenStore) (auth.CredentialProvider, error) {
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

// isDeadGrantError reports whether the token endpoint rejected the refresh
// because the stored grant itself is no longer usable — as opposed to a
// transient failure (network or 5xx) that should simply be retried.
func isDeadGrantError(code string) bool {
	switch code {
	case "invalid_grant", "invalid_scope", "unauthorized_client":
		return true
	default:
		return false
	}
}
