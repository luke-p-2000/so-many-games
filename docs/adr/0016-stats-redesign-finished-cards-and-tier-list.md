# 0016 Stats redesign, finished dates and cards, and the tier list

Status: Accepted · 2026-11-08 (owner decisions; mocks in `docs/spec/ui-mocks/Stats.png` and `FinishedCard.png`)

## Stats (T1)
- A **finished counter** leads the screen: Finished games out of every classified, visible game, with the percentage, a progress bar and the latest dated finish. The Completed checkbox is not counted separately.
- The **status breakdown** is one bar split into coloured segments, plus a tile per status showing its icon, count and share.
- **Removed:** games finished per year, the backlog burn-down and the rating distribution. Syncs no longer log `backlog_snapshot` events; old ones stay in the log unread. This supersedes ADR 0007 and the burn-down part of ADR 0008.
- Hours by genre, hours by tag and most played stay, in one ranked-bar style. Most played shows cover thumbnails.

## Status colours
Each status has one colour (`--st-*` in `styles.css`), used across the app: status buttons, the wheel's status filter and Stats.

## Finished date
- `game.finished_at` holds a local calendar date. Moving a game to Finished sets it to today, and leaving Finished clears it (undo restores it). It can be edited on any Finished game to backfill.
- When this was added, existing Finished games got the date of their latest move to Finished. Games finished on the Classify screen or at import have no real date, so it stays unknown until the owner sets one.
- Export and import carry the date.

## Finished card
- A 1200 × 630 image (saved at twice that size) drawn in the frontend on a canvas, with no new dependencies. It shows the cover art, a finished checkmark, the finished date, hours played, "#N" (the game's place in finish order: oldest first, undated games first by name) and the So Many Games logo. It never uses Steam branding (ADR 0011).
- Setting **one** game to Finished opens a dialog with the card, the date and the rating, plus **Copy image** and **Save PNG…** (through the native save dialog). Bulk changes don't open it.
- Finished rows in the Library and on **Recently finished** have a **Card** button. Recently finished is opened from Home, next to Shelved games, and lists every Finished game, newest first.

## Tier list (T2)
- A sidebar tab with rows S, A, B, C, D and F, plus an **Unranked** panel on the right, which has a search box and scrolls.
- It holds Finished games. An **Include Endless games** checkbox (a setting) adds Endless games.
- Cards are the cover art with the name underneath. They are dragged with the browser's own drag and drop.
- Placements are stored in `tier_entry`. Each drop saves the whole list. A game that leaves Finished keeps its placement, hidden, and is back in place if finished again. Export and import carry the list; files from before it leave the list as it is.
