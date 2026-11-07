import { useState } from "react";
import { EVENTS, list, MetaService, type Banner, type Progress } from "../api";
import { useEvent, useQuery, useRefresh } from "../data";
import { useReportError } from "../toast";
import type { Route } from "../routes";

/** S4–S6: "No data for N games", the fetch progress with one pause/resume button, and failures. */
export function FetchBanner() {
  const banner = useQuery<Banner>(() => MetaService.Banner());
  const [live, setLive] = useState<Progress | null>(null);
  const [showFailures, setShowFailures] = useState(false);
  const refresh = useRefresh();
  const report = useReportError();
  useEvent<Progress>(EVENTS.fetchProgress, (p) => setLive(p.running ? p : null));

  const b = banner.data;
  if (!b) return null;
  const failures = list(b.failures);
  const progress = live ?? (b.progress.running ? b.progress : null);
  const run = (fn: () => Promise<unknown>) => fn().then(refresh, report);

  if (progress) {
    const pct = progress.total ? (progress.done / progress.total) * 100 : 0;
    return (
      <section className="panel" aria-label="Fetching store data">
        <div className="banner" style={{ padding: 0, marginBottom: 12 }}>
          <span>
            {progress.paused ? "Paused" : "Fetching"} <strong>{progress.current || "store data"}</strong>
            <span className="subtle"> · {progress.done} of {progress.total}</span>
          </span>
          <button className="btn" onClick={() => MetaService.TogglePause().then((p) => setLive(p), report)}>
            {progress.paused ? "Resume" : "Pause"}
          </button>
        </div>
        <div className="progress" role="progressbar" aria-valuemin={0} aria-valuemax={progress.total} aria-valuenow={progress.done}>
          <div style={{ width: `${pct}%` }} />
        </div>
      </section>
    );
  }

  if (b.missing === 0 && failures.length === 0) return null;
  return (
    <section className="panel warn" aria-label="Missing store data">
      <div className="banner" style={{ padding: 0 }}>
        <span>
          {b.missing > 0 ? (
            <><span className="warn-text">⚠ No data for {b.missing} {b.missing === 1 ? "game" : "games"}</span> — genres, tags and reviews are missing.</>
          ) : (
            <span className="subtle">Store data is up to date.</span>
          )}
          {failures.length > 0 && (
            <> <button className="link" onClick={() => setShowFailures((s) => !s)}>{failures.length} failed</button></>
          )}
        </span>
        {b.missing > 0 && <button className="btn primary" onClick={() => run(() => MetaService.Fetch())}>Fetch store data</button>}
      </div>
      {showFailures && failures.length > 0 && (
        <div style={{ marginTop: 16, display: "flex", flexDirection: "column", gap: 8 }}>
          {failures.map((f) => (
            <div key={f.appid} className="banner" style={{ padding: "8px 0", borderTop: "1px solid var(--border)" }}>
              <span><strong>{f.name}</strong> <span className="faint">— {f.reason}</span></span>
              <span className="toolbar">
                <button className="btn small" onClick={() => run(() => MetaService.Refetch([f.appid]))}>Retry</button>
                <button className="btn small ghost" onClick={() => run(() => MetaService.SetUnavailable(f.appid, true))}>Mark unavailable</button>
              </span>
            </div>
          ))}
          <div><button className="btn small" onClick={() => run(() => MetaService.RetryAll())}>Retry all</button></div>
        </div>
      )}
    </section>
  );
}

/** L8: how many games still wait on the Classify screen. */
export function UnclassifiedBanner({ count, go }: { count: number; go: (r: Route) => void }) {
  if (count === 0) return null;
  return (
    <section className="panel banner">
      <span>
        <strong>{count} {count === 1 ? "game is" : "games are"} unclassified</strong>
        <span className="subtle"> — they are left out of the wheel, recommendations and stats until you classify them.</span>
      </span>
      <button className="btn primary" onClick={() => go("classify")}>Open Classify</button>
    </section>
  );
}
