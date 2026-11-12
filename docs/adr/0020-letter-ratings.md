# 0020 Ratings use the tier list's letters

Status: Accepted · 2026-11-12 (owner decision)

## Context
Finished games took a 1–10 rating (L4) while the tier list (ADR 0016) ranks them S to F. Two scales for the same judgement didn't tie together.

## Decision
- A Finished game's rating is one of **S, A, B, C, D or F**, picked on the card like before; clicking the current letter clears it. The selected letter takes its tier list colour.
- Existing scores convert once (migration 003): **0 = F, 1–2 = D, 3–4 = C, 5–6 = B, 7–8 = A, 9–10 = S**. Past `rated` events convert too.
- Exports are now version 2 with letter ratings. Version 1 files still import; their scores convert with the same mapping.
- The taste profile's liking factor (ADR 0015) maps each letter to a weight, roughly the old scale's midpoints over 5.5: S 1.75, A 1.4, B 1, C 0.65, D 0.3, F 0.1.

## Consequences
- `recommend.RatingWeights` replaces `NeutralRating` as the dial; B is neutral.
- The rating and the tier list placement stay separate: rating a game doesn't place it in the tier list.
