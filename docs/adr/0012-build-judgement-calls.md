# 0012 Smaller decisions made during the build

Status: Proposed · 2026-11-07 · the requirements were silent on these; each is easy to change

| Area | Decision | Why |
| --- | --- | --- |
| Recommendation "opened" (E1) | Clicking a recommendation's cover opens its Steam store page in the default browser and logs `rec_opened`. | E1 needs an "opened" action; launching games is out of scope, but the store page is a natural "look closer". |
| Recommendation "dismissed" (E1) | "Not now" logs `rec_dismissed` and hides the card until the app restarts. It does not exclude the game permanently. | Permanent exclusion would duplicate Not for me and avoided tags. |
| `rec_shown` | Logged once per game per app run, not on every refresh. | Keeps the log meaningful for a future model. |
| Hidden-name patterns (L6) | Match whole words, case-insensitive, so "Demo" hides "X Demo" but not "Demolition". | Plain substring matching hid real games in testing. |
| Not clutter (L6) | Stored as a per-game override that survives pattern edits and syncs. A manual Hide is the opposite override. | The choice has to stick. |
| Taste-bias wheel (W2) | Weights by Wilson score of games matching the profile, ignoring the review minimum, with a small floor so unmatched games stay possible. | The bias should nudge, not filter. |
| Abandoned (N1) | Uses Steam's last-played date against the setting (default 60 days). Snapshot fallback dropped (ADR 0009). | Last-played is always present. |
| Mock and requirement mismatches | The Abandoned badge offers Shelved, Finished and Not for me (N1). Classify is its own sidebar item, with a count, while games remain (L8). | The requirements win over the mocks. |
| Library row layout | Completed and the rating sit on a line under the game name, so every row's status control lines up. | The mock put them beside the status control, which overflowed the window. |
| Hidden unclassified games | Not shown on Classify and not counted in the banner; they appear if un-hidden. | Hidden entries are excluded everywhere (L6). |
