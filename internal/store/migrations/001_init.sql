-- One row per owned game. Status is stored, never derived (design: Data model).
CREATE TABLE game (
    appid            INTEGER PRIMARY KEY,
    name             TEXT    NOT NULL,
    status           TEXT    NOT NULL,
    completed        INTEGER NOT NULL DEFAULT 0,  -- only meaningful when status = 'finished'
    rating           INTEGER,                     -- 1-10, only when status = 'finished'
    hidden           INTEGER NOT NULL DEFAULT 0,  -- effective hidden flag (L6)
    hidden_reason    TEXT    NOT NULL DEFAULT '', -- e.g. 'Playtest', 'not a game', 'manual'
    hide_override    TEXT    NOT NULL DEFAULT '', -- '' automatic, 'show' (Not clutter), 'hide' (manual)
    classified       INTEGER NOT NULL DEFAULT 1,  -- 0 until Save all on the Classify screen (L8)
    meta_unavailable INTEGER NOT NULL DEFAULT 0,  -- owner marked: never fetch store data (ADR 0003)
    first_seen_at    TEXT    NOT NULL
);

-- A snapshot is written on first sight and whenever playtime or last-played changes.
CREATE TABLE playtime_snapshot (
    appid          INTEGER NOT NULL REFERENCES game (appid) ON DELETE CASCADE,
    taken_at       TEXT    NOT NULL,
    minutes_total  INTEGER NOT NULL,
    last_played_at TEXT,  -- NULL when never played
    PRIMARY KEY (appid, taken_at)
);

CREATE VIEW latest_snapshot AS
SELECT s.appid, s.minutes_total, s.last_played_at
FROM playtime_snapshot s
WHERE s.taken_at = (SELECT MAX(taken_at) FROM playtime_snapshot WHERE appid = s.appid);

-- Absent row means "No data" (S4).
CREATE TABLE game_meta (
    appid        INTEGER PRIMARY KEY REFERENCES game (appid) ON DELETE CASCADE,
    fetched_at   TEXT    NOT NULL,
    release_date TEXT,
    review_pct   INTEGER,
    review_count INTEGER,
    is_game      INTEGER NOT NULL,  -- store type 0; anything else is hidden (L6)
    cover_path   TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE game_tag (
    appid  INTEGER NOT NULL REFERENCES game (appid) ON DELETE CASCADE,
    kind   TEXT    NOT NULL,  -- 'tag', 'genre' or 'category'
    name   TEXT    NOT NULL,
    weight INTEGER,           -- store tag weight; NULL for genres and categories
    PRIMARY KEY (appid, kind, name)
);

-- Latest failed fetch per game, listed with Retry (S6).
CREATE TABLE fetch_failure (
    appid  INTEGER PRIMARY KEY REFERENCES game (appid) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    at     TEXT NOT NULL
);

-- JSON values; never holds the API key (S2).
CREATE TABLE setting (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Append-only history (E1). Payload is JSON so new kinds need no migration.
CREATE TABLE event (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    at      TEXT    NOT NULL,
    kind    TEXT    NOT NULL,
    appid   INTEGER,
    payload TEXT    NOT NULL DEFAULT '{}'
);
CREATE INDEX event_kind_at ON event (kind, at);
