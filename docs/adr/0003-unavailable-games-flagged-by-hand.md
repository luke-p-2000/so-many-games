# 0003 Unavailable games are flagged by hand

Status: Accepted · 2026-11-07 (owner decision; replaces the "two not-found results" proposal)

## Context
The requirements left open whether games Steam can never return data for (delisted, region-locked) should keep counting in "No data for N games".

## Decision
- Nothing is marked unavailable automatically. A failed fetch goes to the failures list (S6) with its reason, as usual.
- Each failure has a **Mark unavailable** action next to Retry. A game marked unavailable:
  - is never fetched again, including by Retry all and Re-fetch all;
  - no longer counts in the "No data" banner;
  - stays in the library with whatever data it has.
- The mark can be undone from the game's row in the Library, which returns the game to the fetch queue.

## Consequences
`game` gets a `meta_unavailable` flag. The fetch queue selects only games with no `game_meta` row whose flag is unset.
