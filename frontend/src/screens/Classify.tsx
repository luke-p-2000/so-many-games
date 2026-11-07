import { useEffect, useMemo, useState } from "react";
import { daysSince, hours, LibraryService, SettingsService, Status, type Game } from "../api";
import { GameRow } from "../components/GameCards";
import { useGames, useQuery, useRefresh } from "../data";
import type { Route } from "../routes";
import { type StatusValue } from "../statuses";
import { useReportError, useToast } from "../toast";

/** L7: first-import triage. Changes are staged until Save all. */
export function Classify({ go }: { go: (r: Route) => void }) {
  const games = useGames();
  const settings = useQuery(() => SettingsService.Get());
  const [windowDays, setWindowDays] = useState<number | null>(null);
  const [staged, setStaged] = useState<Record<number, StatusValue>>({});
  const [saving, setSaving] = useState(false);
  const refresh = useRefresh();
  const report = useReportError();
  const toast = useToast();

  const days = windowDays ?? settings.data?.classify_days ?? 30;
  const pending = useMemo(() => (games.data ?? []).filter((g) => !g.classified && !g.hidden), [games.data]);
  const recent = pending.filter((g) => (daysSince(g.lastPlayed) ?? Infinity) <= days).sort((a, b) => b.minutes - a.minutes);
  const rest = pending.filter((g) => !recent.includes(g)).sort((a, b) => b.minutes - a.minutes);

  // Changing the window re-stages only games the owner hasn't touched.
  const [touched, setTouched] = useState<Set<number>>(new Set());
  useEffect(() => {
    setStaged((prev) => {
      const next: Record<number, StatusValue> = {};
      for (const g of pending) {
        next[g.appid] = touched.has(g.appid) && prev[g.appid] ? prev[g.appid] : recent.includes(g) ? Status.Playing : Status.Shelved;
      }
      return next;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pending, days]);

  const stage = (g: Game, s: StatusValue) => {
    setStaged((p) => ({ ...p, [g.appid]: s }));
    setTouched((t) => new Set(t).add(g.appid));
  };

  const save = async () => {
    setSaving(true);
    try {
      await LibraryService.SaveClassification(pending.map((g) => ({ appid: g.appid, status: staged[g.appid] })));
      toast({ message: `Classified ${pending.length} games` });
      refresh();
      go("home");
    } catch (err) {
      report(err);
    } finally {
      setSaving(false);
    }
  };

  if (!games.loading && pending.length === 0) {
    return <div className="page"><h1>Classify</h1><div className="panel empty">Everything is classified.</div></div>;
  }

  const section = (list: Game[]) => (
    <div className="row-list">
      {list.map((g) => (
        <GameRow key={g.appid} game={g} status={staged[g.appid] ?? g.status} onStatus={(s) => stage(g, s)}
          sub={g.lastPlayed ? `last played ${daysSince(g.lastPlayed)} days ago` : "never played"}
          numbers={<span><b>{hours(g.minutes)}</b></span>} />
      ))}
    </div>
  );

  return (
    <div className="page">
      <div className="page-head">
        <h1>First import — classify</h1>
        <button className="btn primary" disabled={saving} onClick={save}>Save all ({pending.length})</button>
      </div>
      <p className="subtle">Games with 0 minutes were set to Unplayed. Changes here are staged until Save all; until then these games stay out of the wheel, recommendations and stats.</p>
      <h2>
        Played in the last{" "}
        <input className="input number-input" type="number" min={1} max={3650} aria-label="Classify window in days" value={days}
          onChange={(e) => setWindowDays(Math.max(1, Number(e.target.value) || 1))} style={{ width: 80, padding: "4px 8px" }} />{" "}
        days · default Playing
      </h2>
      {recent.length ? section(recent) : <p className="faint">None.</p>}
      <h2>Everything else · by hours, default Shelved</h2>
      {section(rest)}
    </div>
  );
}
