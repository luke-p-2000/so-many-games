import { useEffect, useMemo, useRef, useState } from "react";
import { hours, list, type Filters, Status, WheelService, type Game, type WheelDraw } from "../api";
import { Cover } from "../components/bits";
import { ShareButton } from "../components/ShareCard";
import { reviewsShort, scoreTone } from "../components/GameCards";
import { useGames, useQuery, useRefresh } from "../data";
import { drawWheelPickCard, wheelPickFileName } from "../shareCards";
import { StatusIcon, statusLabel, statusStyle } from "../statuses";
import { useReportError, useSetStatus } from "../toast";

/** Wedge hues from the status palette, drawn faint; the winning wedge lights up. */
const COLOURS = ["#e8b04b", "#5b9cf5", "#a685e8", "#3fc1c0", "#e66a5c", "#8fcf6e"];
/** Allowed sizes in slider order; 0 means All (W2). */
const SIZES = [12, 15, 18, 24, 0];
const spinMs = () => (window.matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : 4300);

/**
 * W1–W3: the wheel is drawn from the filtered pool whenever the filters or size change, so it
 * always shows its games; Spin picks one of them. For All it is a name reel instead of wedges.
 */
export function Wheel() {
  const games = useGames();
  const savedSize = useQuery(() => WheelService.Size());
  const [size, setSize] = useState<number | null>(null);
  useEffect(() => { if (savedSize.data !== undefined && size === null) setSize(savedSize.data); }, [savedSize.data, size]);
  const setStatus = useSetStatus();
  const refresh = useRefresh();
  const report = useReportError();

  const [filters, setFilters] = useState<Filters>(NO_FILTERS);
  const [bias, setBias] = useState(false);
  const [redraws, setRedraws] = useState(0);
  const draw = useQuery<WheelDraw | null>(
    () => (size === null ? Promise.resolve(null) : WheelService.Draw(filters, bias, size)),
    [JSON.stringify(filters), bias, size, redraws],
  );
  const entries = list(draw.data?.entries);

  const [winner, setWinner] = useState(-1);
  const [spins, setSpins] = useState(0);
  const [spinning, setSpinning] = useState(false);
  const [result, setResult] = useState<Game | null>(null);
  // A new draw clears the last winner.
  useEffect(() => { setWinner(-1); setResult(null); }, [draw.data]);

  const { tags, genres } = useMemo(() => {
    const t = new Set<string>(), g = new Set<string>();
    (games.data ?? []).forEach((x) => { list(x.tags).forEach((v) => t.add(v)); list(x.genres).forEach((v) => g.add(v)); });
    return { tags: [...t].sort(), genres: [...g].sort() };
  }, [games.data]);

  const update = (patch: Partial<Filters>) => setFilters((f) => ({ ...f, ...patch }));
  const changeSize = (n: number) => {
    setSize(n);
    WheelService.SetSize(n).catch(report);
  };

  const spin = async () => {
    setResult(null);
    try {
      const w = await WheelService.Pick(entries.map((g) => g.appid), bias);
      if (w < 0) { refresh(); return; } // a game left the pool since the draw
      setWinner(w);
      setSpins((n) => n + 1);
      setSpinning(true);
      window.setTimeout(() => { setSpinning(false); setResult(entries[w]); }, spinMs());
    } catch (err) {
      report(err);
    }
  };
  const act = async (s: Status.Playing | Status.NotForMe) => {
    if (!result) return;
    await setStatus([result.appid], s, result.name);
    setResult(null);
  };

  const empty = draw.data && entries.length === 0;
  return (
    <div className="page">
      <h1>Wheel</h1>
      <div className="wheel-layout">
        <div className="wheel-main">
          <div className="wheel-stage">
            {size === 0 ? <Reel entries={entries} winner={winner} spins={spins} />
              : <WheelSvg entries={entries} winner={winner} spins={spins} landed={!!result} onSpin={spin} spinning={spinning} />}
            {result ? <Winner game={result} onPlay={() => act(Status.Playing)} onAgain={spin} onNotForMe={() => act(Status.NotForMe)} />
              : <div className="winner-placeholder">{spinning ? "Spinning…" : empty ? "No eligible games match these filters." : "Spin the wheel to pick a game."}</div>}
          </div>
          <div className="toolbar">
            {size === 0 && <button className="btn primary" onClick={spin} disabled={spinning || !entries.length}>Spin</button>}
            {size !== 0 && draw.data && draw.data.poolSize > entries.length && (
              <button className="btn" onClick={() => setRedraws((n) => n + 1)} disabled={spinning}>New games</button>
            )}
            {draw.data && <span className="faint">{draw.data.poolSize} eligible {draw.data.poolSize === 1 ? "game" : "games"}</span>}
          </div>
        </div>

        <aside className="panel filters" aria-label="Wheel filters">
          <h2>Filters</h2>
          <fieldset disabled={spinning}>
            <TagPicker tags={tags} value={list(filters.tags)} onChange={(v) => update({ tags: v })} />
            <FilterSelect label="Genre" value={filters.genre} onChange={(v) => update({ genre: v })} options={genres.map((t) => [t, t])} />
            <FilterSelect label="Review score" value={String(filters.minReviewPct || "")} onChange={(v) => update({ minReviewPct: Number(v) || 0 })}
              options={[["70", "70%+"], ["80", "80%+"], ["90", "90%+"]]} />
            <FilterSelect label="Type" value={filters.type} onChange={(v) => update({ type: v })} options={TYPES} />
            <StatusFilter value={list(filters.status)} onChange={(v) => update({ status: v })} />
            <label className="check"><input type="checkbox" checked={bias} onChange={(e) => setBias(e.target.checked)} /> Bias toward my taste</label>
            <div className="slider-field">
              <div className="filter-row"><span>Wheel size</span><b>{size === null ? "…" : size || "All"}</b></div>
              <input type="range" min={0} max={SIZES.length - 1} step={1} aria-label="Wheel size" aria-valuetext={size ? String(size) : "All"}
                value={Math.max(0, SIZES.indexOf(size ?? 12))} onChange={(e) => changeSize(SIZES[Number(e.target.value)])} />
              <div className="slider-ticks" aria-hidden>{SIZES.map((n) => <span key={n}>{n || "All"}</span>)}</div>
            </div>
          </fieldset>
        </aside>
      </div>
    </div>
  );
}

