# Privacy

So Many Games is a single-user desktop app. It has no servers, accounts or telemetry.

## What it stores, and where

Everything is on this PC, in `%APPDATA%\SoManyGames`:

- the owned-games list for the Steam account you connect: names, playtime and last-played dates, with a snapshot each time they change;
- store details for those games: tags, genres, review summary, release date, player modes and cover art;
- what you enter: statuses, ratings, hidden choices, tag lists and settings;
- an event log of status changes, ratings, syncs and which recommendations were shown, opened or dismissed.

Your Steam Web API key is kept only in Windows Credential Manager. It never goes into the database, backups, exports or logs.

## What it sends

- Your API key and SteamID go to the Steam Web API to read your owned games, and to turn a profile URL into a SteamID.
- Game app IDs go to Steam's public store endpoints to fetch store details and cover art.
- When you choose **Check for updates** in Settings, the app asks GitHub for the latest release, and **Install and restart** downloads it from GitHub. GitHub sees your IP address and the app's version. The app never checks by itself.
- Nothing else leaves the PC.

## Removing it

Uninstalling offers to delete the data folder and the saved key. You can also delete `%APPDATA%\SoManyGames` and the `SoManyGames:steam-web-api-key` entry in Credential Manager yourself.

Not affiliated with Valve. Steam data is provided as is.
