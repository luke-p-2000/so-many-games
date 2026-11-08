-- Finished date (ADR 0016): a local calendar date, YYYY-MM-DD. NULL means unknown, for
-- games finished on the Classify screen or at import until the owner enters a date.
ALTER TABLE game ADD COLUMN finished_at TEXT;

-- Backfill from each Finished game's latest move to Finished, as ADR 0007 counted it.
UPDATE game SET finished_at = (
    SELECT CASE WHEN json_extract(e.payload, '$.source') IN ('classify', 'import') THEN NULL
                ELSE substr(e.at, 1, 10) END
    FROM event e
    WHERE e.appid = game.appid AND e.kind = 'status_changed' AND json_extract(e.payload, '$.to') = 'finished'
    ORDER BY e.at DESC, e.id DESC
    LIMIT 1
)
WHERE status = 'finished';

-- The finished-games tier list (T2). Placements outlive leaving Finished, so a game
-- finished again returns to its spot.
CREATE TABLE tier_entry (
    appid    INTEGER PRIMARY KEY REFERENCES game (appid) ON DELETE CASCADE,
    tier     TEXT    NOT NULL,  -- S, A, B, C, D or F
    position INTEGER NOT NULL   -- order within the tier, from 0
);
