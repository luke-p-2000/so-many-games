import { daysSince, RecService, SettingsService, Status, type Game, type Recommendations } from "../api";
import { FetchBanner, UnclassifiedBanner } from "../components/Banners";
import { PickBackUpButton, PlayingCard } from "../components/GameCards";
import { RecSections } from "../components/RecSections";
import { useGames, useQuery } from "../data";
import type { Route } from "../routes";
import { useSetStatus } from "../toast";

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

      <section aria-labelledby="playing-h" style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        <div className="page-head">
          <h2 id="playing-h">Playing</h2>
          <PickBackUpButton go={go} />
        </div>
        {playing.length ? (
          <div className="card-grid playing">{playing.map((g) => <PlayingCard key={g.appid} game={g} />)}</div>
        ) : (
          <p className="subtle">Nothing marked Playing. Spin the wheel or browse recommendations to choose something.</p>
        )}
      </section>

      {abandoned.map((g) => <AbandonedBadge key={g.appid} game={g} />)}

      <RecSections data={recs.data} limit={2} action={<button className="link" onClick={() => go("recommendations")}>All recommendations</button>} />
    </div>
  );
}

/** N1: a Playing game untouched for the abandoned window, with one-click moves. */
function AbandonedBadge({ game }: { game: Game }) {
  const setStatus = useSetStatus();
  const days = daysSince(game.lastPlayed);
  return (
    <section className="panel banner" aria-label={`Abandoned: ${game.name}`}>
      <span>
        <span className="chip accent" style={{ marginRight: 10 }}>Abandoned?</span>
        <strong>{game.name}</strong>
        <span className="subtle"> — untouched for {days === null ? "a long time" : `${days} days`}.</span>
      </span>
      <span className="toolbar">
        <button className="btn small" onClick={() => setStatus([game.appid], Status.Shelved, game.name)}>Shelved</button>
        <button className="btn small" onClick={() => setStatus([game.appid], Status.Finished, game.name)}>Finished</button>
        <button className="btn small" onClick={() => setStatus([game.appid], Status.NotForMe, game.name)}>Not for me</button>
      </span>
    </section>
  );
}
