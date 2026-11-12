# 0015 Ratings and Not for me weight the taste profile

Status: Accepted · 2026-11-07 · rating scale changed to S–F by ADR 0020

## Context
The taste profile (R1) counted every played game by its dampened hours alone. A game rated 2/10 counted the same as one rated 10/10, and a game marked Not for me after many hours counted as fully as a favourite.

## Decision
Each game's weight is its square-root-dampened hours times a liking factor:

| Game | Factor |
| --- | --- |
| Not for me | 0 (left out of the profile) |
| Rated (Finished) | rating / 5.5, so 10 counts about double and 1 under a fifth |
| Anything else | 1 |

It applies to both the All-time and Recent profiles, and so to recommendations, Pick back up's matched tags and the wheel's taste bias.

## Consequences
- One formula in `recommend.liking`; the neutral point (`NeutralRating`, 5.5) is the single dial to adjust.
- Unrated Finished games and games still in play count as before.
