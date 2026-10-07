# AGENTS.md — Archivist

## Project
Go 1.25.5 web app for TTRPG character/guild tracking. Single module, no monorepo.

## Commands
- `go run .` — start dev server on **http://localhost:8080**
- `go run . --create-admin <username>` — create admin user (prompts for password), then exits
- `go run -tags importer . <excel-file>` — import data from Excel into DB
- `go build ./...` — verify compilation
- `go test ./...` — run tests (`admin_seed_test.go`, `services_stats_test.go`); `make test` is equivalent
- `make` — build server (`server`) + importer (`importer`) binaries with version ldflags; `make version` prints the derived version

## Environment
- `SESSION_SECRET` — required in production, long random string for session signing. Falls back to insecure default in dev.
- `GIN_MODE` — set to `release` (default) or `debug` to control Gin output.
- `PORT` — server port (default `8080`).
- `DB_PATH` — SQLite path (default `data/archivist.db`); honored by server and importer.
- `DATABASE_URL` — PostgreSQL DSN; when set (Render injects it), overrides SQLite.
- `ADMIN_USERNAME` / `ADMIN_PASSWORD` — seed admin on boot: created if missing, password re-synced while set. Unset after first login.
- `LOGIN_MAX_ATTEMPTS` / `LOGIN_WINDOW_MINUTES` — login rate limiter (defaults `5` / `15`).
- `.env` — loaded automatically in local dev (skipped when `GIN_MODE=release`); real env vars take precedence.

## Architecture
```
main.go                → Gin HTTP server (release mode by default; //go:build !importer)
importer_main.go       → one-shot Excel → SQLite importer (//go:build importer)
db.go                  → GORM + SQLite/PostgreSQL, path: data/archivist.db, character_stats view
models.go              → 9 models: Character, DLUsage, Transaction,
                         CostOfLiving, CharacterRegistry, Mission, MissionEntry,
                         Guild, GuildTransaction
models_user.go         → User model (10th)
services.go            → business logic: XP/level, gold/DL/renown calculations + stats queries
users.go               → password hashing + user auth + EnsureAdminFromEnv upsert
guild_service.go       → guild invariants: join/leave validation, character guild name sync
guild_treasury.go      → guild treasury sync (Arcas)
importer.go            → Excel import engine
routes.go              → Gin route setup + all CRUD handlers
auth.go                → login/logout handlers
middleware.go          → auth + CSRF + login rate-limit middleware
render.go              → template compilation and rendering
logger.go              → InitLogger: slog JSON handler (log/slog)
version.go             → Version var injected via -ldflags
importer_handlers.go   → web-based Excel import handlers
export.go              → ExportData struct + ExportAll query (all models → single JSON)
export_handlers.go     → GET /export page + POST /export/download JSON file
admin_seed_test.go     → admin env-seed tests
services_stats_test.go → dashboard stats tests
resources/base.html    → Base layout with sidebar, auth status, CSRF, pagination partial
resources/login.html   → Standalone login form (no base layout)
resources/pages/*.html → Content-only templates (define "content" block)
resources/static/      → CSS and other static assets
```

## Key quirks
- **DB auto-migrates** on every server/importer start — schema changes happen live
- **Two binaries in one flat `package main`**: `go run .` runs the server; the importer is built/run with the `importer` build tag (`go run -tags importer . <excel-file>`)
- **Server starts in release mode** by default; set `GIN_MODE=debug` for verbose output
- **Templates**: base layout (`resources/base.html`) + content blocks (`resources/pages/*.html`). Login page is standalone. The `pagination` partial lives in `base.html` and uses `seq`/`add`/`sub` funcs.
- **Auth**: session cookies via `gin-contrib/sessions` + cookie store. All routes except `/login` and `/static` require authentication.
- **CSRF**: token stored in session, validated on all POST/PUT/DELETE requests. Every form includes `<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">`. The token is NOT regenerated per request — it persists across renders.
- **Login rate limit**: in-memory per-IP limiter on `POST /login` (`RateLimitLogin` + `recordFailedLoginAttempt`), configured by `LOGIN_MAX_ATTEMPTS`/`LOGIN_WINDOW_MINUTES`.
- **Logging**: `log/slog` JSON handler to stdout, initialized by `InitLogger()` at the start of both entry points — use `slog.*`, never `log.*`.
- **`character_stats` view**: SQLite view over characters + aggregated XP/gold/renown, queried by dashboard stats; recreated by `RefreshCharacterStatsView` after imports.
- **`Character.Number`** has a unique index and is validated on create/update.
- **SQLite file**: `data/archivist.db` — relative to working directory (override with `DB_PATH`)
- **Excel importer** reads Spanish sheet names (e.g. "Lista de Personajes", "Uso de DL", "Economía de Gremios")
- **Guild treasury** (`Arcas`) is synced from the sum of `GuildTransaction` rows

## Routes
`/login` `/logout` `/` `/health` `/characters` `/characters/detail/:id` `/missions` `/missions/detail/:id`
`/dl` `/dl/usages` `/transactions` `/transactions/detail/:id`
`/cost-of-living` `/import` `/export` `/guilds` `/guilds/detail/:id`
`/guilds/detail/:id/transactions[/:txId]`

List pages `/transactions`, `/dl`, `/cost-of-living` accept `?page=&per_page=` (default 25).

## Project Rules
YOU ARE FORBIDDEN TO RUN `git commit`. Do not stage or commit any files. 
