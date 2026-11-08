import { useState } from "react";
import { finishedDateText, list, StatsService, Status, type Bar, type Stats as S } from "../api";
import { BarList, Cover, Segmented } from "../components/bits";
import { useQuery } from "../data";
import { STATUSES, StatusIcon, statusStyle } from "../statuses";

const hoursText = (v: number) => Math.round(v).toLocaleString();
const pct = (n: number, total: number) => {
  const p = total ? (n / total) * 100 : 0;
  return `${p < 10 ? p.toFixed(1) : Math.round(p)}%`;
};

/** T1: the finished counter, status breakdown, hours by genre and tag, and most played. */
export function Stats() {
  const stats = useQuery<S>(() => StatsService.Stats());
  const [range, setRange] = useState<"all" | "recent">("all");
  const s = stats.data;
  if (!s) return <div className="page"><h1>Stats</h1>{stats.error && <p className="warn-text">{stats.error}</p>}</div>;

  const counts = new Map(list(s.statusCounts).map((b) => [b.label, b.value]));
  const total = [...counts.values()].reduce((a, b) => a + b, 0);
  const rangeLabel = range === "all" ? "All-time" : "Recent";
  const hourBars = (bars: Bar[] | null) => list(bars).map((b) => ({ key: b.label, label: b.label, value: b.value }));

  return (
    <div className="page">
      <div className="page-head">
        <h1>Stats</h1>
        <Segmented label="Hours range" value={range} onChange={setRange} options={[{ value: "all", label: "All-time" }, { value: "recent", label: "Recent" }]} />
      </div>
      <p className="subtle" style={{ marginTop: -8 }}>Classified, visible games only.</p>

      <FinishedCounter finished={counts.get(Status.Finished) ?? 0} total={total} latest={s.latestFinished} />

      <section className="panel stats-card" aria-labelledby="status-h">
        <div className="stats-card-head"><h3 id="status-h">Status breakdown</h3><span className="faint">{total.toLocaleString()} games</span></div>
        <div className="status-stack" role="img" aria-label={STATUSES.map((st) => `${st.label} ${counts.get(st.value) ?? 0}`).join(", ")}>
          {STATUSES.filter((st) => counts.get(st.value)).map((st) => (
            <div key={st.value} style={{ ...statusStyle(st.value), flexGrow: counts.get(st.value) }} title={`${st.label}: ${counts.get(st.value)}`} />
          ))}
        </div>
        <div className="status-tiles">
          {STATUSES.map((st) => {
            const n = counts.get(st.value) ?? 0;
            return (
              <div key={st.value} className={`status-tile${n ? "" : " zero"}`} style={statusStyle(st.value)}>
                <span className="icon"><StatusIcon status={st.value} size={22} /></span>
                <span className="name">{st.label}</span>
                <span className="count"><b>{n.toLocaleString()}</b><span>{pct(n, total)}</span></span>
              </div>
            );
          })}
        </div>
        <p className="faint" style={{ margin: 0, fontSize: 14 }}>Backlog: <b className="strong">{s.backlogSize}</b> games, <b className="strong">{hoursText(s.backlogHours)} h</b> already played.</p>
      </section>

      <div className="stats-two">
        <Card title="Hours by genre" note={<span className="chip">{rangeLabel}</span>}>
          <BarList bars={hourBars(range === "all" ? s.genreHoursAll : s.genreHoursRecent)} format={hoursText} unit="h" />
        </Card>
        <Card title="Hours by tag" note={<span className="chip">{rangeLabel}</span>}>
          <BarList bars={hourBars(range === "all" ? s.tagHoursAll : s.tagHoursRecent)} format={hoursText} unit="h" />
        </Card>
      </div>

      <Card title="Most played" note={<span className="faint">Total hours</span>}>
        <BarList
          bars={list(s.mostPlayed).map((g) => ({ key: String(g.appid), label: g.name, value: g.minutes / 60, lead: <Cover game={g} className="thumb" /> }))}
          format={hoursText}
          unit="h"
        />
      </Card>
    </div>
  );
}

function FinishedCounter({ finished, total, latest }: { finished: number; total: number; latest: S["latestFinished"] }) {
  return (
    <section className="panel finished-counter" aria-label="Games finished">
      <div className="top">
        <span className="ring" aria-hidden><StatusIcon status={Status.Finished} size={44} /></span>
        <div>
          <div className="kicker">Games finished</div>
          <div className="big">{finished.toLocaleString()} <small>/ {total.toLocaleString()}</small></div>
        </div>
        <div className="vr" />
        <div className="pct"><div className="big">{pct(finished, total)}</div><div className="lbl">of your library</div></div>
      </div>
      <div className="meter" role="progressbar" aria-valuemin={0} aria-valuemax={total} aria-valuenow={finished} aria-label="Finished out of all games">
        <div style={{ width: `${total ? (finished / total) * 100 : 0}%` }} />
      </div>
      <div className="foot">
        <span>
          {latest ? <>Latest: <b>{latest.name}</b> · {finishedDateText(latest.finishedAt)}</>
            : finished ? "Add finish dates in Recently finished to see your latest." : "Nothing finished yet."}
        </span>
        <span>{(total - finished).toLocaleString()} to go</span>
      </div>
    </section>
  );
}

function Card({ title, note, children }: { title: string; note?: React.ReactNode; children: React.ReactNode }) {
  return (
    <section className="panel stats-card">
      <div className="stats-card-head"><h3>{title}</h3>{note}</div>
      {children}
    </section>
  );
}