/** Every filter starts at All (W1). */
const NO_FILTERS: Filters = { tags: [], genre: "", minReviewPct: 0, type: "", status: [] };

/** The statuses that can be in the eligible pool (R3). */
const POOL_STATUSES = [Status.Unplayed, Status.ToPlay, Status.Shelved, Status.Endless];

/** Any of the ticked statuses; none ticked means all but Endless, which joins only when ticked. */
function StatusFilter({ value, onChange }: { value: string[]; onChange: (v: string[]) => void }) {
  const toggle = (s: string) => onChange(value.includes(s) ? value.filter((x) => x !== s) : [...value, s]);
  return (
    <div role="group" aria-label="Status">
      <div className="filter-row">
        <span>Status</span>
        <button type="button" className="link" disabled={!value.length} onClick={() => onChange([])} title="Endless games join the wheel only when Endless is ticked">{value.length ? "Clear" : "All but Endless"}</button>
      </div>
      <div className="status-filter">
        {POOL_STATUSES.map((s) => (
          <label key={s} className="check status-check" style={statusStyle(s)}><input type="checkbox" checked={value.includes(s)} onChange={() => toggle(s)} /> <StatusIcon status={s} size={14} /> {statusLabel(s)}</label>
        ))}
      </div>
    </div>
  );
}

/** Steam player categories (internal/wheel Types). */
const TYPES: [string, string][] = [["single-player", "Single-player"], ["co-op", "Co-op"], ["pvp", "PvP"]];

