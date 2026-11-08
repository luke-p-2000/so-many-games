# 0008 What counts as the backlog

Status: Accepted · 2026-11-07 (owner decision; easy to amend)

## Decision
The backlog is the eligible pool (R3): classified, not hidden, and either Unplayed or To play at any hours, or Shelved or Endless under the hours limit. Stats (T1) shows:
- **Backlog size:** the number of games in that pool.
- **Backlog hours:** the hours already played on those games. Time-to-beat is deferred (ADR 0005), so there is no "hours left" figure.
- ~~**Burn-down:** the backlog size at each sync since first sync.~~ Removed by ADR 0016, along with the `backlog_snapshot` events that fed it.

## Why the pool
The backlog then means "what the wheel and recommendations could still offer". If the hours limit changes, the backlog changes with it.

## Amending
Change the single `backlog` predicate in `recommend` (shared with the pool) and note it here.
