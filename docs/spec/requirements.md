# So Many Games — Requirements

Nov 7, 2026 · @Luke

## Vision

So Many Games is a Windows desktop app that answers "what should I play next?" from your own Steam library and tracks what you're playing now. It pairs a status tracker (Playing, Shelved, Finished and so on) with a tag-based recommender and a filtered wheel that surface games you already own but have barely played.

Two jobs, one app:

- **Track:** see every game in progress at a glance and move games between statuses in one click.
- **Decide:** get recommendations and a wheel spin drawn from your unplayed and underplayed games.

Everything is stored locally in one SQLite database. The only network traffic is to Steam.

## Users

One person, on one Windows 10/11 PC, with one Steam account and roughly 260 games. How often they will open the app is unknown, so playtime history may be sparse and the app must not imply precision it doesn't have. They need a free Steam Web API key and a profile whose "Game details" are public.

## Out of scope for v1

v1 touches only games you own and their Steam store data. Not in v1:

- Wishlist and anything about unowned games (see Deferred and future features)
- Non-Steam games and manual game entries
- Notes and custom tags (Steam genres and tags only)
- Achievements, launching games, installed-game detection
- Friends, co-op matching and other social features
- AI and local-model recommenders (the design leaves room)
- Year in review
- Multiple Steam accounts (each install has exactly one account, set up on first launch), other platforms
- Updating from inside the app (see Deferred and future features); a newer installer upgrades in place

## Functional requirements: setup, sync and store data

Each requirement is a short story, then an EARS statement (When / While / If ... the app shall ...).

- **S1 First-run setup.** As the owner, I want to connect Steam once. *When the app starts with no saved credentials, it shall ask for a Steam Web API key and a profile URL or SteamID, test the connection, and, if the owned-games list comes back empty, explain that "Game details" must be public.*
- **S2 Key storage.** As the owner, I want my key kept safe. *The app shall store the API key in the Windows Credential Manager only; the database and the backup download shall never contain it.*
- **S3 Launch sync.** As the owner, I want my library list current without effort. *When the app launches with saved credentials, it shall fetch the owned-games list and store a playtime snapshot, but shall not fetch store data automatically. A Sync now button shall do the same on demand. The app shall never poll in the background.*
- **S4 Missing-data banner.** As the owner, I want to see which games lack store data. *While any visible game has no store data, the app shall show a warning icon and "No data for N games" with a Fetch button.*
- **S5 Fetching store data.** As the owner, I want to fetch data on my terms. *When I press Fetch, the app shall retrieve genres, tags, categories, review summary, release date and cover art for each game without data, one at a time and rate-limited, with a progress bar showing the current game and x of y, and one button that toggles pause and resume.*
- **S6 Failures.** As the owner, I want to recover from failed fetches. *If a fetch fails for a game, the app shall list it with the reason, a Retry button, a Mark unavailable button, and a Retry all button. A game marked unavailable shall never be fetched again and shall not count in the S4 banner until I undo the mark from its Library row.* (ADR 0003)
- **S7 Permanent cache.** As the owner, I want fetched data kept. *The app shall never expire stored store data. A manual Re-fetch shall refresh selected games or all games.*
- **S9 Source fallbacks.** As the owner, I want the app to survive Steam changes. *Each Steam data source shall sit behind a swappable provider with a fallback. If no tag source works, the app shall use Steam genres and categories only.*

## Functional requirements: library, statuses and classification

- **L1 Statuses.** As the owner, I want every game to have one status. *The app shall give each game exactly one of: Unplayed, To play, Playing, Shelved, Finished, Not for me, Endless, Dead. Shelved means started and set aside. Dead means it can no longer be played, for example servers shut down (ADR 0013). Finished carries a Completed checkbox: unchecked means finished with it, checked means completed (credits or 100%). Both take the 1–10 rating.*
- **L2 Automatic changes.** *When a sync first sees a game with 0 minutes played, the app shall set Unplayed. When a sync shows new playtime on an Unplayed, To play or Shelved game, it shall set Playing. It shall never change Finished, Not for me, Endless or Dead automatically, and never set Unplayed on a game it has seen before.* (ADR 0002)
- **L3 One-click status.** As the owner, I want to change status in one click. *The app shall show the eight statuses as icon radio buttons, with the label on hover, on every row card (Library, Home, Classify). Cards in the compressed grid and in recommendation rows shall show the current status icon, which expands to the radio buttons on hover. A change shall apply at once with an undo toast, except in Classify, where changes are staged.*
- **L4 Rating.** *When Finished is chosen, the app shall reveal an optional 1-10 rating on the card, skippable and editable at any time. No other status shall ask for a rating.*
- **L5 Library screen.** As the owner, I want to browse and filter everything. *The Library shall default to row cards, with a toggle for a compressed grid. It shall offer search by name; filters for status, tags, genres, hours range and review score; sorting by name, hours, last played, review score and release date; multi-select with bulk status change; and a show-hidden toggle.*
- **L6 Hidden entries.** As the owner, I want test servers and playtests out of the way. *The app shall flag an entry as hidden when its name matches an editable pattern list (Test Server, Public Test, Playtest, Unstable, Beta, Dedicated Server, SDK, Soundtrack, Demo) or its store type is not a game. Hidden entries shall be excluded from counts, stats, the wheel, recommendations and the taste profile. A Hidden view shall list them, each with Not clutter and Hide actions.*
- **L7 First-import classification.** As the owner, I want to sort my existing library quickly. *On the first import, games with 0 minutes shall become Unplayed. All others shall appear on a Classify screen in two sections: played in the last 30 days (window adjustable on the screen; default status Playing) and the rest sorted by hours descending (default Shelved). Each card shall carry the L3 icon radios, and a Save all button shall apply the staged statuses.*
- **L8 Unclassified state.** *Until Save all is pressed, imported games shall be Unclassified: left out of the wheel, recommendations and stats, with a banner on Home showing how many remain. The Classify screen shall stay in the sidebar until none remain. Games first seen in later syncs shall skip classification and follow L2.*

