# 0018 Share cards for Stats and the tier list, and Endless off the wheel by default

Status: Accepted · 2026-11-09 (owner decisions, from friends' suggestions; mocks in `docs/spec/ui-mocks/share-cards/`)

## Share cards
- Three more images to copy or save, drawn on a canvas in the frontend like the finished card (ADR 0016), with the same dark frame, So Many Games logo and fonts, at twice their size. No Steam branding (ADR 0011).
  - **Games finished** (1200 × 630): the finished count out of every game, the percentage, a striped progress bar with quarter marks, and a footer with the latest dated finish (cover, name, date), finishes this calendar year and how many are left.
  - **Status breakdown** (1200 × 630): the stacked status bar, a tile per status (icon, count, share; dimmed at zero) and the backlog line.
  - **Tier list** (1200 wide, as tall as the list needs): the S–F rows as on the Tier list screen, with every tier shown even when empty, and a count of ranked games. Unranked games are left out.
- A **Share** button on the finished counter, the Status breakdown panel and the Tier list page opens a dialog with the preview, **Save PNG…** and **Copy image**. The tier list's Share is off until a game is ranked.

## Endless on the wheel
- With no status ticked, the wheel draws from the R3 pool **without Endless games**: they rarely end, so they made poor random picks. Ticking Endless (alone or with others) adds them. The filter's reset reads "All but Endless". This amends W1.
