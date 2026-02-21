package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates *template.Template

// Rate limiter - simple in-memory implementation
type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	r := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine to prevent memory leak
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			r.cleanup()
		}
	}()

	return r
}

func (r *rateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-r.window * 2)
	for ip, times := range r.requests {
		// Remove IPs with no recent requests
		if len(times) == 0 {
			delete(r.requests, ip)
			continue
		}
		// Check if most recent request is old
		if times[len(times)-1].Before(cutoff) {
			delete(r.requests, ip)
		}
	}
}

func (r *rateLimiter) allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	// Clean old requests
	var recent []time.Time
	for _, t := range r.requests[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= r.limit {
		r.requests[ip] = recent
		return false
	}

	r.requests[ip] = append(recent, now)
	return true
}

var limiter = newRateLimiter(30, time.Minute) // 30 requests per minute per IP

// State signing for CSRF protection
// The proxy signs states it issues so it can verify them on callback
type signedState struct {
	ClientState string `json:"s"`   // Original state from CLI
	RedirectURI string `json:"r"`   // Original redirect_uri
	Timestamp   int64  `json:"t"`   // Unix timestamp for expiration
	Signature   string `json:"sig"` // HMAC signature
}

const stateMaxAge = 10 * time.Minute

func getSigningKey() []byte {
	// Use CLIENT_SECRET as signing key (it's already secret)
	key := os.Getenv("CLIENT_SECRET")
	if key == "" {
		key = "dev-signing-key" // Fallback for local dev
	}
	return []byte(key)
}

func signState(clientState, redirectURI string) string {
	now := time.Now().Unix()
	data := fmt.Sprintf("%s|%s|%d", clientState, redirectURI, now)

	h := hmac.New(sha256.New, getSigningKey())
	h.Write([]byte(data))
	sig := base64.URLEncoding.EncodeToString(h.Sum(nil))

	state := signedState{
		ClientState: clientState,
		RedirectURI: redirectURI,
		Timestamp:   now,
		Signature:   sig,
	}

	stateJSON, _ := json.Marshal(state)
	return base64.URLEncoding.EncodeToString(stateJSON)
}

func verifyState(encodedState string) (*signedState, error) {
	stateJSON, err := base64.URLEncoding.DecodeString(encodedState)
	if err != nil {
		return nil, fmt.Errorf("invalid state encoding")
	}

	var state signedState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, fmt.Errorf("invalid state format")
	}

	// Check expiration
	if time.Since(time.Unix(state.Timestamp, 0)) > stateMaxAge {
		return nil, fmt.Errorf("state expired")
	}

	// Verify signature
	data := fmt.Sprintf("%s|%s|%d", state.ClientState, state.RedirectURI, state.Timestamp)
	h := hmac.New(sha256.New, getSigningKey())
	h.Write([]byte(data))
	expectedSig := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(state.Signature), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid state signature")
	}

	return &state, nil
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (Fly.io sets this)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Fallback to RemoteAddr; handle both IPv4 and bracketed IPv6.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	// If RemoteAddr has no port or is malformed, keep the original value.
	return r.RemoteAddr
}

