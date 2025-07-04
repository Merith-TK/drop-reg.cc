# Drop-reg.cc Development Summary

**Date:** July 3, 2025  
**Status:** Phase 5 Complete - Backend Modularization  
**Repository:** github.com/Merith-TK/drop-reg.cc

## Project Overview
A Discord invite URL shortener service for Helldivers community servers. Users can register memorable short codes (e.g., `drop-reg.cc/597`) that redirect to Discord server invites.

## Current Implementation Status

### ✅ COMPLETED PHASES

#### Phase 1: Core Redirect Service
- **Pure Go HTTP server** using standard library + SQLite
- **URL validation**: Strict `https://discord.gg/*` only
- **Case-insensitive short codes**: All converted to lowercase
- **Database schema**: SQLite with url_mappings table
- **Static file serving**: CSS/JS assets
- **Error handling**: Custom error pages

#### Phase 2: Modern UI/UX
- **Dark theme**: Professional, eye-friendly design
- **Responsive CSS**: Separated into modular files in `assets/css/`
- **Template system**: Go html/template with 5 pages
- **Registration flow**: Form → Success → Test redirect
- **List view**: Public view of all registered links

#### Phase 3: Discord OAuth Authentication ✅ COMPLETE
- **DisGOAuth integration**: Clean Discord-specific OAuth library
- **Session management**: Secure 32-byte random session IDs
- **User dashboard**: Personal link management interface
- **Database tables**: Added `users` and `sessions` tables
- **Configuration**: TOML config file with Discord credentials
- **Ownership**: Links can be associated with authenticated users

#### Phase 4: Authentication-Only Access & Link Management ✅ COMPLETE
- **Full Authentication Required**: All pages (except redirects) now require Discord login
- **User-Only Access**: Users can only see and manage their own links
- **Link Deletion**: Added ability for users to delete their own shortlinks
- **Redirect URI Auto-Configuration**: Automatically builds redirect URI from domain config

#### Phase 5: Backend Modularization ✅ COMPLETE
- **Code Organization**: Split monolithic `main.go` into logical modules
- **Maintainable Structure**: Clear separation of concerns across files
- **Module Breakdown**:
  - `main.go`: Entry point and application startup (25 lines)
  - `config.go`: Configuration management and helper functions
  - `types.go`: All type definitions (Server, User, Session, URLMapping, Config)
  - `database.go`: Database operations and schema initialization
  - `auth.go`: Authentication handlers and session management
  - `handlers.go`: HTTP request handlers for all routes
  - `server.go`: Server initialization and HTTP routing
  - `utils.go`: Shared utilities (Discord URL validation regex)

### Code Cleanup & Simplification
- **Removed redundant `/list` route**: Dashboard now serves as the single location for viewing user links
- **Streamlined navigation**: Removed duplicate "View Your Links" button from dashboard
- **File cleanup**: Removed `list.html` template and `list.css` since functionality is consolidated in dashboard
- **CSS consolidation**: Moved necessary table styles directly into dashboard template
- **Simplified UI flow**: Users now have a cleaner, more focused experience with fewer redundant pages