function FilterSelect({ label, value, onChange, options }: { label: string; value: string; onChange: (v: string) => void; options: [string, string][] }) {
  return (
    <label className="filter-row">
      {label}
      <select className="select" value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">All</option>
        {options.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
      </select>
    </label>
  );
}

/**
 * Several tags; a game must have all of them. A button opens a searchable checklist, and the
 * chosen tags stay visible as chips that can be removed one by one.
 */
function TagPicker({ tags, value, onChange }: { tags: string[]; value: string[]; onChange: (v: string[]) => void }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const close = (e: MouseEvent) => { if (!box.current?.contains(e.target as Node)) setOpen(false); };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, [open]);

  const toggle = (t: string) => onChange(value.includes(t) ? value.filter((x) => x !== t) : [...value, t]);
  const q = query.trim().toLowerCase();
  // Ticked tags first, so the current choice is visible without scrolling.
  const shown = [...value, ...tags.filter((t) => !value.includes(t))].filter((t) => t.toLowerCase().includes(q));
  const label = value.length === 0 ? "All" : value.length === 1 ? value[0] : `${value.length} tags`;

  return (
    <div className="tag-picker" ref={box} onKeyDown={(e) => e.key === "Escape" && setOpen(false)}>
      <div className="filter-row">
        <span>Tags</span>
        <button type="button" className="select tag-picker-button" aria-haspopup="true" aria-expanded={open} onClick={() => setOpen((o) => !o)}>
          {label}
        </button>
      </div>
      {open && (
        <div className="tag-picker-panel" role="group" aria-label="Choose tags">
          <input className="input" type="search" placeholder="Search tags" aria-label="Search tags" autoFocus value={query} onChange={(e) => setQuery(e.target.value)} />
          <div className="tag-picker-list">
            {shown.map((t) => (
              <label key={t} className="check">
                <input type="checkbox" checked={value.includes(t)} onChange={() => toggle(t)} /> {t}
              </label>
            ))}
            {shown.length === 0 && <span className="faint">No matching tags</span>}
          </div>
          <div className="tag-picker-foot">
            <span className="faint">Games must have every tag ticked.</span>
            <button type="button" className="link" disabled={!value.length} onClick={() => onChange([])}>Clear</button>
          </div>
        </div>
      )}
      {value.length > 0 && (
        <div className="chips">
          {value.map((t) => (
            <span key={t} className="chip accent">
              {t}
              <button type="button" aria-label={`Remove ${t}`} onClick={() => toggle(t)}>×</button>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

/** Wedges; each spin turns forward several times and stops with the winner under the top pointer. */
function WheelSvg({ entries, winner, spins, landed, onSpin, spinning }: {
  entries: Game[]; winner: number; spins: number; landed: boolean; onSpin: () => void; spinning: boolean;
}) {
  const n = Math.max(entries.length, 1);
  const slice = 360 / n;
  const [angle, setAngle] = useState(0);
  useEffect(() => {
    if (winner < 0) return;
    setAngle((a) => a - (a % 360) + 5 * 360 - (winner * slice + slice / 2));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [spins]);
  const r = 100;
  const point = (deg: number) => {
    const a = ((deg - 90) * Math.PI) / 180;
    return [r + r * Math.cos(a), r + r * Math.sin(a)];
  };
  return (
    <div className="wheel-wrap">
      <svg viewBox="0 0 200 200" role="img" aria-label={entries.length ? `Wheel with ${entries.length} games` : "Empty wheel"}>
        <g className="wheel-spin" style={{ transform: `rotate(${angle}deg)` }}>
          {entries.length === 0 && <circle cx="100" cy="100" r="100" fill="var(--panel)" stroke="var(--border)" />}
          {entries.map((g, i) => {
            const [x1, y1] = point(i * slice), [x2, y2] = point((i + 1) * slice);
            const mid = i * slice + slice / 2;
            const won = landed && i === winner;
            const path = n === 1 ? "M100,0 A100,100 0 1,1 99.99,0 Z" : `M100,100 L${x1},${y1} A100,100 0 ${slice > 180 ? 1 : 0},1 ${x2},${y2} Z`;
            return (
              <g key={g.appid}>
                <path d={path} fill={COLOURS[i % COLOURS.length]} fillOpacity={won ? 0.95 : 0.22} stroke="var(--bg)" strokeWidth="1.2" />
                <text x="100" y="100" transform={`rotate(${mid - 90} 100 100) translate(30 0)`} fill={won ? "#1a1408" : "#f3f3f5"}
                  fontWeight={won ? 700 : 600} fontSize={n > 18 ? 4.2 : 5.2} dominantBaseline="middle">
                  {g.name.length > 24 ? g.name.slice(0, 23) + "…" : g.name}
                </text>
              </g>
            );
          })}
        </g>
        <circle cx="100" cy="100" r="99.5" fill="none" stroke="var(--accent)" strokeOpacity="0.5" strokeWidth="0.8" />
      </svg>
      <div className="wheel-pointer" aria-hidden />
      <button className="spin-button" onClick={onSpin} disabled={spinning || !entries.length}>Spin</button>
    </div>
  );
}

/** The picked game: cover over a blurred copy, score, hours, status and tags, with the next moves and a card to share. */
function Winner({ game, onPlay, onAgain, onNotForMe }: { game: Game; onPlay: () => void; onAgain: () => void; onNotForMe: () => void }) {
  return (
    <section className="winner" aria-live="polite">
      {game.cover && <div className="backdrop" style={{ backgroundImage: `url(/covers/${game.cover})` }} aria-hidden />}
      <div className="body">
        <div className="head">
          <div className="kicker">The wheel picked</div>
          <ShareButton title="Up next card" fileName={wheelPickFileName(game)} draw={() => drawWheelPickCard(game)} drawKey={String(game.appid)} />
        </div>
        <Cover game={game} />
        <h2 className="name">{game.name}</h2>
        <div className="meta">
          {game.reviewPct !== null && game.reviewCount !== null && (
            <span><b className={`tone-${scoreTone(game.reviewPct)}`}>{game.reviewPct}%</b> · {reviewsShort(game.reviewCount)} reviews</span>
          )}
          <span>{hours(game.minutes)} played</span>
          <span className="chip" style={{ ...statusStyle(game.status), color: "var(--st)" }}><StatusIcon status={game.status} size={12} /> {statusLabel(game.status)}</span>
        </div>
        {list(game.tags).length > 0 && <div className="chips">{list(game.tags).slice(0, 3).map((t) => <span key={t} className="chip accent">{t}</span>)}</div>}
        <div className="toolbar">
          <button className="btn primary" onClick={onPlay}><StatusIcon status={Status.Playing} size={14} /> Mark Playing</button>
          <button className="btn" onClick={onAgain}><span aria-hidden>↻</span> Spin again</button>
          <button className="btn ghost status-action" style={statusStyle(Status.NotForMe)} onClick={onNotForMe}><StatusIcon status={Status.NotForMe} size={14} /> Not for me</button>
        </div>
      </div>
    </section>
  );
}

const ROW = 44;

/** For All: the names scroll past the marker and stop on the winner (W2). */
function Reel({ entries, winner, spins }: { entries: Game[]; winner: number; spins: number }) {
  const ul = useRef<HTMLUListElement>(null);
  // Enough repeats for a long scroll, without rendering a large pool many times over.
  const loops = Math.max(2, Math.ceil(60 / Math.max(entries.length, 1)) + 1);
  // The marker is the third row; the last names fill the two rows above the first.
  const lead = Math.min(2, entries.length);
  const strip = useMemo(() => [...entries.slice(entries.length - lead), ...Array.from({ length: loops }, () => entries).flat()], [entries, lead, loops]);
  useEffect(() => {
    if (winner < 0 || !ul.current) return;
    // Land the winner's last repeat under the marker.
    const to = -((loops - 1) * entries.length + winner) * ROW;
    ul.current.animate([{ transform: "translateY(0)" }, { transform: `translateY(${to}px)` }],
      { duration: spinMs() - 100, easing: "cubic-bezier(0.12, 0.8, 0.12, 1)", fill: "forwards" });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [spins]);
  // A new draw resets the reel to its start.
  useEffect(() => { ul.current?.getAnimations().forEach((a) => a.cancel()); }, [entries]);
  return (
    <div className="reel" aria-label="Name reel">
      <ul ref={ul} style={{ paddingTop: (2 - lead) * ROW }}>
        {strip.map((g, i) => <li key={i}>{g.name}</li>)}
      </ul>
    </div>
  );
}
