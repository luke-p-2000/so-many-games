# So Many Games

A personal Windows app that tracks a Steam library, recommends underplayed games and wheelspin's next plays. New users start with the [user guide](USER_GUIDE.md). The specs are in `docs/spec`, decisions in `docs/adr`.

## Stack

Wails v3.1.0 (Go 1.27 backend, React 18 + TypeScript frontend), SQLite through `modernc.org/sqlite`, the Windows Credential Manager through `go-keyring`, and an NSIS installer. Versions are pinned in [ADR 0001](docs/adr/0001-pinned-toolchain.md).

## Layout

| Path | What |
| --- | --- |
| `main.go` | Window, services and the `/covers/` route |
| `internal/store` | Database, numbered migrations, pre-upgrade backups, settings |
| `internal/steam` | Paced HTTP client, owned-games provider, API key store |
| `internal/meta` | Store-data provider (keyless Steam store APIs) and the fetch queue |
| `internal/library` | Statuses, sync import, Classify and hidden rules |
| `internal/recommend` | Eligible pool, taste profile, Wilson ranking, Tailored for you, Pick back up |
| `internal/wheel` | Wheel filters and draw |
| `internal/stats` | Stats screen figures |
| `internal/transfer` | Backup, export and import |
| `internal/events` | Event log |
| `internal/app` | Thin Wails services the frontend calls |
| `frontend/src` | Screens, components, the status control and data hooks |
| `build/` | Wails build config, app icon (`appicon.svg`) and the NSIS installer script |
| `docs/spec` | Requirements (including deferred and future features), design and UI mocks |
| `docs/adr` | Decision records, indexed in its README |
| `docs/spike` | Findings from the pre-build spike and the library list it was checked against |

## Develop

```bash
wails3 dev
```

```bash
go test ./internal/...
```

The tests use fake providers and never touch the network.

## Build and install

```bash
wails3 package
```

This writes `bin/somanygames-amd64-installer.exe`, a per-user installer with no admin prompt. App data lives in `%APPDATA%\SoManyGames`; the API key lives only in Windows Credential Manager under `SoManyGames`.

## Releasing

Built files are not committed; they go on a GitHub Release. Pushing a `v*` tag runs [.github/workflows/release.yml](.github/workflows/release.yml), which tests, builds the installer on a Windows runner and publishes a release with `somanygames-amd64-installer.exe` and the portable `somanygames.exe`.

1. Set the new version in `build/config.yml` (`info.version`), `build/windows/info.json` (all four version fields) and `build/windows/nsis/wails_tools.nsh` (`INFO_PRODUCTVERSION`), then commit. Production builds read the version from `info.json` and show it in the bottom-right corner of the window and in the Steam user agent.
2. Tag and push:

```bash
git tag v1.0.1
git push origin v1.0.1
```

The workflow stops if the tag doesn't match all three versions. The builds are unsigned, so Windows SmartScreen warns on first run (More info → Run anyway).

Adding a migration: put a new `NNN_name.sql` file in `internal/store/migrations`. It runs at the next start, after the database has been copied to `backups/` (the newest 3 copies are kept).
