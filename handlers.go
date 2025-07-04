package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandleRegisterPage handles the registration page (GET and POST)
func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated - redirect to login if not
	user, err := s.getCurrentUser(r)
	if err != nil {
		// Not authenticated, redirect to login
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Handle GET request - show registration form
	if r.Method == http.MethodGet {
		data := struct {
			User *User
		}{
			User: user,
		}

		w.Header().Set("Content-Type", "text/html")
		err = s.templates.ExecuteTemplate(w, "register.html", data)
		if err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			log.Printf("Template error: %v", err)
		}
		return
	}

	// Handle POST request - process registration
	if r.Method == http.MethodPost {
		s.handleRegisterSubmit(w, r, user)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleRegisterSubmit processes the registration form submission
func (s *Server) handleRegisterSubmit(w http.ResponseWriter, r *http.Request, user *User) {
	shortCode := strings.ToLower(strings.TrimSpace(r.FormValue("short_code")))
	discordURL := strings.TrimSpace(r.FormValue("discord_url"))

	// Validate inputs
	if shortCode == "" {
		http.Error(w, "Short code is required", http.StatusBadRequest)
		return
	}

	if !DiscordURLRegex.MatchString(discordURL) {
		http.Error(w, "Invalid Discord URL. Must be https://discord.gg/...", http.StatusBadRequest)
		return
	}

	// Create URL mapping
	err := s.createURLMapping(shortCode, discordURL, user.ID)
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

// HandleRedirect handles shortcode redirects to Discord URLs
func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request, shortCode string) {
	// Convert to lowercase for lookup
	shortCode = strings.ToLower(shortCode)

	discordURL, err := s.getURLMappingByShortCode(shortCode)
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

// HandleStatic serves static assets
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

// HandleDashboard displays the user's dashboard with their registered links
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	user, err := s.getCurrentUser(r)
	if err != nil {
		// Not authenticated, redirect to login
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Get user's registered URLs
	links, err := s.getUserMappings(user.ID)
	if err != nil {
		s.renderError(w, 500, "Database Error", "Failed to retrieve your links", err.Error())
		return
	}

	// Format creation times
	for i := range links {
		if t, err := time.Parse("2006-01-02 15:04:05", links[i].CreatedAt); err == nil {
			links[i].CreatedAt = t.Format("Jan 2, 2006 15:04")
		}
	}

	data := struct {
		User  *User
		Links []URLMapping
	}{
		User:  user,
		Links: links,
	}

	w.Header().Set("Content-Type", "text/html")
	err = s.templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

// HandleDelete deletes a user's shortlink
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	user, err := s.getCurrentUser(r)
	if err != nil {
		// Not authenticated, redirect to login
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortCode := strings.ToLower(strings.TrimSpace(r.FormValue("short_code")))
	if shortCode == "" {
		http.Error(w, "Short code is required", http.StatusBadRequest)
		return
	}

	// First check if the link exists and belongs to the user
	existingOwnerID, err := s.getURLMappingOwner(shortCode)
	if err == sql.ErrNoRows {
		s.renderError(w, 404, "Link Not Found",
			fmt.Sprintf("The short code '%s' was not found.", shortCode),
			"The link may have already been deleted.")
		return
	}

	if err != nil {
		s.renderError(w, 500, "Database Error", "Failed to check link ownership", err.Error())
		return
	}

	// Check if the user owns this link
	if existingOwnerID != user.ID {
		s.renderError(w, 403, "Access Denied",
			"You can only delete links that you created.",
			fmt.Sprintf("The link '%s' belongs to another user.", shortCode))
		return
	}

	// Delete the link
	rowsAffected, err := s.deleteURLMapping(shortCode, user.ID)
	if err != nil {
		s.renderError(w, 500, "Database Error", "Failed to delete link", err.Error())
		return
	}

	if rowsAffected == 0 {
		s.renderError(w, 404, "Delete Failed",
			"No link was deleted. It may have already been removed.",
			"Please check your dashboard for current links.")
		return
	}

	// Success - redirect back to dashboard (root)
	http.Redirect(w, r, "/", http.StatusFound)
}

// RenderError displays an error page
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
