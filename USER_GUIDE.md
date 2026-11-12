# So Many Games: user guide

So Many Games tracks your Steam library, suggests games you own but have barely played, and spins a wheel when you can't decide. It runs only on your PC and isn't affiliated with Valve.

**Contents:** [Before you start](#before-you-start) · [Install](#install) · [First run](#first-run) · [Statuses](#statuses) · [Screens](#screens) · [Hidden games](#hidden-games) · [Your data](#your-data) · [Uninstall](#uninstall) · [Troubleshooting](#troubleshooting)

## Before you start

You need:

- **Windows 10 or 11.**
- **Public game details on Steam.** In Steam, open your profile, choose **Edit Profile → Privacy Settings**, and set **Game details** to **Public**.
- **A Steam Web API key.** It's free:
  1. Sign in at <https://steamcommunity.com/dev/apikey>.
  2. Enter any domain name, such as `localhost`. The app doesn't use it.
  3. Agree to the terms and copy the 32-character key.

Keep the key private. The app stores it only in Windows Credential Manager, never in its own files, backups or exports.

## Install

1. Run `somanygames-amd64-installer.exe`. It installs for your Windows user only, so there's no admin prompt.
2. Open **So Many Games** from the Start menu.

**Windows warning:** the installer isn't code-signed, so Windows may say "Windows protected your PC". Choose **More info → Run anyway**.

**Updating:** in **Settings → Updates**, choose **Check for updates**, then **Install and restart**. The app downloads the new version, checks its signature, closes and reopens updated. Your data is kept. The app only contacts GitHub when you choose this. If you use the portable `somanygames.exe`, the button opens the release page so you can download it yourself. Versions before 1.5.0 don't have the button, so run a newer installer instead.

## First run

1. **Connect Steam.** Paste your API key and your profile URL (`steamcommunity.com/id/yourname` or `steamcommunity.com/profiles/7656…`), then choose **Connect and import library**.
2. **Classify.** Games you've played need a status the first time. Each one gets a suggestion: **Playing** if you played it recently, otherwise **Shelved**. Fix any that are wrong, then choose **Save all**. Games you've never played start as **Unplayed**.
3. **Fetch store data.** Choose **Fetch store data** on the banner to download tags, genres, reviews and cover art. It takes about 1.5 seconds per game; you can keep using the app while it runs.

After that, your library syncs from Steam every time you open the app. To sync sooner, choose **Sync now** in Settings.

## Statuses

Every game has one status. Change it with the row of icons on a game (hover an icon to see its name). Each change shows an **Undo** button for a few seconds.

| Status | Meaning |
| --- | --- |
| Unplayed | Owned, never started |
| To play | Next on your list |
| Playing | In progress |
| Shelved | Started, then put down |
| Finished | Done. You can also tick **Completed** and rate it 1–10 |
| Not for me | Tried it, won't go back |
| Endless | No ending, such as sandboxes or multiplayer you dip into |
| Dead | Can't be played any more, such as when servers have shut down |

**Automatic changes:** when Steam shows new playtime on an Unplayed, To play or Shelved game, the app moves it to Playing. It never changes Finished, Not for me, Endless or Dead games.

## Screens

| Screen | What it does |
| --- | --- |
| **Home** | Your Playing games, a reminder for ones you haven't touched in a while, and your top recommendations |
| **Library** | Every game, with search, filters, sorting, and a list or grid layout. Tick several games to change their status or hide them at once |
| **Recommendations** | Games you own but haven't really played, matched to your taste |
| **Wheel** | A random pick from your unplayed and lightly played games |
| **Stats** | How many games you've finished, games by status, your backlog, hours by genre and tag, and your most played games |
| **Tier list** | Drag your finished games into S to F tiers |
| **Settings** | Steam profile, API key and sync; which games can be recommended; taste tags; status rules; hidden and dead games; backups |

### Home

**Shelved games** lists Shelved and Endless games you stopped playing, longest untouched first.

**Recently finished** lists your Finished games, newest first. Set the date on games you finished before you started using the app.

### Finished cards

When you set a game to Finished, a card pops up showing the game, the date, hours played and which number finished game it is. **Copy image** puts it on the clipboard, ready to paste into a chat. **Save PNG…** saves it as a file. Every Finished game has a **Card** button in the Library and in Recently finished, so you can get the card again or after changing the date.

**Share** on Stats (on the games finished counter and on the status breakdown) and on the Tier list gives the same kind of image of your progress, your library by status, or your ranked tiers.

### Tier list

Drag games from **Unranked** on the right into a tier, between tiers, or back. Tick **Include Endless games** to rank Endless games too. Your tiers are saved as you go.

### Recommendations

- **Tailored for you** shows the games that share the most of your taste tags.
- Each **Because you play …** row follows one of your top tags.
- **Refresh** shows a row's next games.
- **Not now** hides a card for a while.
- Click a cover to open the game's Steam store page.

**How your taste is worked out:** from your hours played. Highly rated games count for more, low-rated ones for less, and Not for me games are ignored. **All-time** uses your whole history; **Recent** uses the last 6 months (you can change this in Settings).

**Taste tags (Settings):** ignore generic tags, pin favourites, and avoid tags you never want recommended.

### Wheel

- **Filters:** tags (a game must have every tag you pick), genre, review score, type (single-player, co-op or PvP), and status (tick any of Unplayed, To play, Shelved and Endless). Endless games are left off the wheel unless you tick Endless.
- **Size:** choose how many games the wheel holds.
- **Bias toward my taste:** favours games like the ones you play.
- **After a spin:** mark the winner Playing or Not for me, or spin again. **Share** gives an "Up next" card for the game the wheel picked (cover, review score and tags) to copy or save, to announce your next game alongside the finished card for the last one.

## Hidden games

Demos, soundtracks, test servers, tools and other non-games are hidden automatically, by name and by Steam's store type. They don't count anywhere in the app.

- **See them:** **Settings → Hidden games**.
- **Unhide one:** choose **Not clutter**.
- **Hide one yourself:** also in **Settings → Hidden games**.
- **Change the name rules:** edit the patterns in Settings.

## Your data

- Everything is stored in `%APPDATA%\SoManyGames` on this PC. The app talks to Steam, and to GitHub only when you check for updates.
- **Back up** (Settings) saves a copy of the database.
- **Export** saves your statuses, ratings, finished dates, hidden games, tag settings, tier list and history to a JSON file. **Import** loads that file, for example on a new PC. Store data isn't exported; the app downloads it again.
- Backups and exports never include your API key.

## Uninstall

In Windows, go to **Settings → Apps → Installed apps → So Many Games → Uninstall**.

The uninstaller asks whether to delete your data and saved API key too. Choose **No** to keep them for a later reinstall.

## Troubleshooting

| Problem | Fix |
| --- | --- |
| "Steam returned no games" | Make your **Game details** public (see [Before you start](#before-you-start)), wait a few minutes, then choose **Sync now**. |
| "Steam rejected the API key" | Check the key at steamcommunity.com/dev/apikey, then enter it again in **Settings → API key → Change**. |
| "Steam is rate limiting requests" | Wait a few minutes and try again. The app slows down automatically. |
| Some games have no store data | Delisted or region-locked games have no store page. Open the failures list on the banner, then choose **Retry** or **Mark unavailable**. |
| A game you own is missing | Steam doesn't list free-to-play games you've never played, or games borrowed through Family Sharing. |
| "The update failed its security check" | The download didn't match the signed checksum, so nothing was installed. Try again later, or download the installer from the release page. |
| "GitHub is limiting update checks" | Wait an hour and check again. |
| "Your data was saved by a newer version" | You opened an older copy of the app. Install the latest version. |
| The taskbar shows an old icon | Windows cached the old icon. Restarting Windows usually fixes it. |