func main() {
	var err error
	templates, err = template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/auth/start", handleAuthChoice)
	http.HandleFunc("/auth/oauth", handleOAuthStart)
	http.HandleFunc("/auth/callback", handleAuthCallback)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if err := templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleAuthChoice(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	if !limiter.allow(ip) {
		renderError(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
		return
	}

	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		renderError(w, "Server misconfigured: missing CLIENT_ID", http.StatusInternalServerError)
		return
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	// Validate redirect_uri is localhost
	if redirectURI != "" {
		parsed, err := url.Parse(redirectURI)
		if err != nil || !isLocalhostURI(parsed) {
			renderError(w, "Invalid redirect_uri: must be localhost", http.StatusBadRequest)
			return
		}
	}

	// Build OAuth URL for the button
	oauthURL := fmt.Sprintf("/auth/oauth?redirect_uri=%s&state=%s",
		url.QueryEscape(redirectURI),
		url.QueryEscape(state))

	data := map[string]interface{}{
		"OAuthURL":    oauthURL,
		"RedirectURI": redirectURI,
		"State":       state,
	}

	if err := templates.ExecuteTemplate(w, "auth-choice.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	if !limiter.allow(ip) {
		renderError(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
		return
	}

	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		renderError(w, "Server misconfigured: missing CLIENT_ID", http.StatusInternalServerError)
		return
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	// Validate redirect_uri is localhost
	if redirectURI != "" {
		parsed, err := url.Parse(redirectURI)
		if err != nil || !isLocalhostURI(parsed) {
			renderError(w, "Invalid redirect_uri: must be localhost", http.StatusBadRequest)
			return
		}
	}

	// Build Notion OAuth URL
	notionAuthURL := "https://api.notion.com/v1/oauth/authorize"
	params := url.Values{
		"client_id":     {clientID},
		"response_type": {"code"},
		"owner":         {"user"},
		"redirect_uri":  {proxyCallbackURL(r)},
	}

	// Sign the state to prevent tampering
	signedStateParam := signState(state, redirectURI)
	params.Set("state", signedStateParam)

	http.Redirect(w, r, notionAuthURL+"?"+params.Encode(), http.StatusFound)
}

func handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	if !limiter.allow(ip) {
		renderError(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Check for error from Notion
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		if errDesc == "" {
			errDesc = errParam
		}
		renderError(w, "Authorization failed: "+errDesc, http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		renderError(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Verify signed state
	stateParam := r.URL.Query().Get("state")
	stateData, err := verifyState(stateParam)
	if err != nil {
		log.Printf("State verification failed: %v", err)
		renderError(w, "Invalid or expired authorization request. Please try again.", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := exchangeCodeForToken(code, proxyCallbackURL(r))
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		renderError(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// If we have a redirect_uri, use POST to avoid token in URL/history
	if stateData.RedirectURI != "" {
		redirectURL, err := url.Parse(stateData.RedirectURI)
		if err == nil && isLocalhostURI(redirectURL) {
			// Use auto-submitting form to POST token (avoids browser history exposure)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Redirecting...</title></head>
<body>
<form id="f" method="POST" action="%s">
<input type="hidden" name="token" value="%s">
<input type="hidden" name="state" value="%s">
</form>
<script>document.getElementById('f').submit();</script>
<noscript>Click submit to continue: <input type="submit" value="Submit"></noscript>
</body></html>`, template.HTMLEscapeString(redirectURL.String()), template.HTMLEscapeString(token.AccessToken), template.HTMLEscapeString(stateData.ClientState))
			return
		}
	}

	// Fallback: show token page for manual copy
	data := map[string]interface{}{
		"Token": token.AccessToken,
	}
	if err := templates.ExecuteTemplate(w, "token.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

type tokenResponse struct {
	AccessToken   string `json:"access_token"`
	TokenType     string `json:"token_type"`
	BotID         string `json:"bot_id"`
	WorkspaceName string `json:"workspace_name"`
	WorkspaceIcon string `json:"workspace_icon"`
	WorkspaceID   string `json:"workspace_id"`
	Owner         struct {
		Type string `json:"type"`
		User struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
			Type      string `json:"type"`
			Person    struct {
				Email string `json:"email"`
			} `json:"person"`
		} `json:"user"`
	} `json:"owner"`
}

func exchangeCodeForToken(code, redirectURI string) (*tokenResponse, error) {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing CLIENT_ID or CLIENT_SECRET")
	}

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}

	req, err := http.NewRequest("POST", "https://api.notion.com/v1/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Message != "" {
			return nil, fmt.Errorf("%s: %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func isLocalhostURI(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func proxyCallbackURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil && !strings.Contains(r.Host, "fly.dev") {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/auth/callback", scheme, r.Host)
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	data := map[string]interface{}{
		"Error": message,
	}
	if err := templates.ExecuteTemplate(w, "error.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, message, statusCode)
	}
}
