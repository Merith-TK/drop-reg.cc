package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	disgoauth "github.com/realTristan/disgoauth"
)

// Session management functions

// CreateSession creates a new session for a user
func (s *Server) createSession(userID string) (string, error) {
	sessionID := s.generateSessionID()
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	_, err := s.db.Exec(
		"INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)",
		sessionID, userID, expiresAt.Format("2006-01-02 15:04:05"),
	)

	return sessionID, err
}

// DeleteSession removes a session from the database
func (s *Server) deleteSession(sessionID string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

// GetUserFromSession retrieves a user by their session ID
func (s *Server) getUserFromSession(sessionID string) (*User, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT u.id, u.username, u.avatar, u.discriminator, u.created_at
		FROM users u
		JOIN sessions s ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > datetime('now')
	`, sessionID).Scan(&user.ID, &user.Username, &user.Avatar, &user.Discriminator, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GenerateSessionID generates a random session ID
func (s *Server) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetCurrentUser retrieves the current authenticated user from the request
func (s *Server) getCurrentUser(r *http.Request) (*User, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, err
	}

	return s.getUserFromSession(cookie.Value)
}

// Authentication handlers

// HandleAuth routes authentication requests
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request, authPath string) {
	switch authPath {
	case "login":
		s.handleLogin(w, r)
	case "callback":
		s.handleCallback(w, r)
	case "logout":
		s.handleLogout(w, r)
	default:
		http.NotFound(w, r)
	}
}

// HandleLogin redirects to Discord OAuth
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	s.discordAuth.RedirectHandler(w, r, "")
}

// HandleCallback processes the OAuth callback from Discord
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Get the authorization code from URL parameters
	codes := r.URL.Query()["code"]
	if len(codes) == 0 {
		s.renderError(w, 400, "Authentication Failed", "No authorization code received", "Please try logging in again.")
		return
	}

	// Exchange code for access token
	accessToken, err := s.discordAuth.GetOnlyAccessToken(codes[0])
	if err != nil {
		s.renderError(w, 500, "Authentication Failed", "Failed to get access token", err.Error())
		return
	}

	// Get user data from Discord
	userData, err := disgoauth.GetUserData(accessToken)
	if err != nil {
		s.renderError(w, 500, "Authentication Failed", "Failed to get user data", err.Error())
		return
	}

	// Create or update user in database
	user := &User{
		ID:            userData["id"].(string),
		Username:      userData["username"].(string),
		Discriminator: userData["discriminator"].(string),
	}

	if avatar, ok := userData["avatar"].(string); ok {
		user.Avatar = avatar
	}

	err = s.createOrUpdateUser(user)
	if err != nil {
		s.renderError(w, 500, "Database Error", "Failed to save user", err.Error())
		return
	}

	// Create session
	sessionID, err := s.createSession(user.ID)
	if err != nil {
		s.renderError(w, 500, "Session Error", "Failed to create session", err.Error())
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
	})

	// Redirect to dashboard (root)
	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleLogout logs out the user and clears their session
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie("session_id")
	if err == nil {
		// Delete session from database
		s.deleteSession(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0), // Expire immediately
	})

	// Redirect to dashboard (root)
	http.Redirect(w, r, "/", http.StatusFound)
}
