import { useRef, useState, type KeyboardEvent } from "react";
import type { Status } from "../api";
import { STATUSES, StatusIcon, statusLabel, statusStyle, type StatusValue } from "../statuses";

type Props = {
  value: Status;
  onChange: (s: StatusValue) => void;
  gameName: string;
  /** full: one icon radio per status; compact: the current icon, expanding on hover or click (L3). */
  mode?: "full" | "compact";
};

/** The one status control used everywhere (design: StatusRadio). Arrow keys move the choice. */
export function StatusRadio({ value, onChange, gameName, mode = "full" }: Props) {
  const [open, setOpen] = useState(false);
  const group = useRef<HTMLDivElement>(null);

  const onKey = (e: KeyboardEvent) => {
    const i = STATUSES.findIndex((s) => s.value === value);
    const step = e.key === "ArrowRight" || e.key === "ArrowDown" ? 1 : e.key === "ArrowLeft" || e.key === "ArrowUp" ? -1 : 0;
    if (!step) return;
    e.preventDefault();
    const next = STATUSES[(i + step + STATUSES.length) % STATUSES.length];
    onChange(next.value);
    requestAnimationFrame(() => group.current?.querySelector<HTMLButtonElement>(`[data-status="${next.value}"]`)?.focus());
  };

  const radios = (
    <div className="status-radio" role="radiogroup" aria-label={`Status of ${gameName}`} ref={group} onKeyDown={onKey}>
      {STATUSES.map((s) => (
        <button
          key={s.value}
          type="button"
          role="radio"
          className="status-btn"
          data-status={s.value}
          style={statusStyle(s.value)}
          aria-checked={s.value === value}
          aria-label={s.label}
          tabIndex={s.value === value ? 0 : -1}
          onClick={() => {
            if (s.value !== value) onChange(s.value);
            setOpen(false);
          }}
        >
          <StatusIcon status={s.value} />
          <span className="tip" aria-hidden>{s.label}</span>
        </button>
      ))}
    </div>
  );

  if (mode === "full") return radios;
  return (
    <div className="status-compact" onMouseLeave={() => setOpen(false)} onBlur={(e) => !e.currentTarget.contains(e.relatedTarget) && setOpen(false)}>
      <button
        type="button"
        className="status-btn"
        aria-haspopup="true"
        aria-expanded={open}
        style={statusStyle(value)}
        aria-label={`${statusLabel(value)}: change status of ${gameName}`}
        onMouseEnter={() => setOpen(true)}
        onClick={() => setOpen((o) => !o)}
      >
        <StatusIcon status={value} />
      </button>
      {open && radios}
    </div>
  );
}