## Functional requirements: Home, recommendations and the wheel

- **H1 Home dashboard.** As the owner, I want to see what I'm playing. *Home shall show each Playing game as a card with cover art, hours played, no progress bar, and the L3 status icons.*
- **H2 Home extras.** *Home shall also show the Unclassified banner (L8), the Abandoned badge (N1), and a preview of the top two recommendation rows.*
- **R1 Taste profile.** As the owner, I want recommendations based on what I play. *The app shall build a profile of the top 5 tags by hours played, dampened (log or square-root scale) so one very long game cannot dominate, and scaled by rating, with Not for me games left out (ADR 0015), in All-time and Recent views. Recent counts the games Steam says were last played within the recent window (default 6 months), weighted by their lifetime hours (ADR 0009). A default stoplist of generic tags (such as Singleplayer, Multiplayer, Indie, Action, Adventure, Casual, 2D) shall be ignored and editable. Hidden games shall not count.*
- **R2 Pins and avoids.** *Settings shall let me pin tags (always part of the profile, so tastes formed outside Steam still count) and avoid tags (games with them are never recommended but stay in the library).*
- **R3 Eligible games.** *The app shall recommend and put on the wheel only Unplayed and To play games at any hours, plus Shelved and Endless games with under 10 hours played (adjustable). Finished, Not for me, Dead and Playing games shall never be recommended, however few hours they have.*
- **R7 Pick back up.** As the owner, I want to rediscover games I drifted from. *A Pick back up view (labelled "Resume shelved games", ADR 0014), reachable from Home and the Library, shall list every Shelved and Endless game at any hours, sorted by longest since last played (switchable to most hours), with the L3 icon radios and the matched tags. It is not limited by the R3 hours limit.*
- **R4 Ranking.** *A game shall qualify by matching a profile tag, and candidates shall be sorted by a confidence-adjusted review score (Wilson lower bound) with a minimum of 50 reviews. Each card shall show the raw percentage, review count and matched tags.*
- **R5 Layout.** *Recommendations shall open with a "Tailored for you" row of up to 6 games sharing the most taste tags (at least 2), followed by rows grouped by profile tag ("Because you play ..."), up to 6 games each and only as many as fit on one line. A game shall appear once on screen: in Tailored for you, or else in its best-fitting tag row. Each section shall have a Refresh button that shows its next games (ADR 0014). Each card shall keep its actions on the bottom edge, and show at most two lines of tags with the rest behind a "+N" on hover (ADR 0013).*
- **R6 Swappable engine.** *Recommendations shall come from one interface (profile and candidates in; ranked games with reasons out) so the tag-based engine can later be replaced or joined by others.*
- **W1 Wheel pool and filters.** As the owner, I want a random pick from what I'd actually play. *The wheel shall draw from the R3 pool, narrowed by tags (several allowed; a game must have all of them), genre, review score, and a Type of single-player, co-op or PvP from Steam's player categories. Every filter shall have an All option, which is the default, so an unfiltered wheel draws from the whole pool.*
- **W2 Wheel size and bias.** *The wheel size shall be a slider on the wheel's filter panel: 12, 15, 18, 24 or All, remembered between runs. The wheel shall be drawn as soon as the filters or size change, and Spin shall pick one of the games shown (ADR 0014). The draw shall be plain random, with an optional bias-toward-my-taste switch. For All, the app shall show a spinning name reel instead of wedges.*
- **W3 Result actions.** *After a spin the app shall offer Mark Playing, Spin again and Not for me.*

## Functional requirements: stats, nudges, settings and data

