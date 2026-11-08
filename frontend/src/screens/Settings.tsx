import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  DataService, errorText, Status, EVENTS, LibraryService, MetaService, SettingsService, SetupService,
  list, type Settings as SettingsModel, type Summary, type SyncState,
} from "../api";
import { useEvent, useGames, useQuery, useRefresh } from "../data";
import type { Route } from "../routes";
import { useReportError, useToast } from "../toast";

/** G1: every preference, plus key and sync controls, backup, export and import. */
export function Settings({ go }: { go: (r: Route) => void }) {
  const loaded = useQuery(() => SettingsService.Get());
  const conn = useQuery(() => SetupService.Status());
  const games = useGames();
  const [draft, setDraft] = useState<SettingsModel | null>(null);
  const [sync, setSync] = useState<SyncState | null>(null);
  const refresh = useRefresh();
  const report = useReportError();
  const toast = useToast();
  useEvent<SyncState>(EVENTS.sync, setSync);
  useEffect(() => { if (loaded.data) setDraft(loaded.data); }, [loaded.data]);

  const allTags = useMemo(() => [...new Set((games.data ?? []).flatMap((g) => [...list(g.tags), ...list(g.genres)]))].sort(), [games.data]);
  if (!draft) return <div className="page"><h1>Settings</h1></div>;

  const save = (next: SettingsModel) => {
    setDraft(next);
    SettingsService.Save(next).then(refresh, (e) => { report(e); refresh(); });
  };
  const set = <K extends keyof SettingsModel>(k: K, v: SettingsModel[K]) => save({ ...draft, [k]: v });
  const hiddenCount = (games.data ?? []).filter((g) => g.hidden).length;
  const deadCount = (games.data ?? []).filter((g) => g.status === Status.Dead && !g.hidden).length;
  const lastSync = draft.last_sync_at ? new Date(draft.last_sync_at).toLocaleString() : "never";

  return (
    <div className="page" style={{ maxWidth: 1000 }}>
      <h1>Settings</h1>

      <Section title="Steam" note="Your profile, API key and the data the app keeps from Steam.">
        <ProfileRow current={conn.data?.profileUrl ?? ""} onSaved={refresh} />
        <KeyRow onSaved={refresh} />
        <Row label="Library sync" value={sync?.running ? "Syncing…" : sync?.error ? <span className="warn-text">{sync.error}</span> : `Last synced ${lastSync}`}>
          <button className="btn primary" disabled={!!sync?.running} onClick={() => LibraryService.SyncNow().then(refresh, report)}>Sync now</button>
        </Row>
        <Row label="Store data" value="Kept permanently. Re-fetch replaces it with fresh data from Steam.">
          <button className="btn" onClick={() => MetaService.RefetchAll().then(() => toast({ message: "Re-fetching store data for every game" }), report)}>Re-fetch all</button>
        </Row>
      </Section>

      <Section title="Recommendations and wheel" note="Which games are eligible, and how the recent taste profile is measured.">
        <Row label="Shelved eligible under" value={<NumberField value={draft.shelved_hours} min={0} suffix="hours" onCommit={(v) => set("shelved_hours", v)} />} />
        <Row label="Endless eligible under" value={<NumberField value={draft.endless_hours} min={0} suffix="hours" onCommit={(v) => set("endless_hours", v)} />} />
        <Row label="Minimum reviews" value={<NumberField value={draft.min_reviews} min={0} suffix="reviews" onCommit={(v) => set("min_reviews", v)} />} />
        <Row label="Recent window" value={<NumberField value={draft.recent_months} min={1} suffix="months" onCommit={(v) => set("recent_months", v)} />} />
      </Section>

      <Section title="Taste tags" note="Tune the taste profile: ignore generic tags, always include favourites, and rule out what you never want.">
        <Row label="Stoplist" value={<TagList values={list(draft.stoplist)} suggestions={allTags} onChange={(v) => set("stoplist", v)} />} />
        <Row label="Pinned tags" value={<TagList values={list(draft.pinned)} suggestions={allTags} onChange={(v) => set("pinned", v)} />} />
        <Row label="Avoided tags" value={<TagList values={list(draft.avoided)} suggestions={allTags} onChange={(v) => set("avoided", v)} />} />
      </Section>

      <Section title="Status rules" note="When home flags a 'playing' game as abandoned, and how classify splits new games.">
        <Row label="Abandoned after" value={<NumberField value={draft.abandoned_days} min={1} suffix="days" onCommit={(v) => set("abandoned_days", v)} />} />
        <Row label="Classify active window" value={<NumberField value={draft.classify_days} min={1} suffix="days" onCommit={(v) => set("classify_days", v)} />} />
      </Section>

      <Section title="Hidden and Dead games" note="Games kept out of everyday views.">
        <Row label="Hidden-name patterns" value={<TagList values={list(draft.hidden_patterns)} onChange={(v) => set("hidden_patterns", v)} />} />
        <Row label="Hidden games" value={`${hiddenCount} hidden`}>
          <button className="btn" onClick={() => go("hidden")}>Open</button>
        </Row>
        <Row label="Dead games" value={`${deadCount} marked Dead (servers shut down or unplayable)`}>
          <button className="btn" onClick={() => go("dead")}>Open</button>
        </Row>
      </Section>

      <Section title="Your data" note="Everything stays on this PC. Back up or move it with these.">
        <Row label="Back up" value="Save a copy of the database (never includes the API key).">
          <button className="btn" onClick={() => DataService.Backup().then((p) => toast({ message: `Saved ${p}` }), quiet(report))}>Back up</button>
        </Row>
        <Row label="Export" value="Statuses, ratings, hidden flags, tag settings and history as JSON.">
          <button className="btn" onClick={() => DataService.Export().then((p) => toast({ message: `Exported to ${p}` }), quiet(report))}>Export</button>
        </Row>
        <ImportRow onDone={refresh} />
      </Section>

      <Section title="About">
        <p className="subtle" style={{ margin: "14px 0" }}>
          So Many Games stores your owned games, playtime, store details, cover art and statuses in %APPDATA%\SoManyGames on this PC.
          It talks only to Steam. Not affiliated with Valve; Steam data is provided as is. Your steam API key never leaves your PC credential vault.
        </p>
      </Section>
    </div>
  );
}

