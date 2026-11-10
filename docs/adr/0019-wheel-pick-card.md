# 0019 An "Up next" share card for the wheel's pick

Status: Accepted · 2026-11-10 (owner decision, for the friends' backlog challenge)

## Context
Friends doing a backlog challenge together post the finished card (ADR 0016) when they finish a game. They wanted a matching image to announce the next game when the wheel picks it.

## Decision
- A **Share** button in the wheel's result panel opens the share dialog (ADR 0018) with an **Up next** card to copy or save, named like "Portal 2 up next.png".
- The card is 1200 × 630, drawn at twice that size on a canvas, with the same frame, logo and fonts as the other cards. It's in the wheel's amber where the finished card is green: the blurred cover behind a dark wash, an amber glow and frame, and an **UP NEXT** badge.
- Beside the cover: a glowing "The wheel picked" line with the wheel icon in a ring, then the name (two lines at most, shrinking to fit), the review score in its tone colour with the review count, and up to three tags as amber chips on one row.
- The card has no footer: no date, hours or challenge number, by the owner's choice. A game with no reviews or tags just leaves that row out.
- It is only shown from Share. Mark Playing doesn't open it the way finishing opens the finished card. No Steam branding (ADR 0011).
