package main

import (
	"database/sql"
	"html/template"

	disgoauth "github.com/realTristan/disgoauth"
)

// Config represents the application configuration
type Config struct {
	Client struct {
		ID     string `toml:"id"`
		Secret string `toml:"secret"`
	} `toml:"client"`
	Server struct {
		Domain       string `toml:"domain"`
		Port         int64  `toml:"port"`
		DatabasePath string `toml:"database_path"`
		RedirectURI  string `toml:"redirect_uri"`
	} `toml:"server"`
}

// User represents a Discord user
type User struct {
	ID            string
	Username      string
	Avatar        string
	Discriminator string
	CreatedAt     string
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

// Server holds the application state
type Server struct {
	db          *sql.DB
	templates   *template.Template
	discordAuth *disgoauth.Client
	config      *Config
}