/** Ignores a closed file dialog; reports anything else. */
const quiet = (report: (e: unknown) => void) => (e: unknown) => { if (errorText(e) !== "cancelled") report(e); };

function Section({ title, note, children }: { title: string; note?: string; children: ReactNode }) {
  return (
    <section className="settings-section" aria-label={title}>
      <header>
        <h2>{title}</h2>
        {note && <p className="subtle">{note}</p>}
      </header>
      {children}
    </section>
  );
}

function Row({ label, value, children }: { label: string; value?: ReactNode; children?: ReactNode }) {
  return (
    <div className="setting-row">
      <span>{label}</span>
      <div className="value">{value}</div>
      <div>{children}</div>
    </div>
  );
}

function NumberField({ value, min, suffix, onCommit }: { value: number; min: number; suffix: string; onCommit: (v: number) => void }) {
  const [text, setText] = useState(String(value));
  useEffect(() => setText(String(value)), [value]);
  const commit = () => {
    const n = Math.floor(Number(text));
    if (Number.isFinite(n) && n >= min && n !== value) onCommit(n);
    else setText(String(value));
  };
  return (
    <span className="toolbar">
      <input className="input number-input" type="number" min={min} value={text} aria-label={suffix}
        onChange={(e) => setText(e.target.value)} onBlur={commit} onKeyDown={(e) => e.key === "Enter" && commit()} />
      <span>{suffix}</span>
    </span>
  );
}

