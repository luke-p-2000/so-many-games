import { useState } from "react";
import { errorText, SetupService } from "../api";

/** S1: connect Steam once. The key goes straight to Windows Credential Manager (S2). */
export function Onboarding({ onConnected }: { onConnected: () => void }) {
  const [key, setKey] = useState("");
  const [profile, setProfile] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const connect = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await SetupService.Connect(key, profile);
      setKey("");
      onConnected();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className="page" style={{ maxWidth: 720 }} onSubmit={connect}>
      <h1>Welcome — connect Steam</h1>
      <section className="panel" style={{ display: "flex", flexDirection: "column", gap: 10 }}>
        <label className="field" htmlFor="key">1 · Steam Web API key</label>
        <input id="key" className="input" type="password" autoComplete="off" spellCheck={false} placeholder="32-character key" value={key} onChange={(e) => setKey(e.target.value)} />
        <span className="faint">Get one at steamcommunity.com/dev/apikey (any domain name works). It is stored only in the Windows Credential Manager.</span>
      </section>
      <section className="panel" style={{ display: "flex", flexDirection: "column", gap: 10 }}>
        <label className="field" htmlFor="profile">2 · Profile URL or SteamID</label>
        <input id="profile" className="input" placeholder="steamcommunity.com/id/yourname" value={profile} onChange={(e) => setProfile(e.target.value)} />
        <span className="faint">Set "Game details" to Public in your Steam privacy settings.</span>
      </section>
      {error && <p className="warn-text" role="alert">{error}</p>}
      <button className="btn primary" type="submit" disabled={busy || !key || !profile}>{busy ? "Connecting…" : "Connect and import library"}</button>
      <p className="faint">
        Stored on this PC only, in %APPDATA%\SoManyGames: your owned games, playtime and last-played dates, store details
        (tags, genres, reviews, release dates, cover art) and the statuses you set. Nothing is sent anywhere except requests to
        Steam. Not affiliated with Valve; Steam data is provided as is.
      </p>
    </form>
  );
}
