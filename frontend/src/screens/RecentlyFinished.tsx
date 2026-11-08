import { byFinishedNewest, finishedDateText, finishedNumbers, hours, Status } from "../api";
import { EmptyState } from "../components/bits";
import { GameRow } from "../components/GameCards";
import { useGames } from "../data";

/** Every Finished game, newest finish first; unknown dates last (ADR 0016). */
export function RecentlyFinished() {
  const games = useGames();
  const all = games.data ?? [];
  const items = byFinishedNewest(all);
  const numbers = finishedNumbers(all);

  return (
    <div className="page">
      <h1>Recently finished</h1>
      <p className="subtle">Every Finished game, newest first. Set the date on games you finished before tracking, and use Card to copy or save a finished image.</p>
      {!games.loading && items.length === 0 && (
        <EmptyState status={Status.Finished} title="No Finished games yet">Finish a game to start your tier list. Each one gets a card to share.</EmptyState>
      )}
      <div className="row-list">
        {items.map((g) => (
          <GameRow
            key={g.appid}
            game={g}
            sub={`${finishedDateText(g.finishedAt)} · game #${numbers.get(g.appid)} finished`}
            numbers={<span><b>{hours(g.minutes)}</b></span>}
          />
        ))}
      </div>
    </div>
  );
}
