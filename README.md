# Archivist

> **Version:** `v0.1.0-alpha.2` (preliminary) — see [CHANGELOG.md](CHANGELOG.md)

TTRPG Character and Guild Tracking — a web application built with Go for managing characters, missions, transactions, and guilds in tabletop role-playing games.

## Features

- **Dashboard** — overview stats, level/class/species distributions, recent missions and transactions
- **Characters** — full CRUD with status tracking (Active, Retired, Dead), auto-calculated level, XP, gold balance, and renown
- **Missions** — create missions with per-character XP, gold, and renown entries
- **DL (Días Libres)** — track free-day usage per character with gold adjustments, with pagination
- **Cost of Living** — record recurring upkeep costs per character, with pagination
- **Transactions** — track gold income and expenses per character, with pagination
- **Guilds** — manage guilds with leaders, members, halls, treasuries, and cost of living
- **Import / Export** — bulk import from Excel spreadsheets via drag-and-drop, restore the whole database from an exported JSON backup, and download a full JSON export of every table
- **Authentication** — session-based auth with bcrypt password hashing, CSRF protection, and login rate limiting (brute-force protection)
- **Structured logging** — JSON logs via Go's `log/slog`

## Tech Stack

- **Language:** Go 1.25.5
- **Framework:** [Gin](https://github.com/gin-gonic/gin) v1.12
- **ORM:** [GORM](https://gorm.io) v1.31 with SQLite driver
- **Database:** SQLite
- **Templates:** Go `html/template` with block layout
- **CSS:** [Bulma](https://bulma.io) + custom styles
- **Auth:** bcrypt, session cookies via `gin-contrib/sessions`
- **Logging:** Go `log/slog` with JSON handler (`logger.go`)

## Prerequisites

- Go 1.25.5 or later

## Quick Start

```bash
# Verify compilation
go build ./...

# Start the development server
go run .
```

Open http://localhost:8080 in your browser and log in.

## Configuration

Configuration is handled via environment variables. See `env.example` for a template. In local dev a `.env` file is loaded automatically (skipped when `GIN_MODE=release`); real environment variables always take precedence.

| Variable             | Required | Default                                      | Description                            |
|----------------------|----------|----------------------------------------------|----------------------------------------|
| `SESSION_SECRET`     | In prod  | `dev-secret-change-in-production`            | Key for signing session cookies        |
| `GIN_MODE`           | No       | `release`                                    | Gin mode (`release` or `debug`)        |
| `PORT`               | No       | `8080`                                       | Server port (set automatically by Render) |
| `DB_PATH`            | No       | `data/archivist.db`                          | SQLite path (local dev and importer)   |
| `DATABASE_URL`       | On Render | —                                           | PostgreSQL DSN (set automatically by Render, overrides SQLite) |
| `ADMIN_USERNAME`     | On first deploy | —                                     | Initial admin username (created if missing; password re-synced from env on every boot while set) |
| `ADMIN_PASSWORD`     | On first deploy | —                                     | Initial admin password (unset both after first login; generated values in `render.yaml` use `sync: false` so dashboard edits survive Blueprint syncs) |
| `LOGIN_MAX_ATTEMPTS` | No       | `5`                                          | Max failed login attempts per IP before throttling |
| `LOGIN_WINDOW_MINUTES` | No     | `15`                                         | Sliding window (minutes) for login rate limiting |

## Deployment

### Deploy on Render (free tier)

The repo includes a [`render.yaml`](render.yaml) for one-click deployment.

1. Push this repo to GitHub/GitLab
2. In the [Render Dashboard](https://dashboard.render.com), click **New → Blueprint**
3. Connect your repo — Render auto-detects `render.yaml`
4. Render creates:
   - A **Web Service** (free tier — sleeps after 15 min idle)
   - A **PostgreSQL database** (free tier — 1 GB)
5. Render automatically sets `DATABASE_URL` on the web service — the app detects it and uses PostgreSQL instead of SQLite
6. On first deploy, if `ADMIN_USERNAME` and `ADMIN_PASSWORD` are set, the admin user is created automatically (or its password is reset if it already exists — check service logs for `Admin user '...' created/password synced`)
7. Retrieve `SESSION_SECRET` and `ADMIN_PASSWORD` from Render's **Environment** tab

> **Important**: After the first deploy succeeds, remove `ADMIN_USERNAME` and `ADMIN_PASSWORD` env vars for security.

### Manual setup

| Setting               | Value                     |
|------------------------|---------------------------|
| **Runtime**            | Go                        |
| **Build Command**      | `./build.sh`              |
| **Start Command**      | `./app`                   |
| **Health Check Path**  | `/health`                 |
| **PostgreSQL**         | Link a Render PostgreSQL instance so `DATABASE_URL` is injected (`fromDatabase: archivist-db`) |

Required environment variables:
- `SESSION_SECRET` — set to a long random string
- `DATABASE_URL` — injected from `archivist-db` via `fromDatabase` in `render.yaml` when linked; the app auto-detects this and uses PostgreSQL

Optional (first deploy only):
- `ADMIN_USERNAME` — initial admin username (created if missing)
- `ADMIN_PASSWORD` — initial admin password (resets the admin password on every boot while set, so it also recovers a lost password)

### Local vs Render

The app auto-detects the environment:

| Env | Database | Config |
|-----|----------|--------|
| **Local dev** | SQLite (`data/archivist.db`) | No `DATABASE_URL` set |
| **Render** | PostgreSQL (free 1 GB) | `DATABASE_URL` injected from `archivist-db` via `fromDatabase` in `render.yaml` |

## Usage

### Running the server

```bash
go run .
```

Starts the HTTP server on `:8080`. The SQLite database is auto-created at `data/archivist.db` on first run.

### Creating an admin user

```bash
go run . --create-admin <username>
```

You will be prompted for a password. The command creates the user and exits.

### Importing data from Excel

```bash
go run -tags importer . <path-to-excel-file>
```

Imports data from an Excel file with Spanish sheet names. See [Excel Import Format](#excel-import-format) for details.

### Exporting and restoring data

The web UI provides a matching export/restore pair:

```bash
# In the browser
GET  /export            # shows per-table record counts (incl. soft-deleted rows)
POST /export/download   # downloads archivist-export-<timestamp>.json
POST /import            # upload the .json back to restore the database
```

- The export contains every table as JSON, including soft-deleted records.
- Uploading a `.json` file on the import page performs a **full restore**: all content tables are wiped and reloaded from the file in a single transaction (IDs, timestamps, and soft-deletes are preserved). The `users` table is **never** touched, and an explicit confirmation checkbox is required.
- The sheet-selection checkboxes apply only to `.xlsx` uploads; JSON files always restore every section.
- The CLI importer (`-tags importer`) stays Excel-only.

## Project Structure

```
main.go                HTTP server entry point (server binary)
importer_main.go       Excel → SQLite import tool (importer binary)
routes.go              Route setup and all CRUD handlers
auth.go                Login/logout handlers
middleware.go          Auth, CSRF, and login rate-limit middleware
render.go              Template compilation and rendering
db.go                  GORM + SQLite/PostgreSQL initialization, auto-migration,
                       and the character_stats view
models.go              Models: Character, DLUsage, Transaction, CostOfLiving,
                       CharacterRegistry, Mission, MissionEntry, Guild,
                       GuildTransaction
models_user.go         User model
services.go            XP/level/gold/renown calculations + stats queries
users.go               Password hashing, user auth, admin user upsert
guild_service.go       Guild invariants (join/leave validation, name sync)
guild_treasury.go      Guild treasury sync
importer.go            Excel import engine
importer_handlers.go   Web-based import handlers (.xlsx + .json restore)
import_json.go         JSON restore engine + import format detection
export.go              ExportData struct + ExportAll query (all tables → JSON)
export_handlers.go     Export page and JSON download handlers
logger.go              slog JSON logger initialization
version.go             Version variable (injected via -ldflags)
admin_seed_test.go     Admin env-seed tests
services_stats_test.go Dashboard stats tests
import_json_test.go    JSON restore round-trip and format detection tests

Makefile               Common build/run/test targets
build.sh               Release build used by Render

resources/
├── base.html          Base layout with sidebar, CSRF, pagination partial
├── login.html         Standalone login page
├── pages/             Content templates for each page
└── static/            CSS and other static assets

data/
└── archivist.db       SQLite database (auto-created)
```

Both binaries live in a single flat `package main`. The server is built by default (`go build -tags netgo -o app .`); the importer shares the same code and is built with a build tag: `go build -tags importer -o importer .`.

## Routes

### Public (no authentication required)

| Method | Path          | Description                                |
|--------|---------------|--------------------------------------------|
| GET    | `/login`      | Login page                                 |
| POST   | `/login`      | Login form submit (rate limited per IP)    |
| GET    | `/static/*`   | Static files                               |
| GET    | `/health`     | Health check (DB ping; `503` if unreachable) |

### Authenticated

| Method | Path                                               | Description              |
|--------|----------------------------------------------------|--------------------------|
| GET    | `/`                                                | Dashboard                |
| POST   | `/logout`                                          | Log out                  |
| GET    | `/characters`                                      | Character list           |
| GET    | `/characters/create`                               | New character form       |
| POST   | `/characters`                                      | Create character         |
| GET    | `/characters/detail/:id`                           | Character detail         |
| GET    | `/characters/detail/:id/edit`                      | Edit character form      |
| POST   | `/characters/detail/:id`                           | Update character         |
| POST   | `/characters/detail/:id/delete`                    | Delete character         |
| GET    | `/missions`                                        | Mission list             |
| GET    | `/missions/create`                                 | New mission form         |
| POST   | `/missions`                                        | Create mission           |
| GET    | `/missions/detail/:id`                             | Mission detail           |
| GET    | `/missions/detail/:id/edit`                        | Edit mission form        |
| POST   | `/missions/detail/:id`                             | Update mission           |
| POST   | `/missions/detail/:id/delete`                      | Delete mission           |
| POST   | `/missions/detail/:id/entries`                     | Add entry to mission     |
| GET    | `/missions/detail/:id/entries/:eid/edit`           | Edit mission entry       |
| POST   | `/missions/detail/:id/entries/:eid`                | Update mission entry     |
| POST   | `/missions/detail/:id/entries/:eid/delete`         | Delete mission entry     |
| GET    | `/dl`                                              | DL usage list            |
| POST   | `/dl/usages`                                       | Create DL usage          |
| GET    | `/dl/usages/:id/edit`                              | Edit DL usage form       |
| POST   | `/dl/usages/:id`                                   | Update DL usage          |
| POST   | `/dl/usages/:id/delete`                            | Delete DL usage          |
| GET    | `/transactions`                                    | Transaction list         |
| POST   | `/transactions`                                    | Create transaction       |
| GET    | `/transactions/detail/:id/edit`                    | Edit transaction form    |
| POST   | `/transactions/detail/:id`                         | Update transaction       |
| POST   | `/transactions/detail/:id/delete`                  | Delete transaction       |
| GET    | `/cost-of-living`                                  | Cost of living list      |
| POST   | `/cost-of-living`                                  | Create cost of living    |
| GET    | `/cost-of-living/:id/edit`                         | Edit cost of living form |
| POST   | `/cost-of-living/:id`                              | Update cost of living    |
| POST   | `/cost-of-living/:id/delete`                       | Delete cost of living    |
| GET    | `/import`                                          | Import page              |
| POST   | `/import`                                          | Import `.xlsx` or restore `.json` backup |
| GET    | `/export`                                          | Export page (record counts) |
| POST   | `/export/download`                                 | Download full DB as JSON |
| GET    | `/guilds`                                          | Guild list               |
| GET    | `/guilds/create`                                   | New guild form           |
| POST   | `/guilds`                                          | Create guild             |
| GET    | `/guilds/detail/:id`                               | Guild detail             |
| GET    | `/guilds/detail/:id/edit`                          | Edit guild form          |
| POST   | `/guilds/detail/:id`                               | Update guild             |
| POST   | `/guilds/detail/:id/delete`                        | Delete guild             |
| POST   | `/guilds/detail/:id/transactions`                  | Create guild transaction |
| GET    | `/guilds/detail/:id/transactions/:txId/edit`       | Edit guild transaction   |
| POST   | `/guilds/detail/:id/transactions/:txId`            | Update guild transaction |
| POST   | `/guilds/detail/:id/transactions/:txId/delete`     | Delete guild transaction |

All mutating requests (POST) require a valid `csrf_token` field.

List pages (`/transactions`, `/dl`, `/cost-of-living`) are paginated via `?page=N&per_page=M` (default 25 per page).

## Excel Import Format

The importer reads from an Excel file with Spanish sheet names:

| Sheet Name                  | Description                          |
|-----------------------------|--------------------------------------|
| `Lista de Personajes`       | Character roster                     |
| `Uso de DL`                 | DL usage records                     |
| `Compras`                   | Character transactions               |
| `Costo de Vida`             | Cost of living records               |
| `Registro de Personajes`    | Character registry/event log         |
| `Registro de Misiones`      | Mission records                      |
| `Gremios`                   | Guild records                        |
| `Economía de Gremios`       | Guild treasury ledger (Arcas)        |

Guild treasury (`Arcas`) is computed as the sum of all economy transactions for that guild.

## JSON Export Format

`POST /export/download` produces a single pretty-printed JSON object with:

| Section                  | Contents                                            |
|--------------------------|-----------------------------------------------------|
| `exported_at`, `version` | Export timestamp and build version                  |
| `characters`             | All characters, including soft-deleted (`deleted_at`) |
| `dl_usages`              | Free-day usage records                              |
| `transactions`           | Character transactions                              |
| `cost_of_livings`        | Cost of living records                              |
| `character_registries`   | Character XP/gold/renown event log                  |
| `missions`               | Missions (including soft-deleted)                   |
| `mission_entries`        | Per-character mission XP/gold/renown (including soft-deleted) |
| `guilds`                 | Guilds (treasury, hall, dates)                      |
| `guild_transactions`     | Guild treasury ledger (Arcas)                       |
| `guild_members`          | Guild membership pairs (`guild_id`, `character_id`) |
| `character_stats`        | Derived view output — not imported back, regenerated by `RefreshCharacterStatsView` during a restore |

Uploading this file on the import page restores the database exactly as exported (see [Exporting and restoring data](#exporting-and-restoring-data)).

## Development

The database auto-migrates on every start — schema changes are applied live. Set `GIN_MODE=debug` for verbose Gin output:

```bash
GIN_MODE=debug go run .
```

Other useful commands:

```bash
go test ./...               # run tests
make                        # build server + importer binaries
make run                    # start dev server (GIN_MODE=debug)
make test                   # same as go test ./...
```

Logs are structured JSON written to stdout via `log/slog` (initialized by `InitLogger` in `logger.go`).

### Versioning

This project uses [Semantic Versioning](https://semver.org). The current version is injected at build time:

```bash
go run . --version          # prints Version (dev if no tag)
go build -tags netgo -ldflags "-s -w -X main.Version=v0.1.0-alpha.2" -o app .
make build                  # auto-derives VERSION from git describe --tags
make version                # print the derived VERSION
./build.sh                  # same as make build, used by Render
curl http://localhost:8080/health  # {"status":"ok","version":"v0.1.0-alpha.2"}
```

`/health` also pings the database: if the connection fails it returns `503` with `{"status":"error","db":"unavailable","version":"..."}` (or `"db":"unreachable"` when the ping fails).

See [CHANGELOG.md](CHANGELOG.md) for release history.

## License

MIT
