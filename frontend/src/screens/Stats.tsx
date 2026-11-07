import { useState } from "react";
import { list, StatsService, type Stats as S } from "../api";
import { BarList, Segmented } from "../components/bits";
import { useQuery } from "../data";
import { statusLabel } from "../statuses";
import type { Status } from "../api";

const h = (v: number) => `${Math.round(v).toLocaleString()} h`;

/** T1. History-based charts count since first sync. */
export function Stats() {
  const stats = useQuery<S>(() => StatsService.Stats());
  const [range, setRange] = useState<"all" | "recent">("all");
  const s = stats.data;
  if (!s) return <div className="page"><h1>Stats</h1>{stats.error && <p className="warn-text">{stats.error}</p>}</div>;

  const since = s.firstSyncAt ? new Date(s.firstSyncAt).toLocaleDateString() : "your first sync";
  return (
    <div className="page">
      <div className="page-head">
        <h1>Stats</h1>
        <Segmented label="Hours range" value={range} onChange={setRange} options={[{ value: "all", label: "All-time" }, { value: "recent", label: "Recent" }]} />
      </div>
      <p className="subtle">Classified, visible games only. History counts since first sync ({since}).</p>
      <div className="card-grid" style={{ gridTemplateColumns: "repeat(auto-fill, minmax(420px, 1fr))" }}>
        <Card title="Status breakdown">
          <BarList bars={list(s.statusCounts).map((b) => ({ ...b, label: statusLabel(b.label as Status) }))} />
          <p className="faint" style={{ marginTop: 12 }}>Backlog: <b>{s.backlogSize}</b> games, {h(s.backlogHours)} already played.</p>
        </Card>
        <Card title={`Hours by tag (${range === "all" ? "all-time" : "recent"})`}>
          <BarList bars={list(range === "all" ? s.tagHoursAll : s.tagHoursRecent)} format={h} />
        </Card>
        <Card title={`Hours by genre (${range === "all" ? "all-time" : "recent"})`}>
          <BarList bars={list(range === "all" ? s.genreHoursAll : s.genreHoursRecent)} format={h} />
        </Card>
        <Card title="Finished per year">
          <BarList bars={list(s.finishedPerYear)} />
        </Card>
        <Card title="Rating distribution">
          <BarList bars={list(s.ratings)} />
        </Card>
        <Card title="Most played">
          <BarList bars={list(s.mostPlayed)} format={h} />
        </Card>
        <Card title={`Backlog burn-down (since first sync)`}>
          <BarList bars={list(s.burnDown)} />
        </Card>
      </div>
    </div>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="panel" style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <h3>{title}</h3>
      {children}
    </section>
  );
}
