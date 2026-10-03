package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// MockServer is an in-process OAuth 2.0 authorization server for development
// and testing. It auto-approves all consent, returns dummy tokens, and serves
// RFC 8414 discovery. Not safe for production use.
type MockServer struct {
	mu           sync.Mutex
	srv          *http.Server
	clientID     string
	clientSecret string
	redirectURI  string
	authCodes    map[string]authCodeData
	tokens       map[string]tokenData
	port         int
}

type authCodeData struct {
	redirectURI  string
	codeVerifier string
	expiresAt    time.Time
}

type tokenData struct {
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

// NewMockServer creates a mock OAuth server with the given client credentials
// and redirect URI. If port is 0, a random free port is chosen.
func NewMockServer(clientID, clientSecret, redirectURI string, port int) *MockServer {
	return &MockServer{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		authCodes:    make(map[string]authCodeData),
		tokens:       make(map[string]tokenData),
		port:         port,
	}
}

// Start begins listening. Returns the base URL (e.g. "http://127.0.0.1:18421").
func (m *MockServer) Start() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.srv != nil {
		return "", fmt.Errorf("mock server already started")
	}

	addr := fmt.Sprintf("127.0.0.1:%d", m.port)
	ln, err := listen(addr)
	if err != nil {
		return "", err
	}
	m.port = ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", m.handleAuthorize)
	mux.HandleFunc("/token", m.handleToken)
	mux.HandleFunc("/.well-known/oauth-authorization-server", m.handleDiscovery)
	mux.HandleFunc("/.well-known/openid-configuration", m.handleDiscovery)

	m.srv = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		_ = m.srv.Serve(ln)
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", m.port)
	return base, nil
}

// BaseURL returns the server's base URL after Start().
func (m *MockServer) BaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", m.port)
}

// ClientID returns the configured client ID.
func (m *MockServer) ClientID() string { return m.clientID }

// ClientSecret returns the configured client secret.
func (m *MockServer) ClientSecret() string { return m.clientSecret }

// RedirectURI returns the configured redirect URI.
func (m *MockServer) RedirectURI() string { return m.redirectURI }

// Stop shuts the server down.
func (m *MockServer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := m.srv.Shutdown(ctx)
	m.srv = nil
	return err
}

func (m *MockServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")

	if clientID != m.clientID {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}
	if redirectURI != m.redirectURI {
		http.Error(w, "redirect_uri mismatch", http.StatusBadRequest)
		return
	}
	if state == "" {
		http.Error(w, "state required", http.StatusBadRequest)
		return
	}

	// Generate authorization code
	code := randomString(32)
	m.mu.Lock()
	m.authCodes[code] = authCodeData{
		redirectURI:  redirectURI,
		codeVerifier: codeChallenge, // store challenge for PKCE verification at token endpoint
		expiresAt:    time.Now().Add(10 * time.Minute),
	}
	m.mu.Unlock()

	// Auto-approve: redirect immediately back to the app's callback
	u, _ := url.Parse(redirectURI)
	u.RawQuery = url.Values{
		"code":  {code},
		"state": {state},
	}.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (m *MockServer) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	grantType := r.Form.Get("grant_type")
	if grantType != "authorization_code" {
		http.Error(w, "unsupported grant_type", http.StatusBadRequest)
		return
	}
	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")
	code := r.Form.Get("code")
	codeVerifier := r.Form.Get("code_verifier")
	redirectURI := r.Form.Get("redirect_uri")

	if clientID != m.clientID || clientSecret != m.clientSecret {
		http.Error(w, "invalid client credentials", http.StatusUnauthorized)
		return
	}
	if redirectURI != m.redirectURI {
		http.Error(w, "redirect_uri mismatch", http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	ac, ok := m.authCodes[code]
	if !ok {
		m.mu.Unlock()
		http.Error(w, "invalid or expired authorization code", http.StatusBadRequest)
		return
	}
	// PKCE verification: check code_verifier produces the stored challenge
	if !verifyPKCE(codeVerifier, ac.codeVerifier) {
		m.mu.Unlock()
		http.Error(w, "invalid code_verifier", http.StatusBadRequest)
		return
	}
	if time.Now().After(ac.expiresAt) {
		m.mu.Unlock()
		http.Error(w, "authorization code expired", http.StatusBadRequest)
		return
	}
	delete(m.authCodes, code)
	m.mu.Unlock()

	accessToken := "mock-access-" + randomString(24)
	refreshToken := "mock-refresh-" + randomString(24)
	expiresIn := 3600

	m.mu.Lock()
	m.tokens[accessToken] = tokenData{
		accessToken:  accessToken,
		refreshToken: refreshToken,
		expiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
	m.mu.Unlock()

	writeJSONResponse(w, map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
		"refresh_token": refreshToken,
		"scope":         "openid profile email",
	})
}

func (m *MockServer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	base := m.BaseURL()
	writeJSONResponse(w, map[string]any{
		"authorization_endpoint":           base + "/authorize",
		"token_endpoint":                   base + "/token",
		"scopes_supported":                 []string{"openid", "profile", "email"},
		"response_types_supported":         []string{"code"},
		"grant_types_supported":            []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported": []string{"S256"},
	})
}

func verifyPKCE(verifier, challenge string) bool {
	if verifier == "" || challenge == "" {
		return false
	}
	hash := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(hash[:])
	return expected == challenge
}

func randomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func writeJSONResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// listen is a small helper to allow port 0 (OS-assigned).
func listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
