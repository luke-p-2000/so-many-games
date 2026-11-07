import type { ReactNode } from "react";
import { DataService, hours, list, LibraryService, MetaService, RecService, reviewText, shortReview, Status, type Game, type Ranked } from "../api";
import { useRefresh } from "../data";
import { useReportError, useSetStatus } from "../toast";
import type { Route } from "../routes";
import type { StatusValue } from "../statuses";
import { Cover, Rating } from "./bits";
import { ClampedChips } from "./layout";
import { StatusRadio } from "./StatusRadio";

/** Completed checkbox and rating, shown only on Finished games (L1, L4). */
export function FinishedExtras({ game }: { game: Game }) {
  const refresh = useRefresh();
  const report = useReportError();
  if (game.status !== Status.Finished) return null;
  return (
    <>
      <label className="check">
        <input type="checkbox" checked={game.completed} onChange={(e) => LibraryService.SetCompleted(game.appid, e.target.checked).then(refresh, report)} />
        Completed
      </label>
      <Rating value={game.rating} name={game.name} onChange={(v) => LibraryService.SetRating(game.appid, v).then(refresh, report)} />
    </>
  );
}

type RowProps = {
  game: Game;
  sub?: ReactNode;
  numbers?: ReactNode;
  /** Classify passes staged state; elsewhere changes apply at once. */
  status?: Status;
  onStatus?: (s: StatusValue) => void;
  selected?: boolean;
  onSelect?: (on: boolean) => void;
  extra?: ReactNode;
};

/** Row card used by Library, Pick back up and Classify. */
export function GameRow({ game, sub, numbers, status, onStatus, selected, onSelect, extra }: RowProps) {
  const setStatus = useSetStatus();
  return (
    <article className={`game-row${selected ? " selected" : ""}`} style={onSelect ? undefined : { gridTemplateColumns: "132px minmax(0, 1fr) auto auto" }}>
      {onSelect && <input type="checkbox" aria-label={`Select ${game.name}`} checked={!!selected} onChange={(e) => onSelect(e.target.checked)} />}
      <Cover game={game} />
      <div style={{ minWidth: 0 }}>
        <div className="name">{game.name}</div>
        {sub && <div className="sub">{sub}</div>}
        {!onStatus && game.status === Status.Finished && <div className="finished-line"><FinishedExtras game={game} /></div>}
      </div>
      <div className="numbers">{numbers}</div>
      <div className="extras">
        {extra}
        <StatusRadio value={status ?? game.status} gameName={game.name} onChange={onStatus ?? ((s) => setStatus([game.appid], s, game.name))} />
      </div>
    </article>
  );
}

/** Library row sub-line: tags, genres and review summary. */
export function tagLine(g: Game): string {
  const parts = list(g.tags).slice(0, 3);
  if (!parts.length) parts.push(...list(g.genres).slice(0, 3));
  const r = shortReview(g.reviewPct, g.reviewCount);
  if (r) parts.push(r);
  if (!g.hasMeta) parts.push(g.metaUnavailable ? "no store data (marked unavailable)" : "no store data yet");
  return parts.join(" · ");
}

/** Home card for a Playing game (H1). */
export function PlayingCard({ game }: { game: Game }) {
  const setStatus = useSetStatus();
  return (
    <article className="game-card">
      <Cover game={game} />
      <div className="title">{game.name}</div>
      <div className="meta-line"><span>{hours(game.minutes)} played</span></div>
      <StatusRadio value={game.status} gameName={game.name} onChange={(s) => setStatus([game.appid], s, game.name)} />
    </article>
  );
}

/** Compressed grid tile (L5) with the compact status control. */
export function GridTile({ game }: { game: Game }) {
  const setStatus = useSetStatus();
  return (
    <article className="game-card">
      <Cover game={game} />
      <div className="title">{game.name}</div>
      <div className="meta-line"><span>{hours(game.minutes)}</span></div>
      <div className="actions">
        <StatusRadio mode="compact" value={game.status} gameName={game.name} onChange={(s) => setStatus([game.appid], s, game.name)} />
      </div>
    </article>
  );
}

/** Recommendation card: raw %, review count and matched tags (R4), opened and dismissed are logged (E1). */
export function RecCard({ rec, onDismiss }: { rec: Ranked; onDismiss: (appid: number) => void }) {
  const setStatus = useSetStatus();
  const report = useReportError();
  const g = rec.game;
  const open = () => {
    RecService.Opened(g.appid).catch(report);
    DataService.OpenStorePage(g.appid).catch(report);
  };
  return (
    <article className="game-card rec-card">
      <button className="cover-button" onClick={open} aria-label={`Open ${g.name} on the Steam store`}>
        <Cover game={g} />
      </button>
      <div className="title" title={g.name}>{g.name}</div>
      <div className="faint">{reviewText(g.reviewPct, g.reviewCount)}</div>
      <ClampedChips items={list(rec.matched)} accent />
      <div className="actions">
        <StatusRadio mode="compact" value={g.status} gameName={g.name} onChange={(s) => setStatus([g.appid], s, g.name)} />
        <button
          className="btn ghost small"
          onClick={() => {
            RecService.Dismissed(g.appid).catch(report);
            onDismiss(g.appid);
          }}
        >
          Not now
        </button>
      </div>
    </article>
  );
}

/** Undo the "unavailable" mark from a Library row (ADR 0003). */
export function UnavailableToggle({ game }: { game: Game }) {
  const refresh = useRefresh();
  const report = useReportError();
  if (!game.metaUnavailable) return null;
  return <button className="btn small ghost" onClick={() => MetaService.SetUnavailable(game.appid, false).then(refresh, report)}>Fetch again</button>;
}

/** Opens Pick back up (R7): Shelved and Endless games you stopped playing. */
export function PickBackUpButton({ go }: { go: (r: Route) => void }) {
  return (
    <button className="btn" onClick={() => go("pickbackup")} title="Shelved and Endless games you stopped playing, longest untouched first">
      <span aria-hidden>↺</span> Resume shelved games
    </button>
  );
}
