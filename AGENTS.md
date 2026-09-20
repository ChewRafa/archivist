# AGENTS.md — Archivist

## Project
Go 1.25.5 web app for TTRPG character/guild tracking. Single module, no monorepo.

## Commands
- `go run .` — start dev server on **http://localhost:8080**
- `go run . --create-admin <username>` — create admin user (prompts for password), then exits
- `go run -tags importer . <excel-file>` — import data from Excel into DB
- `go build ./...` — verify compilation

## Environment
- `SESSION_SECRET` — required in production, long random string for session signing. Falls back to insecure default in dev.
- `GIN_MODE` — set to `release` (default) or `debug` to control Gin output.

## Architecture
```
main.go                → Gin HTTP server (release mode by default; //go:build !importer)
importer_main.go       → one-shot Excel → SQLite importer (//go:build importer)
db.go                  → GORM + SQLite/PostgreSQL, path: data/archivist.db
models.go              → 10 models: User, Character, DLUsage, Transaction,
                         CostOfLiving, CharacterRegistry, Mission, MissionEntry,
                         Guild, GuildTransaction
services.go            → business logic: XP/level, gold/DL/renown calculations
users.go               → password hashing + user authentication
guild_treasury.go      → guild treasury sync (Arcas)
importer.go            → Excel import engine
routes.go              → Gin route setup + all CRUD handlers
auth.go                → login/logout handlers
middleware.go          → auth + CSRF middleware
render.go              → template compilation and rendering
importer_handlers.go   → web-based Excel import handlers
resources/base.html    → Base layout with sidebar, auth status, CSRF
resources/login.html   → Standalone login form (no base layout)
resources/pages/*.html → Content-only templates (define "content" block)
resources/static/      → CSS and other static assets
```

## Key quirks
- **DB auto-migrates** on every server/importer start — schema changes happen live
- **Two binaries in one flat `package main`**: `go run .` runs the server; the importer is built/run with the `importer` build tag (`go run -tags importer . <excel-file>`)
- **Server starts in release mode** by default; set `GIN_MODE=debug` for verbose output
- **Templates**: base layout (`resources/base.html`) + content blocks (`resources/pages/*.html`). Login page is standalone.
- **Auth**: session cookies via `gin-contrib/sessions` + cookie store. All routes except `/login` and `/static` require authentication.
- **CSRF**: token stored in session, validated on all POST/PUT/DELETE requests. Every form includes `<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">`.
- **SQLite file**: `data/archivist.db` — relative to working directory
- **Excel importer** reads Spanish sheet names (e.g. "Lista de Personajes", "Uso de DL", "Economía de Gremios")
- **Guild treasury** (`Arcas`) is synced from the sum of `GuildTransaction` rows

## Routes
`/login` `/logout` `/` `/characters` `/characters/detail/:id` `/missions` `/missions/detail/:id`
`/dl` `/dl/usages` `/transactions` `/transactions/detail/:id`
`/cost-of-living` `/import` `/guilds` `/guilds/detail/:id`

## Project Rules
YOU ARE FORBIDDEN TO RUN `git commit`. Do not stage or commit any files. 
