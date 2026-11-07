# So Many Games — Design

Nov 7, 2026 · @Luke

A Wails v3 desktop app: a React frontend over a Go backend that owns a single SQLite file, with Steam access hidden behind provider interfaces so fragile sources can fail without breaking the app. It implements the [requirements doc](https://claude.ai/artifact/Y2BFf3NgwqUpeBwtggmrfq); requirement ids (L3, R2, ...) refer to it.

## Architecture

The frontend never touches Steam or the database; it calls Go services over Wails bindings. Each backend package has one job.

| Package | Responsibility |
| --- | --- |
| `store` | SQLite access (`modernc.org/sqlite`), migrations, pre-upgrade backups (B1) |
| `steam` | Owned-games provider (Web API) and the key in Credential Manager (`go-keyring`) |
| `meta` | Store-metadata providers with fallback order, fetch queue with pause and retry (S5–S7, S9) |
| `library` | Status rules, classify (triage) logic, hidden-name patterns, automatic status changes (L2, L7, L8) |
| `recommend` | Taste profile, candidate pool, ranking engine interface (R1–R6) |
| `wheel` | Pool filters and the random draw (W1–W3) |
| `events` | Append-only event log (E1) |
| `transfer` | Backup download, JSON export and import (B1, X1) |
| `stats` | The Stats screen figures (T1), including the backlog snapshot logged after each sync |
| `app` | Thin Wails services the frontend calls; wiring only, no rules |

Rules: services take their dependencies through constructors; providers and the ranking engine are Go interfaces so they can be swapped or faked in tests; long jobs (sync, fetch) run in goroutines and report progress to the UI with Wails events; no network calls outside `steam` and `meta`. The app has no tray, background service or auto-update, and closing the window quits.

## Data model

One SQLite file in the app-data folder; cover art is cached beside it as files. Migrations are numbered SQL files run at start, after copying the database (last 3 kept).

| Table | Key columns |
| --- | --- |
| `game` | `appid` (PK), name, status, completed (only when Finished; unchecked means finished with it), rating (1–10, only when Finished), hidden, hidden\_reason, hide\_override (Not clutter / manual hide), classified, meta\_unavailable (ADR 0003), first\_seen\_at |
| `playtime_snapshot` | appid, taken\_at, minutes\_total, last\_played\_at (nullable) |
| `game_meta` | appid, fetched\_at, release\_date, review\_pct, review\_count, is\_game (store type 0), cover\_path |
| `fetch_failure` | appid, reason, at: the latest failed fetch per game (S6) |
| `game_tag` | appid, kind (genre, tag, category), name, weight (nullable; the GetItems tag weight) |
| `setting` | key, value (JSON); never the API key |
| `event` | id, at, kind, appid (nullable), payload (JSON) |

Notes:

- Status is stored, not derived. Automatic changes (L2) run after each sync and write an event.
- `game_meta` row absent means "No data" (S5). A game stays in the banner count until a fetch succeeds or the owner marks it unavailable (`meta_unavailable`, ADR 0003). Marked games are never queued.
- Hours and last-played are read from the latest snapshot. The "recent" views, the classify split and Abandoned use `last_played_at`, which Steam returns for every game (ADR 0009).
- Event kinds: `status_changed`, `rated`, `playtime_snapshot`, `backlog_snapshot` (ADR 0008), `rec_shown`, `rec_opened`, `rec_dismissed`. The payload is JSON so new kinds need no migration. A future local model reads this table directly.

## Steam and store providers

Two provider families, both behind interfaces. Endpoints were confirmed by the spike (docs/spike/findings.md, ADR 0006). All calls follow ADR 0011: the key only where required, an honest user agent, at least 1.5 s per host, and backoff on 429.

- **Library provider.** `IPlayerService/GetOwnedGames/v1` (key; `include_appinfo=1`, `include_played_free_games=1`) `-> []Owned{appid, name, minutes, lastPlayed}`. `rtime_last_played` is present for every game, and 0 means never played. `ISteamUser/ResolveVanityURL/v1` turns a profile URL into a SteamID at setup. Runs at launch and on manual sync; each run stores a snapshot and adds new games as Unclassified (first import) or applies L2 (later). No polling. Unplayed free-to-play games and family-shared games are not returned, so they are not in the library.
- **Metadata providers.** `Fetch(appids) -> []Meta`.
  1. `IStoreBrowseService/GetItems/v1` (keyless, 50 appids per call) gives tags (with weights), review % and count, release date, store `type` (0 = game), player category ids and asset paths for cover art. `success: 15` means not found.
  2. `store.steampowered.com/api/appdetails` (keyless, one appid per call) gives genres only.
  3. `IStoreService/GetTagList/v1` maps tag ids to names, cached in the database.

  If a game has no tags, its genres and categories are used (S9). SteamSpy is not used and the store page is never scraped (ADR 0006).
- **Fetch queue.** Games without `game_meta` and not marked unavailable are queued in library order. GetItems runs in batches, then appdetails runs one game at a time. One worker, progress events (current game, x of y), and one pause/resume button. Failures are kept with the error and listed with Retry, Mark unavailable (ADR 0003) and Retry all. Re-fetch of a single game replaces its metadata.
- **Key handling.** The API key lives only in Credential Manager. Settings shows it masked; export and backup never include it.
- **Errors.** Private profile, bad key and rate limiting are distinct errors with plain messages; none blocks the rest of the app.

## Recommender and wheel

Both read one shared pool so they can never disagree about what is eligible.

**Pool (R3).** Not hidden, classified, and not Finished, Not for me, Dead or Playing; either Unplayed or To play at any hours, or Shelved or Endless under the hours limit (10, adjustable). The Pick back up view (R7) does not use this rule: it lists every Shelved and Endless game at any hours, ordered by last played.

**Engine interface.**

```go
type Engine interface {
    Recommend(p Profile, pool []Candidate) []Ranked // Ranked carries game, score, reasons
}
```

`Profile` holds weighted tags (top 5 by dampened hours times rating / 5.5, Not for me excluded, ADR 0015; all-time or last 6 months), pinned and avoided tags, and the stoplist. The built-in engine does the following:

1. Drop stoplist tags; add pinned tags; remove candidates with an avoided tag.
2. Skip candidates with fewer than 50 reviews.
3. Score each candidate by the Wilson lower bound of its review score.
4. Take up to 6 games sharing at least 2 profile tags, most shared first, as Tailored for you (ADR 0014).
5. Group by profile tag. Each section holds up to 30 games and shows what fits on one line (at most 6), with Refresh for the next batch; each card shows raw %, review count and matched tags.

The AI and local-model engines later implement the same interface, receiving the profile and pool and returning ranked games with reasons. Shown, opened and dismissed recommendations go to the event log.

**Wheel.** Filters (tags, all of which must match; genre; review score; and Type: single-player, co-op or PvP) each default to All and narrow the pool by AND. `Draw` fills the wheel with the configured size (12, 15, 18, 24 or All) at random whenever filters or size change; `Pick` spins it, choosing among the games shown. The taste-bias switch weights both by the recommender score. Result actions write status and event changes through `library`.

## Frontend

React and TypeScript, one route per sidebar item plus Classify, Pick back up (labelled Resume shelved games), Hidden and Dead games; the mocks are the visual reference (dark theme, Inter and Space Grotesk).

- **State.** Server state comes from Wails calls, held in a small query cache; events (sync progress, fetch progress) update it. No global store beyond that.
- **Status control.** One `StatusRadio` component with a `full` mode (eight icon radios, label on hover) and a `compact` mode (current icon, expands on hover). Library, Home, Recommendations and Classify all use it. Changes call `SetStatus` and show an undo toast; in Classify they update local staged state until Save all.
- **Classify screen.** Shown from the Home banner and the sidebar until no Unclassified games remain. Two sections (played within the window, rest by hours), with the window adjustable on the screen.
- **Library.** Row cards by default, compressed grid by toggle; search, filters, sorts, multi-select bulk status, show-hidden toggle.
- **Fetch banner.** "No data for N games" with Fetch, then a progress bar, a single pause/resume button, and a failures list with Retry and Retry all.
- **Accessibility.** Real buttons with labels on icon-only controls, contrast of at least 4.5:1, keyboard-operable radios.

## Spike and build order

**Spike first (nothing else starts until it reports).**

1. Probe the Steam endpoints with a real key: owned games and last-played, and each metadata source for tags and review summaries. Check against the 256-game library file, and record which provider chain works.
2. Build a hello-world on the pinned Wails v3 version with SQLite and Credential Manager, and produce a working NSIS installer on Windows. If Wails v3 blocks this, fall back to Wails v2 and say so.
3. Check whether a time-to-beat source is reachable at all; if not, drop that module for v1.

The spike's findings update the provider section of this doc before the build starts.

**Build order.**

1. Store, migrations, settings, key storage.
2. Library sync and the Classify flow (statuses, automatic changes, hidden patterns).
3. Metadata fetch queue and banner.
4. Library, Home and the status control.
5. Recommender and wheel.
6. Stats, Settings, backup, export and import.
7. Event log hooks (added with step 2 onward, verified here) and the installer polish.

Each step ends with the app running and its tests passing; tests cover the status rules, the pool, the Wilson ranking and the importer, with providers faked.

**Ground rules (from the requirements).** Personal tool, simple and maintainable, no overengineering, concise comments and docs, avoid code smells, follow existing patterns.
