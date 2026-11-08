import { hours, lastPlayedText, Status } from "../api";
import { EmptyState } from "../components/bits";
import { GameRow, tagLine } from "../components/GameCards";
import { useGames } from "../data";

/** ADR 0013: games marked Dead, reached from Settings. Change the status to bring one back. */
export function Dead() {
  const games = useGames();
  const dead = (games.data ?? []).filter((g) => g.status === Status.Dead && !g.hidden);
  return (
    <div className="page">
      <h1>Dead games</h1>
      <p className="subtle">Games you marked Dead — servers shut down or no longer playable. They never appear on the wheel, in recommendations or in Shelved games, and syncs never change them.</p>
      {!games.loading && dead.length === 0 && <EmptyState status={Status.Dead} title="No games are marked Dead">Mark a game Dead when it can no longer be played.</EmptyState>}
      <div className="row-list">
        {dead.map((g) => (
          <GameRow key={g.appid} game={g} sub={tagLine(g)}
            numbers={<><span><b>{hours(g.minutes)}</b></span><span>{lastPlayedText(g.lastPlayed)}</span></>} />
        ))}
      </div>
    </div>
  );
}
