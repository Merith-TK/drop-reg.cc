package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	disgoauth "github.com/realTristan/disgoauth"
	_ "modernc.org/sqlite"
)

type Server struct {
	db           *sql.DB
	templates    *template.Template
	discordAuth  *disgoauth.Client
}

// Config represents the application configuration
type Config struct {
	Client struct {
		ID     string `toml:"id"`
		Secret string `toml:"secret"`
	} `toml:"client"`
	Server struct {
		Port         string `toml:"port"`
		DatabasePath string `toml:"database_path"`
		RedirectURI  string `toml:"redirect_uri"`
	} `toml:"server"`
}

// User represents a Discord user
type User struct {
	ID           string
	Username     string
	Avatar       string
	Discriminator string
	CreatedAt    string
}

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	ExpiresAt string
}

// URLMapping represents a database record
type URLMapping struct {
	ID         int
	ShortCode  string
	DiscordURL string
	CreatedAt  string
	ExpiresAt  *string
	OwnerID    *string
}

// Discord URL validation regex
var discordURLRegex = regexp.MustCompile(`^https://discord\.gg/[a-zA-Z0-9]+$`)

func main() {
	// Load configuration
	var config Config
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	server, err := NewServer("drop-reg.db", &config)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer server.db.Close()

	port := config.Server.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting drop-reg.cc server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, server))
}

func NewServer(dbPath string, config *Config) (*Server, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Load templates
	templates, err := template.ParseGlob("assets/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Initialize Discord OAuth client
	redirectURI := config.Server.RedirectURI
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/auth/callback"
	}
	
	discordAuth := disgoauth.Init(&disgoauth.Client{
		ClientID:     config.Client.ID,
		ClientSecret: config.Client.Secret,
		RedirectURI:  redirectURI,
		Scopes:       []string{disgoauth.ScopeIdentify},
	})

	server := &Server{
		db:          db,
		templates:   templates,
		discordAuth: discordAuth,
	}

	if err := server.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return server, nil
}

func (s *Server) initDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS url_mappings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		short_code TEXT UNIQUE NOT NULL,
		discord_url TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME,
		owner_id TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_short_code ON url_mappings(short_code);
	
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		avatar TEXT,
		discriminator TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users (id)
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
	`

	_, err := s.db.Exec(query)
	return err
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	// Handle root path
	if path == "" {
		s.handleRoot(w, r)
		return
	}

	// Handle registration
	if path == "register" {
		s.handleRegister(w, r)
		return
	}

	// Handle list view
	if path == "list" {
		s.handleList(w, r)
		return
	}

	// Handle static assets
	if strings.HasPrefix(path, "assets/") {
		s.handleStatic(w, r)
		return
	}
	
	// Handle authentication routes
	if strings.HasPrefix(path, "auth/") {
		s.handleAuth(w, r, strings.TrimPrefix(path, "auth/"))
		return
	}
	
	// Handle dashboard (requires auth)
	if path == "dashboard" {
		s.handleDashboard(w, r)
		return
	}

	// Handle short code redirect
	s.handleRedirect(w, r, path)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	user, _ := s.getCurrentUser(r)
	
	data := struct {
		User *User
	}{
		User: user,
	}
	
	w.Header().Set("Content-Type", "text/html")
	err := s.templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request, shortCode string) {
	// Convert to lowercase for lookup
	shortCode = strings.ToLower(shortCode)

	var discordURL string
	err := s.db.QueryRow(
		"SELECT discord_url FROM url_mappings WHERE short_code = ? AND (expires_at IS NULL OR expires_at > datetime('now'))",
		shortCode,
	).Scan(&discordURL)

	if err == sql.ErrNoRows {
		s.renderError(w, 404, "Short Link Not Found",
			fmt.Sprintf("The short code '%s' was not found.", shortCode),
			"Please check the link or register a new one.")
		return
	}

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("Database error: %v", err)
		return
	}

	// Redirect to Discord
	http.Redirect(w, r, discordURL, http.StatusFound)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortCode := strings.ToLower(strings.TrimSpace(r.FormValue("short_code")))
	discordURL := strings.TrimSpace(r.FormValue("discord_url"))

	// Validate inputs
	if shortCode == "" {
		http.Error(w, "Short code is required", http.StatusBadRequest)
		return
	}

	if !discordURLRegex.MatchString(discordURL) {
		http.Error(w, "Invalid Discord URL. Must be https://discord.gg/...", http.StatusBadRequest)
		return
	}

	// Get current user (optional for now, but will be required later)
	var ownerID *string
	if user, err := s.getCurrentUser(r); err == nil {
		ownerID = &user.ID
	}

	// Insert into database
	_, err := s.db.Exec(
		"INSERT INTO url_mappings (short_code, discord_url, owner_id) VALUES (?, ?, ?)",
		shortCode, discordURL, ownerID,
	)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			http.Error(w, "Short code already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to register URL", http.StatusInternalServerError)
		log.Printf("Database error: %v", err)
		return
	}

	// Success response
	w.Header().Set("Content-Type", "text/html")
	data := struct {
		ShortCode  string
		DiscordURL string
	}{
		ShortCode:  shortCode,
		DiscordURL: discordURL,
	}

	err = s.templates.ExecuteTemplate(w, "success.html", data)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT short_code, discord_url, created_at 
		FROM url_mappings 
		WHERE expires_at IS NULL OR expires_at > datetime('now')
		ORDER BY created_at DESC
	`)
	if err != nil {
		s.renderError(w, 500, "Database Error", "Failed to retrieve links", err.Error())
		return
	}
	defer rows.Close()

	var links []URLMapping
	for rows.Next() {
		var mapping URLMapping
		err := rows.Scan(&mapping.ShortCode, &mapping.DiscordURL, &mapping.CreatedAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Parse and format the created_at time
		if t, err := time.Parse("2006-01-02 15:04:05", mapping.CreatedAt); err == nil {
			mapping.CreatedAt = t.Format("Jan 2, 2006 15:04")
		}

		links = append(links, mapping)
	}

	data := struct {
		Links []URLMapping
	}{
		Links: links,
	}

	w.Header().Set("Content-Type", "text/html")
	err = s.templates.ExecuteTemplate(w, "list.html", data)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Remove leading slash and serve from assets directory
	filePath := strings.TrimPrefix(r.URL.Path, "/")

	// Security: ensure we can only serve files from assets directory
	if !strings.HasPrefix(filePath, "assets/") {
		http.NotFound(w, r)
		return
	}

	// Set appropriate content type for CSS files
	if strings.HasSuffix(filePath, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}

	// Serve the static file
	http.ServeFile(w, r, filePath)
}

