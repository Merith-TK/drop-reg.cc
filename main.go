package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Server struct {
	db        *sql.DB
	templates *template.Template
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
	server, err := NewServer("drop-reg.db")
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer server.db.Close()

	log.Println("Starting drop-reg.cc server on :8080")
	log.Fatal(http.ListenAndServe(":8080", server))
}

func NewServer(dbPath string) (*Server, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Load templates
	templates, err := template.ParseGlob("assets/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	server := &Server{
		db:        db,
		templates: templates,
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
	
	// Handle short code redirect
	s.handleRedirect(w, r, path)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	err := s.templates.ExecuteTemplate(w, "index.html", nil)
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
	
	// Insert into database
	_, err := s.db.Exec(
		"INSERT INTO url_mappings (short_code, discord_url) VALUES (?, ?)",
		shortCode, discordURL,
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