### Route Reorganization & UI Flow Improvement
- **Dashboard as Home**: `/` now serves as the main dashboard (user's link management interface)
- **Dedicated Registration**: `/register` is now a dedicated page for creating new shortlinks
- **Improved Navigation**: Clear separation between viewing links (dashboard) and creating links (register page)
- **Template Updates**: Updated all navigation links and redirects to match new route structure
- **User Experience**: More intuitive flow - login takes you to dashboard, register takes you to creation form
- **Consistent Redirects**: All auth flows and operations redirect to appropriate pages

### Template File Reorganization
- **Dashboard → index.html**: Main dashboard moved to index.html (served at `/`)
- **New register.html**: Created dedicated registration template (served at `/register`)
- **Removed dashboard.html**: Consolidated dashboard functionality into index.html
- **Template Updates**: Updated Go handlers to use correct template files
- **File Structure**: Cleaner template organization with logical naming

### CSS Modularization & Code Organization
- **Modular CSS Structure**: Split inline styles into reusable CSS modules
  - `base.css` - Core styles, typography, buttons, basic forms
  - `layout.css` - Layout components (user info, headers, containers, info boxes)
  - `forms.css` - Advanced form styling (register page, input prefixes, submit buttons)
  - `tables.css` - Table styling for dashboard links display
  - `error.css` - Error page specific styles
  - `success.css` - Success page specific styles

- **Template Cleanup**: Removed all inline `<style>` blocks from HTML templates
- **File Consolidation**: 
  - Removed `index.css` (consolidated into modular system)
  - Removed `dashboard.html` (functionality moved to `index.html`)
  - Updated all templates to use modular CSS imports

- **Improved Maintainability**: 
  - Shared styles are now reusable across pages
  - Easier to update consistent styling
  - Better separation of concerns
  - Reduced code duplication

- **CSS Loading Strategy**: Each page only loads the CSS modules it needs

## File Structure
```
drop-reg.cc/
├── main.go                 # Main server with all handlers
├── config.toml            # Discord OAuth credentials
├── go.mod / go.sum        # Dependencies
├── drop-reg.exe           # Built executable
├── drop-reg.db            # SQLite database (auto-created)
├── plan.md                # Project planning document
└── assets/
    ├── css/
    │   ├── base.css       # Core dark theme styles
    │   ├── layout.css     # Layout components
    │   ├── forms.css      # Advanced form styling
    │   ├── tables.css     # Table styling
    │   ├── error.css      # Error page specific styles
    │   └── success.css    # Success page specific styles
    ├── index.html         # Home page with login/logout
    ├── success.html       # Registration success page
    ├── register.html      # Dedicated registration page
    └── error.html         # Error page template
```

## Dependencies
```go
// Current go.mod dependencies:
- modernc.org/sqlite v1.38.0          // Pure Go SQLite
- github.com/realTristan/disgoauth     // Discord OAuth
- github.com/BurntSushi/toml v1.5.0    // Config file parsing
```

## Database Schema
```sql
-- URL mappings (core functionality)
CREATE TABLE url_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT UNIQUE NOT NULL,    -- lowercase only
    discord_url TEXT NOT NULL,          -- https://discord.gg/* only
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,                -- optional expiration
    owner_id TEXT NOT NULL               -- links to users.id
);

-- User accounts (Discord OAuth)
CREATE TABLE users (
    id TEXT PRIMARY KEY,                -- Discord user ID
    username TEXT NOT NULL,
    avatar TEXT,                        -- Discord avatar hash
    discriminator TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Session management
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,                -- 32-byte random hex
    user_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,       -- 30 day expiration
    FOREIGN KEY (user_id) REFERENCES users (id)
);
```

## Key Features Implemented

### 🔐 Authentication System
- **Discord OAuth flow**: `/auth/login` → Discord → `/auth/callback`
- **Session cookies**: HttpOnly, 30-day expiration
- **User dashboard**: `/dashboard` (auth required)
- **Logout**: `/auth/logout` clears session

### 🎨 Modern Dark Theme
- **Professional design**: Easy on eyes, responsive
- **Modular CSS**: Easy to customize/theme
- **User-aware UI**: Different experience for logged-in users
- **Discord integration**: Shows user avatar and username

### 🔗 URL Management
- **Anonymous registration**: Anyone can create links
- **Authenticated ownership**: Logged-in users own their links
- **Public listing**: `/list` shows all links
- **Personal dashboard**: Users see only their links
- **Fast redirects**: Direct SQLite lookup
- **Link deletion**: Users can delete their own links

### 🛡️ Security Features
- **URL validation**: Only Discord links allowed
- **Session security**: Cryptographically random IDs
- **Input sanitization**: Lowercase short codes
- **Error handling**: User-friendly error pages

## Configuration Setup
The `config.toml` file contains Discord OAuth credentials:
```toml
[client]
id = "DISCORD_CLIENT_ID"
secret = "DISCORD_CLIENT_SECRET"

[server]
domain = "drop-reg.cc"  # Auto-generates https://drop-reg.cc/auth/callback
# OR
domain = "localhost:8080"  # Auto-generates http://localhost:8080/auth/callback
```

## Build & Run
```bash
go build -o drop-reg.exe    # Build executable
./drop-reg.exe              # Run server on :8080
```

## Current Routes
- `GET /` - Home page (shows login status)
- `POST /register` - Create new short link
- `GET /list` - Public listing of all links
- `GET /dashboard` - User dashboard (auth required)
- `GET /auth/login` - Redirect to Discord OAuth
- `GET /auth/callback` - OAuth callback handler
- `GET /auth/logout` - Clear session and logout
- `GET /{shortcode}` - Redirect to Discord invite
- `GET /assets/*` - Static file serving
- `POST /delete` - Delete a short link (auth required)

## Next Steps for Future Development
1. **Rate limiting**: Prevent abuse
2. **Link editing**: Allow users to update their URLs
3. **Subdomain routing**: Transition from subpath to subdomain
4. **Caddy configuration**: Production reverse proxy setup
5. **Monitoring**: Basic health checks and logs

## Important Notes
- **Windows firewall**: Using built executable avoids repeated permission prompts
- **Pure Go**: No CGO dependencies, easy deployment
- **SQLite**: Single file database, no external DB required
- **Production ready**: Just needs Discord app setup and domain configuration

The application is fully functional with authentication and ready for production deployment.
