import { useMemo, useState } from "react";
import { hours, LibraryService, list, MetaService, lastPlayedText, type Game } from "../api";
import { FetchBanner } from "../components/Banners";
import { Segmented } from "../components/bits";
import { GameRow, GridTile, tagLine, UnavailableToggle } from "../components/GameCards";
import { useGames, useRefresh } from "../data";
import { STATUSES, StatusIcon, statusLabel, statusStyle, type StatusValue } from "../statuses";
import { useReportError, useSetStatus, useToast } from "../toast";

type Sort = "name" | "hours" | "lastPlayed" | "review" | "release";
const HOURS: Record<string, [number, number]> = {
  "": [0, Infinity], under2: [0, 2], "2to10": [2, 10], "10to50": [10, 50], over50: [50, Infinity],
};

type Key = string | number | null;
/** Each sort's keys, most significant first; null means no data. */
const sortKeys: Record<Sort, (g: Game) => Key[]> = {
  name: (g) => [g.name.toLowerCase()],
  hours: (g) => [g.minutes],
  lastPlayed: (g) => [g.lastPlayed || null],
  review: (g) => [g.reviewPct ?? null, g.reviewCount ?? 0],
  release: (g) => [g.releaseDate || null],
};
/** The direction a sort starts in: names A–Z, everything else highest or newest first. */
const DEFAULT_DESC: Record<Sort, boolean> = { name: false, hours: true, lastPlayed: true, review: true, release: true };

/** Compares by the sort's keys in the chosen direction; games with no data stay last either way. */
function compare(sort: Sort, desc: boolean) {
  return (a: Game, b: Game) => {
    const ka = sortKeys[sort](a), kb = sortKeys[sort](b);
    for (let i = 0; i < ka.length; i++) {
      const x = ka[i], y = kb[i];
      if (x === y) continue;
      if (x === null) return 1;
      if (y === null) return -1;
      const c = typeof x === "string" ? x.localeCompare(y as string) : x - (y as number);
      if (c) return desc ? -c : c;
    }
    return a.name.localeCompare(b.name);
  };
}

