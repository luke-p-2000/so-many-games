import type { ReactNode } from "react";
import { DataService, hours, lastPlayedText, list, LibraryService, MetaService, RecService, shortReview, Status, type Game, type Ranked } from "../api";
import { useRefresh } from "../data";
import { useReportError, useSetStatus } from "../toast";
import type { Route } from "../routes";
import { StatusIcon, statusLabel, statusStyle, type StatusValue } from "../statuses";
import { Cover, Rating } from "./bits";
import { CardButton, FinishedDate } from "./FinishedCard";
import { ClampedChips } from "./layout";
import { StatusRadio } from "./StatusRadio";

/** Finished date, Completed checkbox, rating and the card button, shown only on Finished games (L1, L4, ADR 0016). */
export function FinishedExtras({ game }: { game: Game }) {
  const refresh = useRefresh();
  const report = useReportError();
  if (game.status !== Status.Finished) return null;
  return (
    <>
      <FinishedDate game={game} />
      <label className="check">
        <input type="checkbox" checked={game.completed} onChange={(e) => LibraryService.SetCompleted(game.appid, e.target.checked).then(refresh, report)} />
        Completed
      </label>
      <Rating value={game.rating} name={game.name} onChange={(v) => LibraryService.SetRating(game.appid, v).then(refresh, report)} />
      <CardButton game={game} />
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
    <article className={`game-row${selected ? " selected" : ""}`}
      style={{ ...statusStyle(status ?? game.status), ...(onSelect ? undefined : { gridTemplateColumns: "132px minmax(0, 1fr) auto auto" }) }}>
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

/** Home card for a Playing game (H1): the cover over a blurred copy of itself. */
export function PlayingCard({ game }: { game: Game }) {
  const setStatus = useSetStatus();
  const last = game.lastPlayed ? lastPlayedText(game.lastPlayed).replace(/^(last )?played /, "") : "never";
  return (
    <article className="playing-card">
      {game.cover && <div className="backdrop" style={{ backgroundImage: `url(/covers/${game.cover})` }} aria-hidden />}
      <div className="body">
        <div className="art">
          <Cover game={game} />
          <span className="now-badge"><StatusIcon status={Status.Playing} size={12} /> Playing</span>
        </div>
        <div className="title">{game.name}</div>
        <div className="stats"><span><b className="h">{hours(game.minutes)}</b> played</span><span>Last played <b>{last}</b></span></div>
        <StatusRadio value={game.status} gameName={game.name} onChange={(s) => setStatus([game.appid], s, game.name)} />
      </div>
    </article>
  );
}

/** Compressed grid tile (L5): the name over the cover, then the status control and hours. */
export function GridTile({ game }: { game: Game }) {
  const setStatus = useSetStatus();
  return (
    <article className="grid-tile" style={statusStyle(game.status)}>
      <div className="art">
        <Cover game={game} />
        <span className="title" title={game.name}>{game.name}</span>
      </div>
      <div className="foot">
        <span className="status">
          <StatusRadio mode="compact" value={game.status} gameName={game.name} onChange={(s) => setStatus([game.appid], s, game.name)} />
          {statusLabel(game.status)}
        </span>
        <span className="h">{hours(game.minutes)}</span>
      </div>
    </article>
  );
}

export const reviewsShort = (n: number) => (n >= 1e6 ? `${(n / 1e6).toFixed(1).replace(/\.0$/, "")}M` : n >= 1000 ? `${Math.round(n / 1000)}k` : String(n));
export const scoreTone = (p: number) => (p >= 90 ? "great" : p >= 80 ? "good" : "ok");

/** Recommendation card: review score on the cover, review count and matched tags (R4); opened and dismissed are logged (E1). */
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
        {g.reviewPct !== null && <span className={`score ${scoreTone(g.reviewPct)}`}><i aria-hidden />{g.reviewPct}%</span>}
      </button>
      <div className="title" title={g.name}>{g.name}</div>
      {g.reviewCount !== null && <div className="faint" style={{ marginTop: -6 }}>{reviewsShort(g.reviewCount)} reviews</div>}
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

/** Opens Pick back up (R7, labelled Shelved games): Shelved and Endless games you stopped playing. */
export function PickBackUpButton({ go }: { go: (r: Route) => void }) {
  return (
    <button className="btn" onClick={() => go("pickbackup")} title="Shelved and Endless games you stopped playing, longest untouched first">
      <span className="st-icon" style={statusStyle(Status.Shelved)}><StatusIcon status={Status.Shelved} size={14} /></span> Shelved games
    </button>
  );
}

/** Opens Recently finished: Finished games, newest finish first (ADR 0016). */
export function RecentlyFinishedButton({ go }: { go: (r: Route) => void }) {
  return (
    <button className="btn" onClick={() => go("finished")} title="Finished games, newest first, with their cards">
      <span className="st-icon" style={statusStyle(Status.Finished)}><StatusIcon status={Status.Finished} size={14} /></span> Recently finished
    </button>
  );
}
