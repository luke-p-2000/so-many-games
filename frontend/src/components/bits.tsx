// Small shared presentational pieces.
import { useState, type ReactNode } from "react";
import type { Game, Status } from "../api";
import { StatusIcon, statusStyle } from "../statuses";

/** Cached cover art, or the game's initials when none was fetched. */
export function Cover({ game, className = "" }: { game: Pick<Game, "cover" | "name">; className?: string }) {
  const [failed, setFailed] = useState(false);
  const initials = game.name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]).join("").toUpperCase();
  return (
    <div className={`cover ${className}`} aria-hidden>
      {game.cover && !failed ? <img src={`/covers/${game.cover}`} alt="" loading="lazy" onError={() => setFailed(true)} /> : initials}
    </div>
  );
}

/** 1-10 rating for Finished games (L4). Clicking the current value clears it. */
export function Rating({ value, onChange, name }: { value: number | null; onChange: (v: number | null) => void; name: string }) {
  return (
    <div className="rating" role="group" aria-label={`Rating for ${name}`}>
      {Array.from({ length: 10 }, (_, i) => i + 1).map((n) => (
        <button key={n} type="button" aria-pressed={value === n} aria-label={`Rate ${n}`} onClick={() => onChange(value === n ? null : n)}>
          {n}
        </button>
      ))}
    </div>
  );
}

/** One bar in a ranked bar list; lead is shown before the label, such as a cover thumbnail. */
export type RankedBar = { key: string; label: string; value: number; lead?: ReactNode };

/** Ranked horizontal bars used on the Stats screen: rank, label, bar on a track, value. */
export function BarList({ bars, format = (v) => v.toLocaleString(), unit }: { bars: RankedBar[]; format?: (v: number) => string; unit?: string }) {
  if (!bars.length) return <p className="faint">Nothing yet.</p>;
  const max = Math.max(...bars.map((b) => b.value), 1);
  const lead = bars.some((b) => b.lead);
  return (
    <div className={`bars${lead ? " with-lead" : ""}`}>
      {bars.map((b, i) => (
        <div key={b.key} className="bar-row">
          <span className={`rank${i < 3 ? " top" : ""}`}>{i + 1}</span>
          {lead && b.lead}
          <span className="label" title={b.label}>{b.label}</span>
          <div className="track"><div style={{ width: `${(b.value / max) * 100}%` }} /></div>
          <span className="num">{format(b.value)}{unit && <span> {unit}</span>}</span>
        </div>
      ))}
    </div>
  );
}

/** An empty list: a large status icon in its colour, a title, a sentence and a way forward. */
export function EmptyState({ status, title, children, actions }: { status: Status; title: string; children?: ReactNode; actions?: ReactNode }) {
  return (
    <section className="panel empty-state" style={statusStyle(status)}>
      <div className="halo" aria-hidden><StatusIcon status={status} size={36} /></div>
      <h3>{title}</h3>
      {children && <p>{children}</p>}
      {actions && <div className="toolbar">{actions}</div>}
    </section>
  );
}

export function Segmented<T extends string>({ value, options, onChange, label }: { value: T; options: { value: T; label: string }[]; onChange: (v: T) => void; label: string }) {
  return (
    <div className="segmented" role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o.value} type="button" aria-pressed={o.value === value} onClick={() => onChange(o.value)}>{o.label}</button>
      ))}
    </div>
  );
}
