# Archivist

TTRPG Character and Guild Tracking — a web application built with Go for managing characters, missions, transactions, and guilds in tabletop role-playing games.

## Features

- **Dashboard** — overview stats, level/class/species distributions, recent missions and transactions
- **Characters** — full CRUD with status tracking (Active, Retired, Dead), auto-calculated level, XP, gold balance, and renown
- **Missions** — create missions with per-character XP, gold, and renown entries
- **DL (Días Libres)** — track free-day usage per character with gold adjustments
- **Cost of Living** — record recurring upkeep costs per character
- **Transactions** — track gold income and expenses per character
- **Guilds** — manage guilds with leaders, members, halls, treasuries, and cost of living
- **Excel Import** — bulk import data from Excel spreadsheets
- **Authentication** — session-based auth with bcrypt password hashing and CSRF protection

## Tech Stack

- **Language:** Go 1.25.5
- **Framework:** [Gin](https://github.com/gin-gonic/gin) v1.12
- **ORM:** [GORM](https://gorm.io) v1.31 with SQLite driver
- **Database:** SQLite
- **Templates:** Go `html/template` with block layout
- **CSS:** [Bulma](https://bulma.io) + custom styles
- **Auth:** bcrypt, session cookies via `gin-contrib/sessions`

## Prerequisites

- Go 1.25.5 or later

## Quick Start

```bash
# Verify compilation
go build ./...

# Start the development server
go run cmd/server/main.go
```

Open http://localhost:8080 in your browser and log in.

## Configuration

Configuration is handled via environment variables. See `env.example` for a template.

| Variable          | Required | Default                                      | Description                            |
|-------------------|----------|----------------------------------------------|----------------------------------------|
| `SESSION_SECRET`  | In prod  | `dev-secret-change-in-production`            | Key for signing session cookies        |
| `GIN_MODE`        | No       | `release`                                    | Gin mode (`release` or `debug`)        |
| `PORT`            | No       | `8080`                                       | Server port (set automatically by Render) |
| `DB_PATH`         | No       | `data/archivist.db`                          | SQLite path (local dev)                |
| `DATABASE_URL`    | On Render | —                                           | PostgreSQL DSN (set automatically by Render, overrides SQLite) |
| `ADMIN_USERNAME`  | On first deploy | —                                     | Initial admin username (auto-created if no users exist) |
| `ADMIN_PASSWORD`  | On first deploy | —                                     | Initial admin password                 |

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
6. On first deploy, if `ADMIN_USERNAME` and `ADMIN_PASSWORD` are set, an admin user is created automatically
7. Retrieve `SESSION_SECRET` and `ADMIN_PASSWORD` from Render's **Environment** tab

> **Important**: After the first deploy succeeds, remove `ADMIN_USERNAME` and `ADMIN_PASSWORD` env vars for security.

### Manual setup

| Setting               | Value                     |
|------------------------|---------------------------|
| **Runtime**            | Go                        |
| **Build Command**      | `./build.sh`              |
| **Start Command**      | `./app`                   |
| **Health Check Path**  | `/health`                 |
| **PostgreSQL**         | Create a free Render PostgreSQL instance |

Required environment variables:
- `SESSION_SECRET` — set to a long random string
- `DATABASE_URL` — set automatically when PostgreSQL is linked; the app auto-detects this and uses PostgreSQL

Optional (first deploy only):
- `ADMIN_USERNAME` — initial admin username
- `ADMIN_PASSWORD` — initial admin password

### Local vs Render

The app auto-detects the environment:

| Env | Database | Config |
|-----|----------|--------|
| **Local dev** | SQLite (`data/archivist.db`) | No `DATABASE_URL` set |
| **Render** | PostgreSQL (free 1 GB) | `DATABASE_URL` set automatically |

## Usage

### Running the server

```bash
go run cmd/server/main.go
```

Starts the HTTP server on `:8080`. The SQLite database is auto-created at `data/archivist.db` on first run.

### Creating an admin user

```bash
go run cmd/server/main.go --create-admin <username>
```

You will be prompted for a password. The command creates the user and exits.

### Importing data from Excel

```bash
go run cmd/importer/main.go <path-to-excel-file>
```

Imports data from an Excel file with Spanish sheet names. See [Excel Import Format](#excel-import-format) for details.

## Project Structure

```
cmd/
├── server/main.go         HTTP server entry point
└── importer/main.go       Excel → SQLite import tool

internal/
├── db/db.go               GORM + SQLite initialization and auto-migration
├── handlers/
│   ├── auth.go            Login/logout handlers
│   ├── handlers.go        All CRUD handlers and route setup
│   ├── importer.go        Web-based Excel import handler
│   ├── middleware.go      Auth and CSRF middleware
│   └── render.go          Template compilation and rendering
├── models/
│   ├── models.go          Character, Transaction, CostOfLiving, CharacterRegistry,
│                          Mission, MissionEntry, Guild
│   └── user.go            User model
└── services/
    ├── auth.go            Password hashing and user authentication
    └── services.go        XP/level/gold/renown calculations

templates/
├── base.html              Base layout with sidebar and CSRF
├── login.html             Standalone login page
└── pages/                 Content templates for each page

static/
├── app.css                Custom styles
└── bulma.min.css          Bulma CSS framework

data/
└── archivist.db           SQLite database (auto-created)
```

## Routes

### Public (no authentication required)

| Method | Path          | Description        |
|--------|---------------|--------------------|
| GET    | `/login`      | Login page         |
| POST   | `/login`      | Login form submit  |
| GET    | `/static/*`   | Static files       |

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
| POST   | `/import`                                          | Submit Excel import      |
| GET    | `/guilds`                                          | Guild list               |
| GET    | `/guilds/create`                                   | New guild form           |
| POST   | `/guilds`                                          | Create guild             |
| GET    | `/guilds/detail/:id`                               | Guild detail             |
| GET    | `/guilds/detail/:id/edit`                          | Edit guild form          |
| POST   | `/guilds/detail/:id`                               | Update guild             |
| POST   | `/guilds/detail/:id/delete`                        | Delete guild             |

All mutating requests (POST) require a valid `csrf_token` field.

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

## Development

The database auto-migrates on every start — schema changes are applied live. Set `GIN_MODE=debug` for verbose Gin output:

```bash
GIN_MODE=debug go run cmd/server/main.go
```

## License

MIT
