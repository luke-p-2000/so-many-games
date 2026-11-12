# 0021 In-app updates through the signed installer

Status: Accepted · 2026-11-12

## Context
Releases come every few days (v1.2.0 to v1.4.0 in four days), and updating meant finding the GitHub release and running the new installer by hand. The requirements deferred a **Check for updates** button, provided that it never runs by itself, that the app keeps no background service, and that Privacy says the app also contacts GitHub.

## Decision
- **The owner asks; nothing runs by itself.** Settings → Updates has **Check for updates**. It calls `GET api.github.com/repos/luke-p-2000/so-many-games/releases/latest`, which skips drafts and prereleases, and compares the tag with the running version as X.Y.Z numbers. Development builds don't check.
- **The installer does the upgrade.** **Install and restart** downloads `somanygames-amd64-installer.exe` to `%LOCALAPPDATA%\SoManyGames\update`, starts it with `/S /RELAUNCH` and quits. The installer waits up to 10 s for the app's exe to unlock, upgrades in place (per-user, no admin prompt, ADR 0001), then reopens the app. The next launch deletes the downloaded file. Swapping the exe from inside the app was rejected because it leaves the uninstaller's version and shortcuts stale. MSIX was rejected because it needs code signing.
- **Every download is verified against a signed checksum list.** The release workflow publishes `SHA256SUMS` and `SHA256SUMS.sig`, an ed25519 signature made with the `SOMANYGAMES_SIGNING_KEY` secret (`go run ./cmd/release sign`). The app has the public key built in (`internal/update/github.go`). It checks the signature before downloading the installer, and the installer's hash before renaming it into place. A failure deletes the download and installs nothing. A checksum on its own would only catch corruption, because whoever can replace the installer on GitHub can also replace the checksum. Files downloaded by Go get no "downloaded from the internet" mark, so SmartScreen never checks them, and the signature is the only gate.
- **The portable exe doesn't install.** When the app isn't running from `%LOCALAPPDATA%\Programs\So Many Games`, an available update offers **Open release page** instead.
- **An older app won't open a newer database.** `store.Open` refuses a database whose `user_version` is above the newest migration it knows, and shows a message box at startup. Before this, an older portable exe would have run against a newer schema.
- **One command sets the version.** `wails3 task release:bump VERSION=x.y.z` (or `go run ./cmd/release bump x.y.z`) writes `build/config.yml`, `info.json` and `wails_tools.nsh`, which the workflow checks against the tag.

## Consequences
- The app now contacts GitHub, only when the owner clicks. GitHub sees the IP address and the `SoManyGames/<version>` user agent. PRIVACY.md and the design's network rule say so.
- The signing key's private half must be kept safe outside the repo. If it's lost, a new key needs a manual reinstall of the release that carries the new public key. If it leaks, rotate it the same way.
- v1.5.0 is the first version that can update itself. Older versions update by running the new installer by hand.
- Unauthenticated GitHub API calls are limited to 60 an hour per IP address. The app reports this as "try again in an hour".
- Everything used is free. Code signing, which would also stop the SmartScreen warning on manual installs, stays optional.
