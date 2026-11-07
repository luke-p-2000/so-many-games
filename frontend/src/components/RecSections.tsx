import { useEffect, useState, type ReactNode } from "react";
import { RecService, recSections, type Recommendations, type RecSectionData } from "../api";
import { useReportError } from "../toast";
import { RecCard } from "./GameCards";
import { useFitCount } from "./layout";

/**
 * R5: Tailored for you, then one section per taste tag. The games Tailored for you is
 * showing are left out of the tag sections, so a game never appears twice on screen.
 */
export function RecSections({ data, limit, action }: { data: Recommendations | undefined; limit?: number; action?: ReactNode }) {
  const [dismissed, setDismissed] = useState<Set<number>>(new Set());
  const [tailoredShowing, setTailoredShowing] = useState<number[]>([]);
  const dismiss = (id: number) => setDismissed((s) => new Set(s).add(id));
  const inTagRows = new Set([...dismissed, ...tailoredShowing]);
  return (
    <>
      {recSections(data).slice(0, limit).map((s) => (
        <RecSection key={s.key} section={s} action={action} onDismiss={dismiss}
          hidden={s.key === "tailored" ? dismissed : inTagRows}
          onShowing={s.key === "tailored" ? setTailoredShowing : undefined} />
      ))}
    </>
  );
}

/** One line of cards, as many as fit (up to 6); Refresh moves on to the section's next games. */
function RecSection({ section, hidden, onDismiss, onShowing, action }: {
  section: RecSectionData; hidden: Set<number>; onDismiss: (appid: number) => void;
  onShowing?: (appids: number[]) => void; action?: ReactNode;
}) {
  const report = useReportError();
  const { ref, count } = useFitCount<HTMLDivElement>(210, 16, 6);
  const [offset, setOffset] = useState(0);
  const games = section.games.filter((r) => !hidden.has(r.game.appid));
  // Wrap around the end so every batch fills the line.
  const start = offset < games.length ? offset : 0;
  const showing = games.length <= count ? games : Array.from({ length: count }, (_, i) => games[(start + i) % games.length]);

  const ids = showing.map((r) => r.game.appid);
  const key = ids.join(",");
  useEffect(() => {
    onShowing?.(ids);
    if (ids.length) RecService.Shown(ids, section.key).catch(report);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  return (
    <section className="panel rec-section" aria-label={section.title} hidden={!games.length}>
      <div className="page-head">
        <div className="section-title">
          <h2>{section.title}</h2>
          {section.note && <span className="subtle"><span aria-hidden>·</span> {section.note}</span>}
        </div>
        <div className="toolbar">
          {action}
          <button className="btn small" disabled={games.length <= count} title="Show the next games in this section"
            onClick={() => setOffset((start + count) % games.length)}>
            <span aria-hidden>↻</span> Refresh
          </button>
        </div>
      </div>
      <div ref={ref} className="card-row" style={{ gridTemplateColumns: `repeat(${count}, minmax(0, 1fr))` }}>
        {showing.map((r) => <RecCard key={r.game.appid} rec={r} onDismiss={onDismiss} />)}
      </div>
    </section>
  );
}
