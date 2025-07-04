package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	disgoauth "github.com/realTristan/disgoauth"
)

// InitServer initializes a new server instance with all dependencies
func InitServer(dbPath string, config *Config) (*Server, error) {
	// Open database connection
	db, err := OpenDatabase(dbPath)
	if err != nil {
		return nil, err
	}

	// Load templates
	templates, err := template.ParseGlob("assets/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Initialize Discord OAuth client
	redirectURI := config.GetRedirectURI()

	discordAuth := disgoauth.Init(&disgoauth.Client{
		ClientID:     config.Client.ID,
		ClientSecret: config.Client.Secret,
		RedirectURI:  redirectURI,
		Scopes:       []string{disgoauth.ScopeIdentify}, // identify scope provides: id, username, avatar, discriminator
	})

	server := &Server{
		db:          db,
		templates:   templates,
		discordAuth: discordAuth,
	}

	// Initialize database schema
	if err := server.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return server, nil
}

// ServeHTTP implements http.Handler for routing
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	// Handle root path (dashboard)
	if path == "" {
		s.handleDashboard(w, r)
		return
	}

	// Handle registration page
	if path == "register" {
		s.handleRegisterPage(w, r)
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

	// Handle delete (requires auth)
	if path == "delete" {
		s.handleDelete(w, r)
		return
	}

	// Handle short code redirect
	s.handleRedirect(w, r, path)
}
