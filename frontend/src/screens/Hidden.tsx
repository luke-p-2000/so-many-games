import { useState } from "react";
import { LibraryService } from "../api";
import { Cover } from "../components/bits";
import { useGames, useRefresh } from "../data";
import { useReportError } from "../toast";

/** L6: hidden entries with Not clutter, plus hiding a game the patterns missed. */
export function Hidden() {
  const games = useGames();
  const refresh = useRefresh();
  const report = useReportError();
  const [name, setName] = useState("");

  const all = games.data ?? [];
  const hidden = all.filter((g) => g.hidden);
  const candidates = all.filter((g) => !g.hidden);
  const match = candidates.find((g) => g.name.toLowerCase() === name.trim().toLowerCase());
  const override = (appid: number, o: string) => LibraryService.SetHideOverride(appid, o).then(refresh, report);

  return (
    <div className="page">
      <h1>Hidden</h1>
      <p className="subtle">Hidden entries are left out of counts, stats, the wheel, recommendations and the taste profile.</p>
      <div className="row-list">
        {hidden.length === 0 && <div className="panel empty">Nothing is hidden.</div>}
        {hidden.map((g) => (
          <article key={g.appid} className="game-row" style={{ gridTemplateColumns: "100px minmax(0,1fr) auto auto" }}>
            <Cover game={g} className="small" />
            <div className="name">{g.name}</div>
            <span className="subtle">{g.hiddenReason === "manual" ? "hidden by you" : `matched: ${g.hiddenReason}`}</span>
            <button className="btn" onClick={() => override(g.appid, "show")}>Not clutter</button>
          </article>
        ))}
      </div>
      <section className="panel" style={{ display: "flex", flexDirection: "column", gap: 10 }}>
        <h3>Hide a game that was missed</h3>
        <div className="toolbar">
          <input className="input" list="hide-candidates" placeholder="Game name" aria-label="Game name to hide" value={name} onChange={(e) => setName(e.target.value)} style={{ minWidth: 320 }} />
          <datalist id="hide-candidates">{candidates.map((g) => <option key={g.appid} value={g.name} />)}</datalist>
          <button className="btn" disabled={!match} onClick={() => match && override(match.appid, "hide").then(() => setName(""))}>Hide</button>
        </div>
      </section>
    </div>
  );
}
