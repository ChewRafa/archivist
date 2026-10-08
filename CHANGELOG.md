# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-alpha.2] - 2026-10-07

### Added
- JSON export page — `GET /export` shows per-table record counts (including soft-deleted rows) and `POST /export/download` dumps every table as a pretty-printed JSON file (`export.go`, `export_handlers.go`, `resources/pages/export.html`)
- JSON restore import — `POST /import` now accepts `.xlsx` or `.json`; a JSON upload performs a full restore that wipes all content tables *except `users`* and reloads the export as-is (IDs, timestamps, and `deleted_at` preserved) in a single transaction, then refreshes `character_stats`. Requires an explicit confirmation checkbox (`import_json.go`)
- Import format detection by content sniffing (`{` vs `PK\x03\x04`) with extension fallback, BOM-tolerant (`detectImportFormat`)
- Import result extras for restores: origin metadata (version, export date, row count) and a guild-member count card
- Login rate limiting with an in-memory per-IP limiter and failed-attempt tracking (`middleware.go`, `auth.go`); configurable via `LOGIN_MAX_ATTEMPTS` (default `5`) and `LOGIN_WINDOW_MINUTES` (default `15`)
- Structured logging with `log/slog` and a JSON handler initialized by `InitLogger` (`logger.go`)
- Pagination for Transactions, DL, and Cost of Living (`?page=&per_page=`, default 25) with a reusable `pagination` partial and `seq`/`add`/`sub` template funcs
- `character_stats` database view to eliminate N+1 queries in dashboard stats, rebuilt via `RefreshCharacterStatsView` (`db.go`, `services.go`)
- Unique index and create/update validation for `Character.Number` (`models.go`, `routes.go`)
- Database connectivity check in `/health` — returns `503` with `db: unavailable`/`db: unreachable` on failure (`main.go`)
- Drag-and-drop Excel import upload UI with a result dashboard
- Admin user upsert from `ADMIN_USERNAME`/`ADMIN_PASSWORD` on boot (`EnsureAdminFromEnv` in `users.go`)
- `.env` file loading in local dev and `DB_PATH` support in the importer
- Spanish `dd/mmm/yyyy` date display across the UI
- Sortable character tables, gold alerts, default sort, and dynamic form selects
- Tests: `admin_seed_test.go` (admin env seed), `services_stats_test.go` (dashboard stats), and `import_json_test.go` (JSON restore round-trip, users preserved, ID sequence, format detection)

### Fixed
- Import error responses no longer drop the sheet-selection checkboxes — `renderImportPage` always injects `Sheets`/`Format` (`importer_handlers.go`)
- Importer binary failed to compile — restore the missing `log/slog` import (`importer_main.go`)
- Stop regenerating the CSRF token on every request — the session token now persists across renders (`render.go`)
- Race condition in `UpsertAdminUser` by wrapping the lookup/create in a transaction (`users.go`)
- Use the root context for `Pagination.Page` so the template resolves the current page correctly

### Changed
- Replace all `log.*` calls with `slog.*` (`db.go`, `render.go`, `users.go`)
- Rewrite the dashboard stats query with inline error handling (`services.go`)
- Link release binaries with the `netgo` tag and strip debug info (`build.sh`)
- Wire `DATABASE_URL` via `fromDatabase` and pin generated secrets in `render.yaml` (`sync: false`)

## [0.1.0-alpha.1] - 2026-09-22

### Added
- Semantic versioning with `v0.1.0-alpha.1` preliminary release
- `Version` variable (`version.go:1`) injected via `-ldflags "-X main.Version=..."`
- `--version` flag for both server and importer binaries (`main.go:18`, `importer_main.go:17`)
- Version exposed via `/health` endpoint as `{"status":"ok","version":"..."}` (`main.go:71`)
- Build integration: `build.sh:3` and `Makefile:14` now derive `VERSION` from `git describe --tags`

### Fixed
- `db.go:23` - ensure database directory exists before opening SQLite (`os.MkdirAll`)
- `cost-of-living` - correct CSRF token scope in delete form
- `guild` - display only alive members and enforce active-only capacity

### Changed
- `importer` - move `log.SetPrefix("[importer] ")` from `init()` to `main()` (`importer_main.go:15`)
- Guild invariants and dashboard charts replication
- Flatten to single `package main` with build tags and unified resources

[0.1.0-alpha.2]: https://codeberg.org/chewrafa/archivist/releases/tag/v0.1.0-alpha.2
[0.1.0-alpha.1]: https://codeberg.org/chewrafa/archivist/releases/tag/v0.1.0-alpha.1
