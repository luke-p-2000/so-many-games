# 0013 Dead status and first UI feedback

Status: Accepted · 2026-11-07 (owner feedback after the first build)

## Dead status
- An eighth status, **Dead**, after Endless, with a headstone icon. It is for games that can no longer be played, such as online games whose servers are gone or broken games.
- Dead games are never on the wheel, in recommendations, in Pick back up or in the backlog. Syncs never change them, as with Finished, Not for me and Endless (L2). They take no rating.
- Dead games stay in the Library (filterable by status). Settings links to a **Dead games** view that lists them, where the status can be changed back.

## Wheel
- Wheel size moves from Settings to a slider in the wheel's filter panel (12, 15, 18, 24, All). The last value is remembered.
- The Single-player, Co-op and PvP yes/no filters become one **Type** dropdown: All, Single-player, Co-op or PvP. It matches games that have that Steam player category.

## Window and layout
- The window opens at the width the layout is designed for (1728 × 1000) and is centred. On smaller screens it shrinks to fit the work area. The minimum size is 1280 × 720.
- Filter bars stay on one line. Library sort, Show hidden, Pick back up and Rows/Grid sit in the page header.
- Recommendation rows show as many cards as fit on one line, up to six (R5), instead of wrapping a sixth card onto a new line.

## Recommendation cards
- ~~A **To play** button sits between the status control and Not now.~~ Removed by ADR 0014.
- The action row is pinned to the bottom of the card, so every card in a row lines up.
- Matched tags show at most two lines. A "+N" chip shows the rest on hover or keyboard focus.