export function Library() {
  const games = useGames();
  const setStatus = useSetStatus();
  const refresh = useRefresh();
  const report = useReportError();
  const toast = useToast();

  const [search, setSearch] = useState("");
  const [status, setStatusFilter] = useState("");
  const [tag, setTag] = useState("");
  const [genre, setGenre] = useState("");
  const [hoursRange, setHoursRange] = useState("");
  const [minReview, setMinReview] = useState(0);
  const [sort, setSort] = useState<Sort>("name");
  const [desc, setDesc] = useState(false);
  const [showHidden, setShowHidden] = useState(false);
  const [view, setView] = useState<"rows" | "grid">("rows");
  const [selected, setSelected] = useState<Set<number>>(new Set());

  const all = games.data ?? [];
  const { tags, genres } = useMemo(() => {
    const t = new Set<string>(), g = new Set<string>();
    all.forEach((x) => { list(x.tags).forEach((v) => t.add(v)); list(x.genres).forEach((v) => g.add(v)); });
    return { tags: [...t].sort(), genres: [...g].sort() };
  }, [all]);

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    const [lo, hi] = HOURS[hoursRange];
    return all
      .filter((g) => (showHidden || !g.hidden)
        && (!q || g.name.toLowerCase().includes(q))
        && (!status || g.status === status)
        && (!tag || list(g.tags).includes(tag))
        && (!genre || list(g.genres).includes(genre))
        && g.minutes / 60 >= lo && g.minutes / 60 < hi
        && (!minReview || (g.reviewPct ?? -1) >= minReview))
      .sort(compare(sort, desc));
  }, [all, search, status, tag, genre, hoursRange, minReview, sort, desc, showHidden]);

  // The hours bar is on a square-root scale against the most played game, so short games still show.
  const maxMinutes = Math.max(1, ...all.map((g) => g.minutes));
  const toggle = (id: number, on: boolean) => setSelected((s) => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });
  const ids = [...selected];

  return (
    <div className="page">
      <div className="page-head">
        <h1>Library</h1>
        <Segmented label="Layout" value={view} onChange={setView} options={[{ value: "rows", label: "Rows" }, { value: "grid", label: "Grid" }]} />
      </div>

      <div className="toolbar one-line" role="search">
        <input className="input search" type="search" placeholder="Search library" aria-label="Search library" value={search} onChange={(e) => setSearch(e.target.value)} />
        <select className="select" aria-label="Status" value={status} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="">All statuses</option>
          {STATUSES.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
        </select>
        <select className="select" aria-label="Tag" value={tag} onChange={(e) => setTag(e.target.value)}>
          <option value="">All tags</option>
          {tags.map((t) => <option key={t}>{t}</option>)}
        </select>
        <select className="select" aria-label="Genre" value={genre} onChange={(e) => setGenre(e.target.value)}>
          <option value="">All genres</option>
          {genres.map((t) => <option key={t}>{t}</option>)}
        </select>
        <select className="select" aria-label="Hours played" value={hoursRange} onChange={(e) => setHoursRange(e.target.value)}>
          <option value="">Any hours</option>
          <option value="under2">Under 2 h</option>
          <option value="2to10">2–10 h</option>
          <option value="10to50">10–50 h</option>
          <option value="over50">50 h or more</option>
        </select>
        <select className="select" aria-label="Review score" value={minReview} onChange={(e) => setMinReview(Number(e.target.value))}>
          <option value={0}>Any reviews</option>
          <option value={70}>70%+</option>
          <option value={80}>80%+</option>
          <option value={90}>90%+</option>
        </select>
      </div>

      <FetchBanner />
      <div className="list-head">
        <span className="faint">{visible.length} of {all.filter((g) => !g.hidden).length} games</span>
        <div className="toolbar">
          <label className="check faint"><input type="checkbox" checked={showHidden} onChange={(e) => setShowHidden(e.target.checked)} /> Show hidden</label>
          <label className="sort-by faint">
            Sort by
            <select className="select" value={sort} onChange={(e) => { const s = e.target.value as Sort; setSort(s); setDesc(DEFAULT_DESC[s]); }}>
              <option value="name">Name</option>
              <option value="hours">Hours</option>
              <option value="lastPlayed">Last played</option>
              <option value="review">Review score</option>
              <option value="release">Release date</option>
            </select>
          </label>
          <button type="button" className="btn small ghost sort-dir" onClick={() => setDesc((d) => !d)}
            aria-label={desc ? "Descending, switch to ascending" : "Ascending, switch to descending"} title={desc ? "Descending" : "Ascending"}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden
              style={{ transform: desc ? "none" : "rotate(180deg)" }}>
              <path d="M12 5v14M6 13l6 6 6-6" />
            </svg>
          </button>
        </div>
      </div>

      {view === "rows" ? (
        <div className="row-list">
          {visible.map((g) => (
            <GameRow
              key={g.appid}
              game={g}
              selected={selected.has(g.appid)}
              onSelect={(on) => toggle(g.appid, on)}
              sub={<><span className="row-status" style={statusStyle(g.status)}><StatusIcon status={g.status} size={12} />{statusLabel(g.status)}</span> · {tagLine(g)}{g.hidden && <> · hidden ({g.hiddenReason})</>}{!g.classified && <> · unclassified</>}</>}
              numbers={<>
                <span className="hours-cell"><b>{hours(g.minutes)}</b><span className="hours-bar" aria-hidden><span style={{ width: `${Math.min(100, Math.sqrt(g.minutes / maxMinutes) * 100)}%` }} /></span></span>
                <span style={{ minWidth: 150 }}>{lastPlayedText(g.lastPlayed).replace("last played ", "")}</span>
              </>}
              extra={<UnavailableToggle game={g} />}
            />
          ))}
        </div>
      ) : (
        <div className="card-grid dense">{visible.map((g) => <GridTile key={g.appid} game={g} />)}</div>
      )}

      {selected.size > 0 && (
        <div className="bulk-bar" role="region" aria-label="Bulk actions">
          <span>{selected.size} selected</span>
          <select className="select" aria-label="Set status for selected games" value="" onChange={(e) => {
            if (!e.target.value) return;
            setStatus(ids, e.target.value as StatusValue, `${ids.length} games`).then(() => setSelected(new Set()));
          }}>
            <option value="">Set status…</option>
            {STATUSES.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
          </select>
          <button className="btn" onClick={async () => {
            try {
              for (const id of ids) await LibraryService.SetHideOverride(id, "hide");
              toast({ message: `Hid ${ids.length} games` });
              setSelected(new Set());
              refresh();
            } catch (err) { report(err); }
          }}>Hide</button>
          <button className="btn" onClick={() => MetaService.Refetch(ids).then(() => { setSelected(new Set()); refresh(); }, report)}>Re-fetch store data</button>
          <button className="btn ghost" onClick={() => setSelected(new Set())}>Clear</button>
        </div>
      )}
    </div>
  );
}
