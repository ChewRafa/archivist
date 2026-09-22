# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://codeberg.org/chewrafa/archivist/compare/v0.1.0-alpha.1...HEAD
[0.1.0-alpha.1]: https://codeberg.org/chewrafa/archivist/releases/tag/v0.1.0-alpha.1
