import { useMemo, useState } from "react";
import { hours, lastPlayedText, list, RecService, Status, type BackUp } from "../api";
import { EmptyState, Segmented } from "../components/bits";
import { GameRow } from "../components/GameCards";
import { useQuery } from "../data";

/** R7: every Shelved and Endless game at any hours. */
export function PickBackUp() {
  const backUp = useQuery<BackUp[]>(() => RecService.PickBackUp().then(list));
  const [order, setOrder] = useState<"untouched" | "hours">("untouched");
  const items = useMemo(() => {
    const l = [...(backUp.data ?? [])];
    return order === "hours" ? l.sort((a, b) => b.game.minutes - a.game.minutes) : l;
  }, [backUp.data, order]);

  return (
    <div className="page">
      <h1>Shelved games</h1>
      <p className="subtle">Every Shelved and Endless game you started and put down, at any hours. Mark one Playing to pick it back up.</p>
      <Segmented label="Order" value={order} onChange={setOrder} options={[{ value: "untouched", label: "Longest untouched" }, { value: "hours", label: "Most hours" }]} />
      {!backUp.loading && items.length === 0 && (
        <EmptyState status={Status.Shelved} title="No Shelved or Endless games">Games you start and put down will gather here, ready to pick back up.</EmptyState>
      )}
      <div className="row-list">
        {items.map(({ game, matched }) => (
          <GameRow
            key={game.appid}
            game={game}
            sub={[...(list(matched).length ? list(matched) : list(game.tags).slice(0, 2)), lastPlayedText(game.lastPlayed)].join(" · ")}
            numbers={<span><b>{hours(game.minutes)}</b></span>}
          />
        ))}
      </div>
    </div>
  );
}
