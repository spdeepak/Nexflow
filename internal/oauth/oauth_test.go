package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestDiscover_RFC8414(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/.well-known/oauth-authorization-server", r.URL.Path)
		writeJSON(t, w, map[string]any{
			"authorization_endpoint": "https://auth.example.com/authorize",
			"token_endpoint":         "https://auth.example.com/token",
			"scopes_supported":       []string{"read", "write"},
		})
	}))
	defer srv.Close()

	md, err := Discover(context.Background(), srv.Client(), srv.URL)
	require.NoError(t, err)
	require.Equal(t, "https://auth.example.com/authorize", md.AuthorizationURL)
	require.Equal(t, "https://auth.example.com/token", md.TokenURL)
	require.Equal(t, []string{"read", "write"}, md.ScopesSupported)
}

func TestDiscover_OIDCFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			http.NotFound(w, r)
		case "/.well-known/openid-configuration":
			writeJSON(t, w, map[string]any{
				"authorization_endpoint": "https://auth.example.com/authorize",
				"token_endpoint":         "https://auth.example.com/token",
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	md, err := Discover(context.Background(), srv.Client(), srv.URL+"/some/mcp/path")
	require.NoError(t, err)
	require.NotEmpty(t, md.AuthorizationURL)
	require.NotEmpty(t, md.TokenURL)
}

func TestDiscover_NoEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	_, err := Discover(context.Background(), srv.Client(), srv.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "discovery failed")
}

func TestDiscover_InvalidBaseURL(t *testing.T) {
	_, err := Discover(context.Background(), http.DefaultClient, "not a url")
	require.Error(t, err)
}

func TestCallbackServer_Success(t *testing.T) {
	state, err := GenerateState()
	require.NoError(t, err)

	cs, err := StartCallbackServer("127.0.0.1:0", state)
	require.NoError(t, err)
	defer cs.Close()

	u, err := url.Parse(cs.RedirectURL())
	require.NoError(t, err)
	u.RawQuery = url.Values{"state": {state}, "code": {"auth-code-123"}}.Encode()

	resp, err := http.Get(u.String())
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	res, err := cs.Wait(context.Background())
	require.NoError(t, err)
	require.Equal(t, "auth-code-123", res.Code)
}

func TestCallbackServer_StateMismatch(t *testing.T) {
	cs, err := StartCallbackServer("127.0.0.1:0", "expected-state")
	require.NoError(t, err)
	defer cs.Close()

	u, err := url.Parse(cs.RedirectURL())
	require.NoError(t, err)
	u.RawQuery = url.Values{"state": {"wrong"}, "code": {"abc"}}.Encode()

	resp, err := http.Get(u.String())
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	_, err = cs.Wait(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "state mismatch")
}

func TestCallbackServer_ProviderError(t *testing.T) {
	cs, err := StartCallbackServer("127.0.0.1:0", "state")
	require.NoError(t, err)
	defer cs.Close()

	u, err := url.Parse(cs.RedirectURL())
	require.NoError(t, err)
	u.RawQuery = url.Values{"state": {"state"}, "error": {"access_denied"}}.Encode()

	resp, err := http.Get(u.String())
	require.NoError(t, err)
	resp.Body.Close()

	_, err = cs.Wait(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "access_denied")
}

func TestGenerateVerifierAndAuthorizationURL(t *testing.T) {
	verifier, err := GenerateVerifier()
	require.NoError(t, err)
	require.Len(t, verifier, 43)

	cfg := &oauth2.Config{
		ClientID:    "client-1",
		RedirectURL: DefaultRedirectURI,
		Endpoint:    oauth2.Endpoint{AuthURL: "https://auth.example.com/authorize", TokenURL: "https://auth.example.com/token"},
	}
	authURL, err := AuthorizationURL(cfg, "state-1", verifier)
	require.NoError(t, err)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	q := parsed.Query()
	require.Equal(t, "state-1", q.Get("state"))
	require.Equal(t, "S256", q.Get("code_challenge_method"))
	require.NotEmpty(t, q.Get("code_challenge"))
	require.Equal(t, DefaultRedirectURI, q.Get("redirect_uri"))
	require.Equal(t, "client-1", q.Get("client_id"))
}

func TestExchangeCode_SendsVerifier(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())
		require.Equal(t, "authorization_code", r.PostForm.Get("grant_type"))
		require.Equal(t, "the-code", r.PostForm.Get("code"))
		require.Equal(t, "the-verifier", r.PostForm.Get("code_verifier"))
		require.Equal(t, DefaultRedirectURI, r.PostForm.Get("redirect_uri"))
		writeJSON(t, w, map[string]any{
			"access_token":  "access-tok",
			"refresh_token": "refresh-tok",
			"expires_in":    3600,
		})
	}))
	defer tokenSrv.Close()

	cfg := &oauth2.Config{
		ClientID:    "client-1",
		RedirectURL: DefaultRedirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.example.com/authorize",
			TokenURL: tokenSrv.URL,
		},
	}
	tok, err := ExchangeCode(context.Background(), cfg, "the-code", "the-verifier")
	require.NoError(t, err)
	require.Equal(t, "access-tok", tok.AccessToken)
	require.Equal(t, "refresh-tok", tok.RefreshToken)
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}
