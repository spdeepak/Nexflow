// Package oauth implements the client side of the OAuth 2.0 authorization code
// flow with PKCE and a loopback redirect, used by the desktop app to connect an
// MCP server. Discovery (RFC 8414 / OIDC) and the callback listener are kept
// transport-agnostic so callers can drive the whole flow themselves.
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// DefaultRedirectURI is the redirect URI used when the user does not register
// one. Many providers accept a loopback IP with an arbitrary port, but to work
// with providers that require an exact match the port is fixed.
const DefaultRedirectURI = "http://127.0.0.1:48421/callback"

// Metadata is the subset of the server's discovery document (RFC 8414, with an
// OpenID Connect fallback) that we need to build an oauth2.Config.
type Metadata struct {
	AuthorizationURL string
	TokenURL         string
	ScopesSupported  []string
}

// Discover fetches the provider's authorization server metadata from the
// well-known endpoints served by the given baseURL's origin. It tries RFC 8414
// first and falls back to the OpenID Connect configuration document.
func Discover(ctx context.Context, hc *http.Client, baseURL string) (*Metadata, error) {
	origin, err := originOf(baseURL)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSuffix(origin, "/")
	candidates := []string{
		root + "/.well-known/oauth-authorization-server",
		root + "/.well-known/openid-configuration",
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	var lastErr error
	for _, u := range candidates {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err := hc.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("discovery endpoint %s returned %s", u, resp.Status)
			continue
		}
		var doc struct {
			AuthorizationEndpoint string   `json:"authorization_endpoint"`
			TokenEndpoint         string   `json:"token_endpoint"`
			ScopesSupported       []string `json:"scopes_supported"`
		}
		if err := json.Unmarshal(body, &doc); err != nil {
			lastErr = fmt.Errorf("parsing discovery document at %s: %w", u, err)
			continue
		}
		if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" {
			lastErr = fmt.Errorf("discovery document at %s is missing authorization/token endpoints", u)
			continue
		}
		return &Metadata{
			AuthorizationURL: doc.AuthorizationEndpoint,
			TokenURL:         doc.TokenEndpoint,
			ScopesSupported:  doc.ScopesSupported,
		}, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no well-known discovery endpoints found")
	}
	return nil, fmt.Errorf("oauth discovery failed for %s: %w", origin, lastErr)
}

func originOf(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid base url %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid base url %q: scheme must be http or https", raw)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid base url %q: missing host", raw)
	}
	return u.Scheme + "://" + u.Host, nil
}

// GenerateVerifier returns a random PKCE code verifier suitable for S256.
func GenerateVerifier() (string, error) {
	return randomBase64URL(32)
}

// GenerateState returns a random state value for the authorization request.
func GenerateState() (string, error) {
	return randomBase64URL(16)
}

func randomBase64URL(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random value: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// AuthorizationURL builds the authorization request URL with PKCE S256 and the
// redirect URI, ready to be opened in a browser.
func AuthorizationURL(cfg *oauth2.Config, state, verifier string) (string, error) {
	if cfg.ClientID == "" {
		return "", errors.New("oauth2 config is missing a client id")
	}
	opts := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(verifier)}
	if cfg.RedirectURL != "" {
		opts = append(opts, oauth2.SetAuthURLParam("redirect_uri", cfg.RedirectURL))
	}
	return cfg.AuthCodeURL(state, opts...), nil
}

// ExchangeCode exchanges an authorization code for a token, sending the PKCE
// code verifier and the registered redirect URI.
func ExchangeCode(ctx context.Context, cfg *oauth2.Config, code, verifier string) (*oauth2.Token, error) {
	opts := []oauth2.AuthCodeOption{oauth2.VerifierOption(verifier)}
	if cfg.RedirectURL != "" {
		opts = append(opts, oauth2.SetAuthURLParam("redirect_uri", cfg.RedirectURL))
	}
	return cfg.Exchange(ctx, code, opts...)
}

// CallbackResult carries the authorization code (or the error the provider
// returned) from the loopback callback to the waiting caller.
type CallbackResult struct {
	Code string
	Err  error
}

// CallbackServer is a single-use loopback HTTP listener that receives the
// provider's redirect after the user approves in the browser.
type CallbackServer struct {
	srv    *http.Server
	ln     net.Listener
	state  string
	result chan CallbackResult
}

// StartCallbackServer binds a loopback listener on addr (normally
// "127.0.0.1:<port>") and begins serving the callback. It only accepts a
// single callback, matching the given state.
func StartCallbackServer(addr, state string) (*CallbackServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("binding loopback callback listener on %s: %w", addr, err)
	}
	cs := &CallbackServer{
		ln:     ln,
		state:  state,
		result: make(chan CallbackResult, 1),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", cs.handle)
	cs.srv = &http.Server{Handler: mux}
	go func() {
		if err := cs.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			cs.send(CallbackResult{Err: fmt.Errorf("oauth callback server failed: %w", err)})
		}
	}()
	return cs, nil
}

// RedirectURL is the redirect URI that must be used with this listener
// (e.g. registered with the provider as http://127.0.0.1:<port>/callback).
func (cs *CallbackServer) RedirectURL() string {
	return "http://" + cs.ln.Addr().String() + "/callback"
}

// Wait blocks until the callback arrives, the server fails, or ctx is done.
func (cs *CallbackServer) Wait(ctx context.Context) (CallbackResult, error) {
	select {
	case res := <-cs.result:
		return res, res.Err
	case <-ctx.Done():
		return CallbackResult{}, ctx.Err()
	}
}

// Close shuts the listener down. It is safe to call after Wait returns.
func (cs *CallbackServer) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := cs.srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_ = cs.ln.Close()
		return err
	}
	return nil
}

func (cs *CallbackServer) handle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("state") != cs.state {
		http.Error(w, "invalid state", http.StatusBadRequest)
		cs.send(CallbackResult{Err: errors.New("oauth callback state mismatch")})
		return
	}
	if providerErr := q.Get("error"); providerErr != "" {
		http.Error(w, providerErr, http.StatusBadRequest)
		cs.send(CallbackResult{Err: fmt.Errorf("provider denied authorization: %s", providerErr)})
		return
	}
	code := q.Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		cs.send(CallbackResult{Err: errors.New("oauth callback is missing the authorization code")})
		return
	}
	cs.send(CallbackResult{Code: code})
	_, _ = io.WriteString(w, "Authorization received. You can close this window.")
}

func (cs *CallbackServer) send(res CallbackResult) {
	select {
	case cs.result <- res:
	default:
	}
}