- **T1 Stats.** As the owner, I want to see my library's shape. *The Stats screen shall show the status breakdown (including backlog size and its total hours played), hours by top tags and genres (all-time and recent), games finished per year, rating distribution, most played, and a backlog burn-down. Time-based charts shall be labelled "since first sync". The backlog is the R3 pool (ADR 0008). Finished per year counts the date a game was set to Finished; games set to Finished on the Classify screen go in a "Before first sync" bar (ADR 0007).*
- **N1 Abandoned nudge.** As the owner, I want to notice games I've dropped without deciding. *When a Playing game has not been played for 60 days (adjustable), the app shall show an in-app Abandoned? badge with one-click moves to Shelved, Finished or Not for me. It shall not send Windows notifications. It uses Steam's last-played date (ADR 0009).*
- **G1 Settings.** *Settings shall hold the API key, sync controls, the recommendation hours limit (10), minimum reviews (50, ADR 0010), recent window (6 months), abandoned days (60), classify window (30 days), tag stoplist, pinned and avoided tags, hidden-name patterns, links to the Hidden and Dead games views, and the backup button.*
- **B1 Backup.** As the owner, I want my data safe. *A Back up button shall download the database. The app shall also copy the database before each schema upgrade, keeping the last 3 copies.*
- **X1 Export and import.** As the owner, I want to move my data between machines and keep a readable copy. *Export shall write one JSON file holding statuses, ratings, hidden flags, tag settings and the event log. Import shall read that file, show a summary of what will change, and ask before replacing the current data. The API key is never exported. Store metadata and cover art are not exported; they are re-fetched.*
- **E1 Event log.** As the owner, I want my history kept for future features. *The app shall record, with timestamps: status changes, ratings, playtime snapshots, and which recommendations were shown, opened or dismissed.*

## Non-functional requirements and ground rules

This is a personal, single-user tool, so simple and maintainable beats clever. Items marked (proposed) are my suggestions, not decisions you made.

**Platform and stack**

- Windows 10/11 x64 only. One window, opening at the layout's designed width and shrunk to fit smaller screens (ADR 0013); closing it quits the app. No tray icon, no background service, no auto-update.
- Wails v3 (exact version pinned) with a React and TypeScript frontend; Go backend; SQLite through the pure-Go `modernc.org/sqlite` driver; `go-keyring` for the Credential Manager; NSIS installer.
- Dark theme by default.

**Behaviour**

- The UI never blocks on the network. Steam requests are rate-limited, with backoff on errors.
- Fully usable offline from the cache. The only network traffic is to Steam. No telemetry.
- Stays within the Steam Web API Terms of Use and any third-party API rules: the key is used only where required, requests are paced with backoff, the client identifies itself honestly, there is no scraping or getting around gates, stored data is disclosed, and the app does not claim affiliation with Valve (ADR 0011).
- (proposed) Library views, filters and sorts respond in under 100 ms for 300 games; cold start under 3 seconds.
- (proposed) Every control is keyboard reachable; icon-only controls carry text labels for hover and screen readers.

**Ground rules**

- Favour simple, maintainable implementations; no overengineering.
- Concise comments and docs, without over-defensiveness.
- Follow existing code patterns; avoid code smells even where a workaround is simpler.
- No local-versus-production split is needed.

## Deferred and future features

The design leaves room for all of these; none is built in v1.

**Wishlist (deferred as one feature)**

- Read-only sync of the Steam wishlist on each sync, badged Steam
- Add games by Steam store link or name search, badged Added by me
- Priority, date added and an optional "why" on each wanted game
- Price in your region, a target price, and a flag when it is reached (checked on launch or refresh, no polling)
- Release reminder for unreleased wishlist games
- Automatic move into the library when a wanted game becomes owned, starting as To play
- A view for "you wished for this, but you already own similar games"

**Discovery and recommenders**

- A discovery engine that adds a couple of unowned games to the recommendation rows
- Tag combinations beyond Tailored for you (ADR 0014), which ranks by how many taste tags a game shares: learning which pairs the owner plays together (for example Roguelike with Automation) and weighting those
- An AI recommender using the user's own API token
- A local recommender model trained on the event log; it can read the SQLite file from outside the app

**Time-to-beat (deferred until a suitable API is found, ADR 0005)**

- Estimated hours to beat on Home and Library cards, blank where a lookup fails
- A time-to-beat range filter on the wheel
- Candidate source: IGDB with the owner's own credentials. HowLongToBeat has no public API and gates its search against bots, so it is not used.

**Updates**

- A **Check for updates** button in Settings, run by the owner rather than on every launch. It asks GitHub's releases API for the latest version and, if newer, downloads the installer, verifies it against a published SHA-256 checksum, runs it silently and closes. The per-user installer upgrades in place without admin rights and keeps all data.
- Needs: releases published to GitHub with the installer and checksum; the "no auto-update" non-functional requirement and PRIVACY.md updated, since the app would then contact GitHub as well as Steam; an ADR. The installer is unsigned, so Windows SmartScreen warns on first run unless it is code-signed.

**Other**

- Year in review
- Achievements progress, launching games and installed-game detection
- Notes, custom tags, non-Steam games, friends and co-op

## Spike and open risks

The spike ran before the build (findings in `docs/spike/findings.md`). It confirmed the Steam endpoints against the owner's 256-game list and produced a working Wails v3 installer. Tags and last-played dates turned out to be available (ADR 0006, ADR 0009).

**Risks that remain**

- **Undocumented store endpoints.** The store browse API and appdetails are public but undocumented, so Steam may change them. They stay behind the provider interface (S9).
- **Wails v3 is young.** Versions are pinned (ADR 0001). Fallback: Wails v2 with the same frontend and backend.
- **Playtime history is sparse.** Snapshots exist only for the days the app was used, so charts are labelled accordingly.
