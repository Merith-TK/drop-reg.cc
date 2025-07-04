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
		config:      config,
	}

	// Initialize database schema
	if err := server.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return server, nil
}

// getBaseDomain returns the base domain from the Host header
func (s *Server) getBaseDomain(host string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	
	// Split by dots
	parts := strings.Split(host, ".")
	
	// For localhost or IP addresses, return as-is
	if len(parts) < 2 || strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") {
		return host
	}
	
	// For subdomains, get the last 2 parts (domain.tld)
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	
	// Fallback to config domain if available
	if s.config != nil && s.config.Server.Domain != "" {
		return s.config.Server.Domain
	}
	
	return host
}

// extractSubdomain extracts the subdomain from the Host header
// Returns empty string if no subdomain or if subdomain is www
func (s *Server) extractSubdomain(host string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Split by dots
	parts := strings.Split(host, ".")

	// Need at least 3 parts for a subdomain (subdomain.domain.tld)
	if len(parts) < 3 {
		return ""
	}

	// Get the first part (subdomain)
	subdomain := parts[0]

	// Ignore www subdomain
	if subdomain == "www" {
		return ""
	}

	// Convert to lowercase for consistency
	return strings.ToLower(subdomain)
}

// ServeHTTP implements http.Handler for routing
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	// Extract subdomain from Host header
	subdomain := s.extractSubdomain(r.Host)

	// If we have a subdomain, treat it as a shortcode redirect
	if subdomain != "" {
		s.handleRedirect(w, r, subdomain)
		return
	}

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

	// If no subdomain and path doesn't match any route, show 404
	http.NotFound(w, r)
}
