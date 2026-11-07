# 0009 Recent views and Abandoned use Steam's last-played date

Status: Accepted · 2026-11-07 (owner decision, confirmed by the probe)

## Context
The probe showed `GetOwnedGames` returns `rtime_last_played` for every owned game, and 0 for games never played. It matched the owner's own list on every game compared except one, where the list predates the latest play. `playtime_2weeks` only appears for games played in the last two weeks: 5 of 231.

## Decision
- **Recent taste profile (R1):** games last played within the recent window (default 6 months, a setting), weighted by their dampened lifetime hours. No snapshot arithmetic.
- **Abandoned (N1):** a Playing game whose last-played date is older than the abandoned setting (default 60 days).
- **Classify split (L7):** "played in the last N days" uses the same field.
- **Recent stats (T1):** use the same rule as the recent profile.
- Snapshots are still stored (E1) for the burn-down and future features. Nothing waits for them to build up.

## Trade-off
One recent session counts a game's whole lifetime hours toward the Recent profile. That is accepted as simple and good enough. If it skews results, weight by `playtime_2weeks` or snapshot deltas instead and record that here.
