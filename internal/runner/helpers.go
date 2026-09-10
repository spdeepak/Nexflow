package runner

import (
	"net/url"
	"strings"

	"google.golang.org/adk/v2/session"
)

// normalizeBaseURL forces loopback endpoints to plain HTTP. Ollama and other
// local model servers do not speak TLS, so a credential saved as
// https://localhost:11434 or https://127.0.0.1:11434 must be dialed over http.
func normalizeBaseURL(raw string) string {
	if raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "https" {
		return raw
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "127.") {
		u.Scheme = "http"
		return u.String()
	}
	return raw
}

// extractText pulls plain text parts out of an ADK event's content.
func extractText(event *session.Event) string {
	if event == nil || event.Content == nil {
		return ""
	}
	var text strings.Builder
	for _, part := range event.Content.Parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			text.WriteString(part.Text)
		}
	}
	return text.String()
}
