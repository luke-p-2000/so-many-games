# 0007 Games finished per year counts from status changes

Status: Accepted · 2026-11-07 (owner decision)

The year of a finish is the year of the `status_changed` event that moved the game to Finished. Games set to Finished on the Classify screen (L7) have no real finish date, so they go in one bar labelled **Before first sync**, never in the year of the import. If a game is finished again after being moved away, its latest Finished event counts.
