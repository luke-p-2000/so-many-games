# 0002 L2: 0 minutes sets Unplayed only for newly seen games

Status: Accepted · 2026-11-07 (owner decision)

## Context
L2 says "when a sync shows 0 minutes played, the app shall set Unplayed". Read literally, that would reset a 0-minute game the owner had set to To play back to Unplayed on every sync.

## Decision
- A game first seen with 0 minutes starts as Unplayed, both on first import (L7) and in later syncs.
- After that, a sync only promotes Unplayed, To play or Shelved to Playing when playtime increases.
- Syncs never change Finished, Not for me or Endless, and never demote anything to Unplayed.

## Consequences
Statuses set by hand stick. The requirements text for L2 should be updated to match.
