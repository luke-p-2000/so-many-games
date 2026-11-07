# 0011 Staying within the Steam Web API terms

Status: Accepted · 2026-11-07 (owner asked that all Steam Web API terms are respected)

## Context
These are the Steam Web API Terms of Use (steamcommunity.com/dev/apiterms), as they apply to this app:
- The key is confidential and personal.
- At most 100,000 calls a day.
- Only fetch data about a user at that user's request.
- Tell the user what Steam data is stored, and post a privacy statement.
- Don't present the app as endorsed by or affiliated with Valve.
- Provide Steam data "as is".
- Delete Steam data if API access ends.

## Decision
| Rule | How the app meets it |
| --- | --- |
| Key confidential | Stored only in Credential Manager. Sent only to the methods that need it (`GetOwnedGames`, `ResolveVanityURL`); the store methods are called without it. Scrubbed from every error, log and event. Never in the database, exports or backups (S2). |
| Volume | One owned-games call per launch or Sync now. Metadata is fetched only on Fetch, paced at least 1.5 s per host, with backoff on HTTP 429. A full first fetch of about 260 games is a few hundred calls. |
| Own data only | The app fetches only the owner's library, and only at launch or when the owner presses a button (S3). No background polling. |
| Honest client | User agent `SoManyGames/<version>`. No browser impersonation, no page scraping, no getting around age gates or bot gates. |
| Disclosure and privacy | Onboarding and Settings say which Steam data is stored and where (one SQLite file and the cover images in `%APPDATA%\SoManyGames`). `PRIVACY.md` in the repo states the data never leaves this PC except store-data requests, which send only app ids to Steam. |
| No implied endorsement, "as is" | Settings has an About line: "Not affiliated with Valve. Steam data is provided as is." The app uses no Valve logos. |
| Deletion | The uninstaller offers to delete the app-data folder and the Credential Manager entry. |

## Consequences
These are small additions to onboarding (S1), Settings (G1) and the installer. Build step 7 checks them.
