import { daysSince, hours, RecService, SettingsService, Status, type Game, type Recommendations } from "../api";
import { FetchBanner, UnclassifiedBanner } from "../components/Banners";
import { Cover, EmptyState } from "../components/bits";
import { PickBackUpButton, PlayingCard, RecentlyFinishedButton } from "../components/GameCards";
import { RecSections } from "../components/RecSections";
import { useGames, useQuery } from "../data";
import type { Route } from "../routes";
import { StatusIcon, statusLabel, statusStyle } from "../statuses";
import { useSetStatus } from "../toast";
import { pctText, useTotals } from "../totals";

export function Home({ go }: { go: (r: Route) => void }) {
  const games = useGames();
  const settings = useQuery(() => SettingsService.Get());
  const recs = useQuery<Recommendations>(() => RecService.Rows("all"));

  const all = games.data ?? [];
  const unclassified = all.filter((g) => !g.classified && !g.hidden).length;
  const playing = all.filter((g) => g.classified && !g.hidden && g.status === Status.Playing);
  const abandonedDays = settings.data?.abandoned_days ?? 60;
  const abandoned = playing.filter((g) => (daysSince(g.lastPlayed) ?? 0) >= abandonedDays);

  return (
    <div className="page">
      <h1>Home</h1>
      <UnclassifiedBanner count={unclassified} go={go} />
      <FetchBanner />
      <Glance go={go} />

      <section aria-labelledby="playing-h" style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        <h2 id="playing-h">Playing</h2>
        {playing.length ? (
          <div className="card-grid playing">{playing.map((g) => <PlayingCard key={g.appid} game={g} />)}</div>
        ) : (
          !games.loading && (
            <EmptyState status={Status.Playing} title="Nothing marked Playing"
              actions={<><button className="btn primary" onClick={() => go("wheel")}>Spin the wheel</button><button className="btn" onClick={() => go("recommendations")}>Recommendations</button></>}>
              Spin the wheel or browse recommendations to choose something.
            </EmptyState>
          )
        )}
      </section>

      {abandoned.map((g) => <AbandonedCard key={g.appid} game={g} />)}

      <RecSections data={recs.data} limit={2} action={<button className="link" onClick={() => go("recommendations")}>All recommendations</button>} />
    </div>
  );
}

/** Finished, Playing and backlog totals, with the Shelved games and Recently finished links. */
function Glance({ go }: { go: (r: Route) => void }) {
  const t = useTotals();
  if (!t) return null;
  const tiles: [Status, React.ReactNode, string][] = [
    [Status.Finished, <>{t.finished.toLocaleString()} <small>/ {t.total.toLocaleString()}</small></>, `finished · ${pctText(t.finishedPct)} of library`],
    [Status.Playing, t.playing.toLocaleString(), "playing"],
    [Status.Unplayed, t.backlog.toLocaleString(), "in the backlog"],
  ];
  return (
    <div className="glance">
      {tiles.map(([s, n, label]) => (
        <div key={s} className="glance-tile" style={statusStyle(s)}>
          <span className="icon"><StatusIcon status={s} size={20} /></span>
          <span className="n">{n}</span>
          <span className="label">{label}</span>
        </div>
      ))}
      <div className="glance-links">
        <PickBackUpButton go={go} />
        <RecentlyFinishedButton go={go} />
      </div>
    </div>
  );
}

/** N1: a Playing game untouched for the abandoned window, with one-click moves. */
function AbandonedCard({ game }: { game: Game }) {
  const setStatus = useSetStatus();
  const days = daysSince(game.lastPlayed);
  return (
    <section className="panel abandoned" aria-label={`Abandoned: ${game.name}`}>
      <Cover game={game} />
      <div style={{ minWidth: 0 }}>
        <div className="kicker"><StatusIcon status={Status.Playing} size={12} /> Abandoned?</div>
        <div className="name">{game.name}</div>
        <div className="faint">Still marked Playing, but untouched for {days === null ? "a long time" : `${days} days`} · {hours(game.minutes)} played</div>
      </div>
      <span className="toolbar">
        {([Status.Shelved, Status.Finished, Status.NotForMe] as const).map((s) => (
          <button key={s} className="btn small status-action" style={statusStyle(s)} onClick={() => setStatus([game.appid], s, game.name)}>
            <StatusIcon status={s} size={14} /> {statusLabel(s)}
          </button>
        ))}
      </span>
    </section>
  );
}
