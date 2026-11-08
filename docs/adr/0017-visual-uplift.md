# 0017 Visual uplift across the app

Status: Accepted · 2026-11-08 (owner decisions; mocks in `docs/spec/ui-mocks/uplift/`)

These changes carry the style of the new Stats screen and finished card (ADR 0016) to the rest of the app. They change how things look, not what they do.

- **Sidebar.** Each item has an icon. The current page has an amber bar on the left and a soft tint, replacing the solid amber block. A small card under "Not affiliated with Valve." shows finished out of all games and the backlog size, and opens Stats.
- **Home.** A strip across the top shows finished (with the share of the library), playing and backlog counts, each with its status icon, next to the Shelved games and Recently finished buttons. Playing cards show the cover over a blurred copy of itself, a Playing badge, hours and last played. The Abandoned nudge (N1) is a card with the cover.
- **Library.** Rows have a stripe and a faint tint in the status colour, the status name at the start of the sub-line, and an hours bar on a square-root scale. Grid tiles show the name over the cover and the status control, status name and hours below. The Library has no Shelved games button; that stays on Home only.
- **Recommendations.** The review score is a badge on the cover: green at 90% and up, light green from 80%, amber below. The review count is shortened ("766k reviews"). The taste profile is a row of numbered chips.
- **Wheel.** Wedges use the status palette, drawn faint, and the winning wedge lights up. The pointer and the centre Spin button glow amber. The pick shows as a card beside the wheel with the cover, score, hours, status and tags. Labels read outward from the centre: flipping them can't keep them upright once the wheel has spun.
- **Toasts.** A status change toast shows the status icon and name in the status colour, and the game's cover when it is one game.
- **Empty states.** Empty lists show a large status icon, a title, a sentence and, where useful, buttons to move on.
