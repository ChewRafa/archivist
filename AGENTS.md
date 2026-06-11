# AGENTS.md — Archivist

## Project
Go 1.25.5 web app for TTRPG character/guild tracking. Single module, no monorepo.

## Commands
- `go run cmd/server/main.go` — start dev server on **http://localhost:8080**
- `go run cmd/server/main.go --create-admin <username>` — create admin user (prompts for password), then exits
- `go run cmd/importer/main.go <excel-file>` — import data from Excel into DB
- `go build ./...` — verify compilation

## Environment
- `SESSION_SECRET` — required in production, long random string for session signing. Falls back to insecure default in dev.
- `GIN_MODE` — set to `release` (default) or `debug` to control Gin output.

## Architecture
```
cmd/server/main.go      → Gin HTTP server (release mode by default)
cmd/importer/main.go    → one-shot Excel → SQLite importer
internal/db/db.go       → GORM + SQLite, path: data/archivist.db
internal/models/        → 10 models: User, Character, DLUsage, Transaction,
                          CostOfLiving, CharacterRegistry, Mission, MissionEntry, Guild,
                          GuildTransaction
internal/services/      → business logic: XP/level, gold/DL/renown calculations + auth
internal/handlers/      → Gin route setup, CRUD handlers, auth, middleware, render
templates/base.html     → Base layout with sidebar, auth status, CSRF
templates/login.html    → Standalone login form (no base layout)
templates/pages/*.html  → Content-only templates (define "content" block)
static/                 → CSS and other static assets
```

## Key quirks
- **DB auto-migrates** on every server/importer start — schema changes happen live
- **All go.mod deps are `// indirect`** — direct imports only in subpackages, not root
- **Server starts in release mode** by default; set `GIN_MODE=debug` for verbose output
- **Templates**: base layout (`templates/base.html`) + content blocks (`templates/pages/*.html`). Login page is standalone.
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
