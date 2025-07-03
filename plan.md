# Drop-reg.cc Project Plan

## Project Overview
A web service for Helldivers community server owners to create easy-to-share Discord invite links using the drop-reg.cc domain.

**Created:** July 3, 2025  
**Status:** Phase 3 - Authentication & Security Implementation

## Project Purpose
- Allow server owners to link Discord invites to memorable subdomains (e.g., `597.drop-reg.cc`) or subpaths (e.g., `drop-reg.cc/597`)
- Provide easy sharing in game chats with short, memorable URLs
- Prevent abuse through proper authentication and validation

## Core Requirements

### Functional Requirements
1. **URL Registration System**
   - Server owners can register custom short codes (e.g., "597")
   - Support both subdomain and subpath routing
   - Validate Discord invite URLs
   - Handle URL expiration/renewal

2. **Authentication & Authorization**
   - User registration/login system
   - Prevent abuse and spam
   - Rate limiting
   - Ownership verification for registered URLs

3. **Redirection Service**
   - Fast, reliable redirects to Discord invites
   - Analytics/usage tracking (optional)
   - Handle invalid/expired links gracefully

4. **Management Interface**
   - Web dashboard for users to manage their registrations
   - Edit/update Discord invite URLs
   - View usage statistics

### Technical Challenges
- Frontend development (registration UI, dashboard)
- Data storage design (users, registrations, analytics)
- Authentication system implementation
- DNS/subdomain routing configuration
- Abuse prevention mechanisms

## Current State
- Repository initialized with Git
- No existing source files
- Requirements defined

## Development Phases

### Phase 1: Core Redirect Service (No Auth) ✅ COMPLETE
1. ✅ Define project requirements and scope
2. ✅ Determine technology stack and architecture
3. ✅ Design database schema for URL mappings
4. ✅ Set up Go development environment
5. ✅ Create basic redirect service (subpath routing only)
6. ✅ Implement Discord URL validation (`https://discord.gg/*` only)
7. ✅ Configure static file serving
8. ✅ Test redirect functionality with lowercase short codes

### Phase 2: Management Interface (Still No Auth) ✅ COMPLETE
9. ✅ Design modern dark theme web interface
10. ✅ Implement frontend with Go templating and separated CSS
11. ✅ Add registration form and success pages
12. ✅ Create list view for all registered links
13. ✅ Add comprehensive error handling with custom error pages
14. ✅ Test full registration → redirect flow

### Phase 3: Authentication & Security (IN PROGRESS)
15. Implement Discord OAuth integration
16. Create user authentication system
17. Add user accounts and session management
18. Implement ownership verification for registered URLs
19. Add user dashboard for managing personal registrations
20. Implement rate limiting and abuse prevention
21. Add login/logout functionality
22. Production deployment preparation

## Technology Stack (DECIDED)

### Backend
- **Language:** Go (Golang) - chosen for reliability and deployment flexibility
- **Framework:** TBD (Gin, Echo, or Fiber - all good options for APIs)
- **Database:** SQLite - simple, lightweight, perfect for this use case

### Frontend
- **Approach:** Go-based web GUI library (exploring options like Templ, html/template, or HTMX integration)
- **Styling:** Minimal CSS framework (TailwindCSS or similar)

### Authentication
- **Method:** Discord OAuth (to be implemented in final phase)
- **Note:** Authentication and management features will be the last development step

### Infrastructure
- **Hosting:** Self-hosted server
- **Reverse Proxy:** Caddy (with wildcard subdomain support)
- **DNS:** Cloudflare
  - A record: `drop-reg.cc` → server IP
  - CNAME: `*.drop-reg.cc` → `drop-reg.cc`
- **SSL:** Cloudflare wildcard certificate or self-managed cert

### Deployment Architecture
```
Internet → Cloudflare DNS → Caddy (Reverse Proxy) → Go Application
                ↓
            SQLite Database
```

## Technical Specifications

### URL Routing Strategy
- **Phase 1:** Subpath routing (`drop-reg.cc/597`)
- **Future:** Transition to subdomain routing (`597.drop-reg.cc`)
- **Reason:** Simpler implementation, easier Caddy configuration

### Data Policy
- **Analytics:** No click tracking or user analytics
- **Data Retention:** Minimal - only store necessary mapping data
- **Privacy:** Following existing TOS - no retention of unnecessary data

### Security Constraints
- **Short Codes:** Case-insensitive (all converted to lowercase)
- **URL Validation:** STRICT - only allow `https://discord.gg/*` links
- **Redirect Enforcement:** Server-side validation before any redirect

### Database Schema (Preliminary)
```sql
CREATE TABLE url_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT UNIQUE NOT NULL, -- always lowercase
    discord_url TEXT NOT NULL,       -- must match https://discord.gg/*
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,             -- optional expiration
    owner_id TEXT                    -- for future auth phase
);
```

## Revisions
- **v1.0** (July 3, 2025): Initial plan creation
- **v1.1** (July 3, 2025): Added project requirements and scope definition for Helldivers Discord invite service
- **v1.2** (July 3, 2025): Technology stack decisions and development phases defined
- **v1.3** (July 3, 2025): Technical specifications added - subpath routing, no analytics, Discord URL validation only
- **v1.4** (July 3, 2025): Phase 1 & 2 completed - Core functionality and modern dark theme UI implemented. Starting Phase 3 authentication.

---

*This plan will be updated as we discuss and refine the project requirements.*
