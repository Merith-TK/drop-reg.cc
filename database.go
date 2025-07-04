package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// InitDB initializes the database schema
func (s *Server) initDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS url_mappings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		short_code TEXT UNIQUE NOT NULL,
		discord_url TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME,
		owner_id TEXT NOT NULL
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

// OpenDatabase opens a database connection
func OpenDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

// CreateOrUpdateUser creates or updates a user in the database
func (s *Server) createOrUpdateUser(user *User) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO users (id, username, avatar, discriminator)
		VALUES (?, ?, ?, ?)
	`, user.ID, user.Username, user.Avatar, user.Discriminator)

	return err
}

// GetUserMappings retrieves all URL mappings for a specific user
func (s *Server) getUserMappings(userID string) ([]URLMapping, error) {
	rows, err := s.db.Query(`
		SELECT short_code, discord_url, created_at 
		FROM url_mappings 
		WHERE owner_id = ? AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []URLMapping
	for rows.Next() {
		var mapping URLMapping
		err := rows.Scan(&mapping.ShortCode, &mapping.DiscordURL, &mapping.CreatedAt)
		if err != nil {
			continue
		}
		links = append(links, mapping)
	}

	return links, nil
}

// CreateURLMapping creates a new URL mapping in the database
func (s *Server) createURLMapping(shortCode, discordURL, ownerID string) error {
	_, err := s.db.Exec(
		"INSERT INTO url_mappings (short_code, discord_url, owner_id) VALUES (?, ?, ?)",
		shortCode, discordURL, ownerID,
	)
	return err
}

// GetURLMappingByShortCode retrieves a URL mapping by its short code
func (s *Server) getURLMappingByShortCode(shortCode string) (string, error) {
	var discordURL string
	err := s.db.QueryRow(
		"SELECT discord_url FROM url_mappings WHERE short_code = ? AND (expires_at IS NULL OR expires_at > datetime('now'))",
		shortCode,
	).Scan(&discordURL)
	return discordURL, err
}

// GetURLMappingOwner retrieves the owner ID of a URL mapping
func (s *Server) getURLMappingOwner(shortCode string) (string, error) {
	var ownerID string
	err := s.db.QueryRow(
		"SELECT owner_id FROM url_mappings WHERE short_code = ?",
		shortCode,
	).Scan(&ownerID)
	return ownerID, err
}

// DeleteURLMapping deletes a URL mapping for a specific user
func (s *Server) deleteURLMapping(shortCode, ownerID string) (int64, error) {
	result, err := s.db.Exec(
		"DELETE FROM url_mappings WHERE short_code = ? AND owner_id = ?",
		shortCode, ownerID,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
