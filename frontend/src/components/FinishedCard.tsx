// The finished-game card dialog (ADR 0016): shown when one game is set to Finished, and
// from a Finished game's Card button. It previews the card and copies or saves it.
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { DataService, errorText, finishedNumbers, LibraryService, Status, todayISO, type Game } from "../api";
import { useGames, useRefresh } from "../data";
import { cardBlob, cardFileName, drawFinishedCard } from "../finishedCard";
import { FinishedCardCtx, useShowFinishedCard } from "../finishedCardContext";
import { useReportError, useToast } from "../toast";
import { Rating } from "./bits";

export function FinishedCardProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState<{ appid: number; justFinished: boolean } | null>(null);
  const show = useCallback((appid: number, justFinished = false) => setOpen({ appid, justFinished }), []);
  return (
    <FinishedCardCtx.Provider value={show}>
      {children}
      {open && <FinishedDialog key={open.appid} {...open} onClose={() => setOpen(null)} />}
    </FinishedCardCtx.Provider>
  );
}

/** Finished date picker for backfilling; empty means unknown. */
export function FinishedDate({ game }: { game: Game }) {
  const refresh = useRefresh();
  const report = useReportError();
  return (
    <label className="check finished-date">
      Finished
      <input
        type="date"
        className="input"
        value={game.finishedAt ?? ""}
        max={todayISO()}
        aria-label={`Date ${game.name} was finished`}
        onChange={(e) => LibraryService.SetFinishedAt(game.appid, e.target.value).then(refresh, (err) => { report(err); refresh(); })}
      />
    </label>
  );
}

/** Opens the card dialog from a Finished game's row. */
export function CardButton({ game }: { game: Game }) {
  const show = useShowFinishedCard();
  return (
    <button type="button" className="btn small" onClick={() => show(game.appid)} title="Image to copy or save showing this game is finished">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
        <rect x="3" y="5" width="18" height="14" rx="2" /><path d="M7.5 12.5l3 3 6-6" />
      </svg>
      Card
    </button>
  );
}

function FinishedDialog({ appid, justFinished, onClose }: { appid: number; justFinished: boolean; onClose: () => void }) {
  const games = useGames();
  const refresh = useRefresh();
  const report = useReportError();
  const toast = useToast();
  const ref = useRef<HTMLDialogElement>(null);
  const canvas = useRef<HTMLCanvasElement | null>(null);
  const [preview, setPreview] = useState("");

  const all = games.data ?? [];
  const game = all.find((g) => g.appid === appid);
  const number = finishedNumbers(all).get(appid);
  const finished = game?.status === Status.Finished;

  // The dialog renders once the game has loaded, so open it then.
  useEffect(() => {
    if (ref.current && !ref.current.open) ref.current.showModal();
  }, [game]);
  // Undo or another change moved the game off Finished: the card no longer applies.
  useEffect(() => {
    if (games.data && !finished) onClose();
  }, [games.data, finished, onClose]);

  const key = game && `${game.name}|${game.cover}|${game.minutes}|${game.finishedAt}|${number}`;
  useEffect(() => {
    if (!game || !finished) return;
    let live = true;
    drawFinishedCard(game, number).then((c) => {
      if (!live) return;
      canvas.current = c;
      setPreview(c.toDataURL("image/png"));
    }, report);
    return () => {
      live = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  if (!game) return null;

  const copy = async () => {
    try {
      if (!canvas.current) return;
      const blob = await cardBlob(canvas.current);
      await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
      toast({ message: "Card copied to the clipboard" });
    } catch (err) {
      report(err);
    }
  };
  const save = async () => {
    if (!canvas.current) return;
    try {
      const path = await DataService.SaveImage(cardFileName(game), canvas.current.toDataURL("image/png").split(",")[1]);
      toast({ message: `Saved ${path}` });
    } catch (err) {
      if (errorText(err) !== "cancelled") report(err);
    }
  };

  return (
    <dialog ref={ref} className="modal finished-dialog" aria-labelledby="finished-dialog-h" onCancel={(e) => { e.preventDefault(); onClose(); }}>
      <div className="modal-head">
        <h2 id="finished-dialog-h">{justFinished ? `Finished ${game.name}!` : `${game.name}: finished card`}</h2>
        <button className="btn ghost small" aria-label="Close" onClick={onClose}>✕</button>
      </div>
      <div className="card-preview">{preview ? <img src={preview} alt={`Finished card for ${game.name}`} /> : <span className="faint">Drawing the card…</span>}</div>
      <div className="toolbar finished-line">
        <FinishedDate game={game} />
        <Rating value={game.rating} name={game.name} onChange={(v) => LibraryService.SetRating(game.appid, v).then(refresh, report)} />
      </div>
      <div className="modal-foot">
        <button className="btn ghost" onClick={onClose}>Close</button>
        <button className="btn" onClick={save} disabled={!preview}>Save PNG…</button>
        <button className="btn primary" onClick={copy} disabled={!preview}>Copy image</button>
      </div>
    </dialog>
  );
}