function TagList({ values, suggestions = [], onChange }: { values: string[]; suggestions?: string[]; onChange: (v: string[]) => void }) {
  const [text, setText] = useState("");
  const id = useMemo(() => `tags-${Math.random().toString(36).slice(2)}`, []);
  const add = () => {
    const v = text.trim();
    if (v && !values.some((x) => x.toLowerCase() === v.toLowerCase())) onChange([...values, v]);
    setText("");
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <div className="chips">
        {values.map((v) => (
          <span key={v} className="chip">{v}<button aria-label={`Remove ${v}`} onClick={() => onChange(values.filter((x) => x !== v))}>✕</button></span>
        ))}
        {values.length === 0 && <span className="faint">None</span>}
      </div>
      <div className="toolbar">
        <input className="input" list={id} value={text} placeholder="Add…" aria-label="Add a tag" onChange={(e) => setText(e.target.value)} onKeyDown={(e) => e.key === "Enter" && add()} />
        <datalist id={id}>{suggestions.map((s) => <option key={s} value={s} />)}</datalist>
        <button className="btn small" onClick={add} disabled={!text.trim()}>Add</button>
      </div>
    </div>
  );
}

function ProfileRow({ current, onSaved }: { current: string; onSaved: () => void }) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(current);
  const [error, setError] = useState<string | null>(null);
  const save = () => SetupService.Connect("", value).then(() => { setEditing(false); setError(null); onSaved(); }, (e) => setError(errorText(e)));
  return (
    <Row label="Profile" value={editing ? (
      <span className="toolbar">
        <input className="input" value={value} aria-label="Profile URL or SteamID" onChange={(e) => setValue(e.target.value)} style={{ minWidth: 320 }} />
        {error && <span className="warn-text">{error}</span>}
      </span>
    ) : current || "Not set"}>
      {editing ? <button className="btn primary" onClick={save}>Save</button> : <button className="btn" onClick={() => { setValue(current); setEditing(true); }}>Change</button>}
    </Row>
  );
}

function KeyRow({ onSaved }: { onSaved: () => void }) {
  const [editing, setEditing] = useState(false);
  const [key, setKey] = useState("");
  const [error, setError] = useState<string | null>(null);
  const status = useQuery(() => SetupService.Status());
  const save = async () => {
    try {
      await SetupService.Connect(key, status.data?.profileUrl ?? "");
      setKey(""); setEditing(false); setError(null); onSaved();
    } catch (e) {
      setError(errorText(e));
    }
  };
  return (
    <Row label="API key" value={editing ? (
      <span className="toolbar">
        <input className="input" type="password" autoComplete="off" value={key} aria-label="New Steam Web API key" onChange={(e) => setKey(e.target.value)} />
        {error && <span className="warn-text">{error}</span>}
      </span>
    ) : "Saved in Windows Credential Manager"}>
      {editing ? <button className="btn primary" disabled={!key} onClick={save}>Save</button> : <button className="btn" onClick={() => setEditing(true)}>Change</button>}
    </Row>
  );
}

function ImportRow({ onDone }: { onDone: () => void }) {
  const [summary, setSummary] = useState<Summary | null>(null);
  const report = useReportError();
  const toast = useToast();
  return (
    <>
      <Row label="Import" value="Replace statuses, ratings, hidden flags, tag settings and history from an export.">
        <button className="btn" onClick={() => DataService.PreviewImport().then(setSummary, quiet(report))}>Import…</button>
      </Row>
      {summary && (
        <div className="panel warn" role="alertdialog" aria-label="Confirm import" style={{ margin: "12px 0" }}>
          <h3>Replace your data with this file?</h3>
          <ul className="subtle">
            <li>{summary.gamesInFile} games in the file ({summary.newGames} not in your library yet)</li>
            <li>{summary.statusChanges} status changes, {summary.ratingChanges} rating changes, {summary.hiddenChanges} hidden-flag changes</li>
            <li>History: {summary.eventsInFile} events replace your current {summary.eventsNow}</li>
            <li>Tag settings {summary.tagSettingsDiff ? "will change" : "are the same"}</li>
          </ul>
          <div className="toolbar">
            <button className="btn primary" onClick={() => DataService.ConfirmImport().then(() => { setSummary(null); toast({ message: "Import complete" }); onDone(); }, report)}>Replace my data</button>
            <button className="btn" onClick={() => setSummary(null)}>Cancel</button>
          </div>
        </div>
      )}
    </>
  );
}
