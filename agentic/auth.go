// Package agentic provides FOSS GitHub OAuth authentication for the agentic orchestration system.
package agentic

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/xerrors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHubUser represents a GitHub user from the API.
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// AuthSession represents an authenticated user session.
type AuthSession struct {
	UserID       string    `json:"user_id"`
	Login        string    `json:"login"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	AvatarURL    string    `json:"avatar_url"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expires      time.Time `json:"expires"`
	CreatedAt    time.Time `json:"created_at"`
}

// GitHubAuthConfig holds GitHub OAuth configuration.
type GitHubAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// GitHubAuthProvider implements GitHub OAuth2 authentication using FOSS libraries.
type GitHubAuthProvider struct {
	config      *oauth2.Config
	secretStore SecretStore
	sessions    map[string]*AuthSession // In-memory session store (production should use Redis/DB)
	logger      func(string, ...interface{})
}

// NewGitHubAuthProvider creates a new GitHub OAuth provider.
func NewGitHubAuthProvider(cfg GitHubAuthConfig, secretStore SecretStore) *GitHubAuthProvider {
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"user:email", "read:user"}
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
		Endpoint:     github.Endpoint,
	}

	return &GitHubAuthProvider{
		config:      oauthConfig,
		secretStore: secretStore,
		sessions:    make(map[string]*AuthSession),
	}
}

// SetLogger sets a logger function for the auth provider.
func (g *GitHubAuthProvider) SetLogger(logger func(string, ...interface{})) {
	g.logger = logger
}

// GenerateAuthURL generates a GitHub OAuth authorization URL with state parameter.
func (g *GitHubAuthProvider) GenerateAuthURL(state string) (string, error) {
	if state == "" {
		// Generate a random state if none provided
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return "", xerrors.Errorf("failed to generate state: %w", err)
		}
		state = base64.URLEncoding.EncodeToString(b)
	}

	url := g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	if g.logger != nil {
		g.logger("Generated GitHub auth URL with state: %s", state)
	}

	return url, nil
}

// HandleCallback handles the OAuth callback and exchanges code for token.
func (g *GitHubAuthProvider) HandleCallback(ctx context.Context, code, state string) (*AuthSession, error) {
	if g.logger != nil {
		g.logger("Handling GitHub OAuth callback with code and state")
	}

	// Exchange authorization code for token
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, xerrors.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user information from GitHub API
	user, err := g.fetchGitHubUser(ctx, token.AccessToken)
	if err != nil {
		return nil, xerrors.Errorf("failed to fetch user info: %w", err)
	}

	// Create session
	session := &AuthSession{
		UserID:       fmt.Sprintf("%d", user.ID),
		Login:        user.Login,
		Email:        user.Email,
		Name:         user.Name,
		AvatarURL:    user.AvatarURL,
		Token:        token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expires:      token.Expiry,
		CreatedAt:    time.Now(),
	}

	// Store session (in production, use persistent storage)
	sessionID := g.generateSessionID()
	g.sessions[sessionID] = session

	// Optionally store user token securely
	if g.secretStore != nil {
		tokenKey := fmt.Sprintf("github_token_%s", user.Login)
		if err := g.secretStore.Set(tokenKey, token.AccessToken); err != nil {
			if g.logger != nil {
				g.logger("Warning: Failed to store GitHub token for user %s: %v", user.Login, err)
			}
		}
	}

	if g.logger != nil {
		g.logger("Successfully authenticated GitHub user: %s (%s)", user.Login, user.Email)
	}

	return session, nil
}

// fetchGitHubUser fetches user information from GitHub API.
func (g *GitHubAuthProvider) fetchGitHubUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	// Fetch user email if not included in user object
	if user.Email == "" {
		email, err := g.fetchGitHubUserEmail(ctx, accessToken)
		if err == nil {
			user.Email = email
		}
	}

	return &user, nil
}

// fetchGitHubUserEmail fetches the primary email from GitHub API.
func (g *GitHubAuthProvider) fetchGitHubUserEmail(ctx context.Context, accessToken string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", xerrors.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary {
			return email.Email, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", xerrors.New("no email found")
}

// generateSessionID generates a secure session ID.
func (g *GitHubAuthProvider) generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GetSession retrieves a session by ID.
func (g *GitHubAuthProvider) GetSession(sessionID string) (*AuthSession, bool) {
	session, exists := g.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(session.Expires) {
		delete(g.sessions, sessionID)
		return nil, false
	}

	return session, true
}

// RevokeSession removes a session.
func (g *GitHubAuthProvider) RevokeSession(sessionID string) {
	delete(g.sessions, sessionID)
}

// RefreshToken refreshes an expired access token using the refresh token.
func (g *GitHubAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := g.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, xerrors.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// AuthMiddleware creates HTTP middleware for authentication.
func (g *GitHubAuthProvider) AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract session from cookie or header
			sessionID := r.Header.Get("X-Session-ID")
			if sessionID == "" {
				if cookie, err := r.Cookie("session_id"); err == nil {
					sessionID = cookie.Value
				}
			}

			if sessionID == "" {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			session, exists := g.GetSession(sessionID)
			if !exists {
				http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
				return
			}

			// Add session to request context
			ctx := context.WithValue(r.Context(), "session", session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext extracts the authenticated user from request context.
func GetUserFromContext(ctx context.Context) (*AuthSession, bool) {
	session, ok := ctx.Value("session").(*AuthSession)
	return session, ok
}

// AuthHandler creates HTTP handlers for OAuth flow.
type AuthHandler struct {
	provider *GitHubAuthProvider
	logger   func(string, ...interface{})
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(provider *GitHubAuthProvider) *AuthHandler {
	return &AuthHandler{
		provider: provider,
	}
}

// SetLogger sets a logger function.
func (ah *AuthHandler) SetLogger(logger func(string, ...interface{})) {
	ah.logger = logger
}

// LoginHandler handles the OAuth login initiation.
func (ah *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")

	authURL, err := ah.provider.GenerateAuthURL(state)
	if err != nil {
		http.Error(w, "Failed to generate auth URL", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the OAuth callback.
func (ah *AuthHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Authorization code not provided", http.StatusBadRequest)
		return
	}

	session, err := ah.provider.HandleCallback(r.Context(), code, state)
	if err != nil {
		if ah.logger != nil {
			ah.logger("OAuth callback error: %v", err)
		}
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	sessionID := ""
	for id, sess := range ah.provider.sessions {
		if sess == session {
			sessionID = id
			break
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.Expires,
	})

	// Return user info as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":         session.UserID,
			"login":      session.Login,
			"email":      session.Email,
			"name":       session.Name,
			"avatar_url": session.AvatarURL,
		},
	})
}

// LogoutHandler handles user logout.
func (ah *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		if cookie, err := r.Cookie("session_id"); err == nil {
			sessionID = cookie.Value
		}
	}

	if sessionID != "" {
		ah.provider.RevokeSession(sessionID)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}
