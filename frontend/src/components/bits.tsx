// Small shared presentational pieces.
import { useState } from "react";
import type { Bar, Game } from "../api";

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

/** Horizontal bar chart used on the Stats screen. */
export function BarList({ bars, format = (v) => v.toLocaleString() }: { bars: Bar[]; format?: (v: number) => string }) {
  if (!bars.length) return <p className="faint">Nothing yet.</p>;
  const max = Math.max(...bars.map((b) => b.value), 1);
  return (
    <div className="bars">
      {bars.map((b) => (
        <div key={b.label} style={{ display: "contents" }}>
          <span>{b.label}</span>
          <div className="bar" style={{ width: `${(b.value / max) * 100}%` }} />
          <span className="num">{format(b.value)}</span>
        </div>
      ))}
    </div>
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
