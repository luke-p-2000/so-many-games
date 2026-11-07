import { useMemo, useState } from "react";
import { hours, LibraryService, list, MetaService, lastPlayedText, type Game } from "../api";
import { FetchBanner } from "../components/Banners";
import { Segmented } from "../components/bits";
import { GameRow, GridTile, PickBackUpButton, tagLine, UnavailableToggle } from "../components/GameCards";
import { useGames, useRefresh } from "../data";
import type { Route } from "../routes";
import { STATUSES, type StatusValue } from "../statuses";
import { useReportError, useSetStatus, useToast } from "../toast";

type Sort = "name" | "hours" | "lastPlayed" | "review" | "release";
const HOURS: Record<string, [number, number]> = {
  "": [0, Infinity], under2: [0, 2], "2to10": [2, 10], "10to50": [10, 50], over50: [50, Infinity],
};

const sorters: Record<Sort, (a: Game, b: Game) => number> = {
  name: (a, b) => a.name.localeCompare(b.name),
  hours: (a, b) => b.minutes - a.minutes,
  lastPlayed: (a, b) => (b.lastPlayed ?? "").localeCompare(a.lastPlayed ?? ""),
  review: (a, b) => (b.reviewPct ?? -1) - (a.reviewPct ?? -1) || (b.reviewCount ?? 0) - (a.reviewCount ?? 0),
  release: (a, b) => b.releaseDate.localeCompare(a.releaseDate),
};

export function Library({ go }: { go: (r: Route) => void }) {
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
      .sort(sorters[sort]);
  }, [all, search, status, tag, genre, hoursRange, minReview, sort, showHidden]);

  const toggle = (id: number, on: boolean) => setSelected((s) => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });
  const ids = [...selected];

  return (
    <div className="page">
      <div className="page-head">
        <h1>Library</h1>
        <div className="toolbar one-line">
          <select className="select" aria-label="Sort" value={sort} onChange={(e) => setSort(e.target.value as Sort)}>
            <option value="name">Sort: Name</option>
            <option value="hours">Sort: Hours</option>
            <option value="lastPlayed">Sort: Last played</option>
            <option value="review">Sort: Review score</option>
            <option value="release">Sort: Release date</option>
          </select>
          <label className="check"><input type="checkbox" checked={showHidden} onChange={(e) => setShowHidden(e.target.checked)} /> Show hidden</label>
          <PickBackUpButton go={go} />
          <Segmented label="Layout" value={view} onChange={setView} options={[{ value: "rows", label: "Rows" }, { value: "grid", label: "Grid" }]} />
        </div>
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
      <p className="faint">{visible.length} of {all.filter((g) => !g.hidden).length} games</p>

      {view === "rows" ? (
        <div className="row-list">
          {visible.map((g) => (
            <GameRow
              key={g.appid}
              game={g}
              selected={selected.has(g.appid)}
              onSelect={(on) => toggle(g.appid, on)}
              sub={<>{tagLine(g)}{g.hidden && <> · hidden ({g.hiddenReason})</>}{!g.classified && <> · unclassified</>}</>}
              numbers={<><span><b>{hours(g.minutes)}</b></span><span style={{ minWidth: 150 }}>{lastPlayedText(g.lastPlayed).replace("last played ", "")}</span></>}
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
