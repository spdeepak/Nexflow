package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"

	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpstore"
	"github.com/spdeepak/nexflow/internal/oauth"
	"github.com/spdeepak/nexflow/internal/schema"
)

// oauthConnectTimeout bounds how long a ConnectOAuthMCP call waits for the user
// to finish the authorization in the browser.
const oauthConnectTimeout = 5 * time.Minute

// ConnectOAuthMCP runs the authorization code flow against the MCP server's
// OAuth provider, stores the OAuth client configuration and the resulting
// grant, and is a no-op safe to call again (re-connect) at any time.
func (a *App) ConnectOAuthMCP(mcpID string) error {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	mcp, err := a.GetMCP(mcpID)
	if err != nil {
		return err
	}
	if mcp.AuthType == nil || *mcp.AuthType != enums.McpAuthTypeOAUTH {
		return fmt.Errorf("mcp %q is not configured for oauth authorization", mcp.Name)
	}
	serverCfg, err := parseOAuthServerConfig(mcp.AuthConfig)
	if err != nil {
		return err
	}

	// Resolve the authorization/token endpoints from the explicit config first,
	// and fall back to RFC 8414 discovery on the MCP server's origin.
	if serverCfg.AuthURL == "" || serverCfg.TokenURL == "" {
		var baseURL string
		if mcp.Endpoint != nil {
			baseURL = *mcp.Endpoint
		}
		if baseURL == "" {
			return errors.New("mcp server has no endpoint; provide auth_url and token_url in the oauth auth config")
		}
		metadata, err := oauth.Discover(ctx, nil, baseURL)
		if err != nil {
			return err
		}
		if serverCfg.AuthURL == "" {
			serverCfg.AuthURL = metadata.AuthorizationURL
		}
		if serverCfg.TokenURL == "" {
			serverCfg.TokenURL = metadata.TokenURL
		}
		if len(serverCfg.Scopes) == 0 {
			serverCfg.Scopes = metadata.ScopesSupported
		}
	}

	redirectURI := serverCfg.RedirectURI
	if redirectURI == "" {
		redirectURI = oauth.DefaultRedirectURI
	}
	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("invalid oauth redirect_uri %q: %w", redirectURI, err)
	}
	if redirectURL.Port() == "" {
		return fmt.Errorf("oauth redirect_uri must include a localhost port: %q", redirectURI)
	}

	verifier, err := oauth.GenerateVerifier()
	if err != nil {
		return err
	}
	state, err := oauth.GenerateState()
	if err != nil {
		return err
	}

	clientCfg := mcpstore.OAuthClientConfig{
		AuthURL:      serverCfg.AuthURL,
		TokenURL:     serverCfg.TokenURL,
		ClientID:     serverCfg.ClientID,
		ClientSecret: serverCfg.ClientSecret,
		Scopes:       serverCfg.Scopes,
		AuthStyle:    oauthAuthStyle(serverCfg.AuthStyle),
	}
	if err := a.tokenStore.SaveClientConfig(ctx, mcpID, clientCfg); err != nil {
		return fmt.Errorf("saving oauth client config: %w", err)
	}

	cfg := &oauth2.Config{
		ClientID:     clientCfg.ClientID,
		ClientSecret: clientCfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   clientCfg.AuthURL,
			TokenURL:  clientCfg.TokenURL,
			AuthStyle: clientCfg.AuthStyle,
		},
		Scopes:      clientCfg.Scopes,
		RedirectURL: redirectURI,
	}

	authURL, err := oauth.AuthorizationURL(cfg, state, verifier)
	if err != nil {
		return err
	}

	cb, err := oauth.StartCallbackServer(redirectURL.Host, state)
	if err != nil {
		return err
	}
	defer cb.Close()

	runtime.BrowserOpenURL(ctx, authURL)

	waitCtx, cancel := context.WithTimeout(ctx, oauthConnectTimeout)
	defer cancel()
	res, err := cb.Wait(waitCtx)
	if err != nil {
		return fmt.Errorf("waiting for oauth authorization: %w", err)
	}

	tok, err := oauth.ExchangeCode(waitCtx, cfg, res.Code, verifier)
	if err != nil {
		return fmt.Errorf("exchanging oauth authorization code: %w", err)
	}

	if err := a.tokenStore.SaveGrant(ctx, a.deviceID.String(), mcpID, tok); err != nil {
		return fmt.Errorf("saving oauth grant: %w", err)
	}
	return nil
}

// IsOAuthConnected reports whether the device's user has a stored OAuth grant
// for the given MCP server, so the UI can surface connect/disconnect state.
func (a *App) IsOAuthConnected(mcpID string) (bool, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := a.tokenStore.LoadGrant(ctx, a.deviceID.String(), mcpID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("checking oauth connection: %w", err)
}

type oauthServerConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	RedirectURI  string   `json:"redirect_uri"`
	AuthStyle    string   `json:"auth_style"`
	Scopes       []string `json:"-"` // handled manually
}

func parseOAuthServerConfig(cfg schema.AuthConfig) (oauthServerConfig, error) {
	var out oauthServerConfig
	data, err := json.Marshal(cfg)
	if err != nil {
		return out, fmt.Errorf("marshaling auth config: %w", err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("parsing oauth config: %w", err)
	}

	// scopes can be a comma-separated string or an array
	switch v := cfg["scopes"].(type) {
	case string:
		for scope := range strings.SplitSeq(v, ",") {
			if scope = strings.TrimSpace(scope); scope != "" {
				out.Scopes = append(out.Scopes, scope)
			}
		}
	case []any:
		for _, scope := range v {
			if str, ok := scope.(string); ok && strings.TrimSpace(str) != "" {
				out.Scopes = append(out.Scopes, strings.TrimSpace(str))
			}
		}
	}

	out.ClientID = strings.TrimSpace(out.ClientID)
	if out.ClientID == "" {
		return out, errors.New("oauth auth config requires a client_id")
	}
	out.AuthURL = strings.TrimSpace(out.AuthURL)
	out.TokenURL = strings.TrimSpace(out.TokenURL)
	out.RedirectURI = strings.TrimSpace(out.RedirectURI)
	out.AuthStyle = strings.TrimSpace(out.AuthStyle)
	return out, nil
}

func oauthAuthStyle(v string) oauth2.AuthStyle {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "in_params", "in_query", "post":
		return oauth2.AuthStyleInParams
	case "in_header", "header":
		return oauth2.AuthStyleInHeader
	default:
		return oauth2.AuthStyleAutoDetect
	}
}