func (s *Server) renderError(w http.ResponseWriter, statusCode int, title, message, details string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/html")

	data := struct {
		StatusCode int
		Title      string
		Message    string
		Details    string
	}{
		StatusCode: statusCode,
		Title:      title,
		Message:    message,
		Details:    details,
	}

	err := s.templates.ExecuteTemplate(w, "error.html", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

// Authentication handlers
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

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Redirect to Discord OAuth
	s.discordAuth.RedirectHandler(w, r, "")
}

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
		ID:           userData["id"].(string),
		Username:     userData["username"].(string),
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

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

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

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusFound)
}

// Session management
func (s *Server) createSession(userID string) (string, error) {
	sessionID := s.generateSessionID()
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	_, err := s.db.Exec(
		"INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)",
		sessionID, userID, expiresAt.Format("2006-01-02 15:04:05"),
	)
	
	return sessionID, err
}

func (s *Server) deleteSession(sessionID string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

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

func (s *Server) createOrUpdateUser(user *User) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO users (id, username, avatar, discriminator)
		VALUES (?, ?, ?, ?)
	`, user.ID, user.Username, user.Avatar, user.Discriminator)
	
	return err
}

func (s *Server) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *Server) getCurrentUser(r *http.Request) (*User, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, err
	}
	
	return s.getUserFromSession(cookie.Value)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	user, err := s.getCurrentUser(r)
	if err != nil {
		// Not authenticated, redirect to login
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Get user's registered URLs
	rows, err := s.db.Query(`
		SELECT short_code, discord_url, created_at 
		FROM url_mappings 
		WHERE owner_id = ? AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY created_at DESC
	`, user.ID)
	if err != nil {
		s.renderError(w, 500, "Database Error", "Failed to retrieve your links", err.Error())
		return
	}
	defer rows.Close()

	var links []URLMapping
	for rows.Next() {
		var mapping URLMapping
		err := rows.Scan(&mapping.ShortCode, &mapping.DiscordURL, &mapping.CreatedAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		// Parse and format the created_at time
		if t, err := time.Parse("2006-01-02 15:04:05", mapping.CreatedAt); err == nil {
			mapping.CreatedAt = t.Format("Jan 2, 2006 15:04")
		}
		
		links = append(links, mapping)
	}

	data := struct {
		User  *User
		Links []URLMapping
	}{
		User:  user,
		Links: links,
	}

	w.Header().Set("Content-Type", "text/html")
	err = s.templates.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}
