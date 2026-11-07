# 0001 Pinned toolchain and per-user installer

Status: Accepted · 2026-11-07

## Context
The design asks for Wails v3 at an exact version. v3.0.0 shipped 2026-10-26 and nightlies follow weekly.

## Decision
| Tool | Version |
| --- | --- |
| Go | 1.27.3 |
| Wails (CLI and module) | v3.1.0, latest stable on 2026-11-07 |
| `@wailsio/runtime` | 3.1.0 |
| `modernc.org/sqlite` | v1.61.1 (SQLite 3.54.1) |
| `github.com/zalando/go-keyring` | v0.2.8 |
| NSIS | 3.13 |
| Node | 22.18 |

Upgrade by deliberate bumps only, never by floating ranges. The NSIS installer uses `INSTALL_SCOPE=user`: it installs to `%LOCALAPPDATA%\Programs` with no admin prompt, which suits a single-user tool.

## Consequences
The spike built, installed and ran on this stack (see docs/spike/findings.md), so the Wails v2 fallback is not needed.
