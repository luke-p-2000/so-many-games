# 0006 Metadata providers: store browse API first, no scraping

Status: Accepted · 2026-11-07 (owner decisions: drop scraping, drop SteamSpy)

## Evidence
From the probe run of 2026-11-07 against the owner's library (231 owned games). Raw output was in `spike/probe/out/`, removed with the spike after V1.

| Source | Coverage | Notes |
| --- | --- | --- |
| `IStoreBrowseService/GetItems/v1` (keyless, 50 per call, 5 calls) | 226 of 231 found. Tags on 224, reviews on 226, release date and categories on 226. | The 2 found with no tags are playtests. The 5 not found (`success: 15`) are 3 test servers plus 2 delisted games (Age of Empires Online, Deiland), and they come back with `appid: 0`, so match them by request order or `id`. `type` is 0 for games and 12 for playtests. |
| `IStoreService/GetTagList/v1` (keyless) | 446 tag names | Maps tag id to name. |
| `store/api/appdetails` (1 per call, paced) | 225 of 231 | The only source of genres. It calls playtests `type: "game"`, so it can't be used for the L6 non-game check. |
| `store/appreviews` | 0 | Redirects to the store home page. Dropped; GetItems has reviews. |
| SteamSpy `appdetails` | Tags on 204 of 231 | Adds tags for exactly one game GetItems lacks (Deiland, delisted). Has no tags for recent releases. Third-party. |
| Store-page scraping | Not used | Dropped (owner decision). |

## Decision
1. **GetItems** is the main metadata provider: tags (with weights), review % and count, release date, store type, player categories and cover art.
2. **appdetails** is called per game for genres only, at least 1.5 s apart. A game whose appdetails fails still keeps its GetItems data.
3. **SteamSpy is not used.** In this library it filled one game, and it would be the only non-Steam traffic. The S9 fallback for a game with no tags is its genres and categories. The `TagProvider` interface stays, so a provider can be added later.
4. L6 hides by store type using GetItems `type != 0`, not appdetails `type`.
5. Category ids map to the wheel flags: single-player 2; co-op 9, 38, 39, 48; PvP 49, 36, 37, 47.

## Consequences
A full first fetch is about 5 GetItems calls plus about 231 appdetails calls, roughly 6 minutes at 1.5 s per call. The S5 progress bar follows the appdetails pass. Games GetItems can't find show up in the failures list for the owner to mark unavailable (ADR 0003).
