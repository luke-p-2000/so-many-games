import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { errorText, LibraryService, list, Status, type State } from "./api";
import { Cover } from "./components/bits";
import { useGames, useRefresh } from "./data";
import { useShowFinishedCard } from "./finishedCardContext";
import { StatusIcon, statusLabel, statusStyle, type StatusValue } from "./statuses";

/** A status change toast also names the status, so it can show its icon, colour and the game's cover. */
type Toast = { message: string; undo?: () => void; error?: boolean; status?: StatusValue; what?: string; appid?: number };
const Ctx = createContext<(t: Toast) => void>(() => {});

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<Toast | null>(null);
  const timer = useRef<number>();
  const show = useCallback((t: Toast) => {
    setToast(t);
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => setToast(null), t.error ? 8000 : 6000);
  }, []);
  useEffect(() => () => window.clearTimeout(timer.current), []);
  return (
    <Ctx.Provider value={show}>
      {children}
      {toast && (
        <div className={`toast${toast.error ? " error" : ""}${toast.status ? " status" : ""}`} role="status" aria-live="polite"
          style={toast.status ? statusStyle(toast.status) : undefined}>
          {toast.status && toast.what ? <StatusToast toast={toast as Required<Pick<Toast, "status" | "what">> & Toast} /> : <span>{toast.message}</span>}
          {toast.undo && (
            <button className="btn small" onClick={() => { toast.undo?.(); setToast(null); }}>Undo</button>
          )}
          <button className="btn ghost small" aria-label="Dismiss" onClick={() => setToast(null)}>✕</button>
        </div>
      )}
    </Ctx.Provider>
  );
}

function StatusToast({ toast }: { toast: Toast & { status: StatusValue; what: string } }) {
  const games = useGames();
  const game = toast.appid ? games.data?.find((g) => g.appid === toast.appid) : undefined;
  return (
    <>
      <span className="icon" aria-hidden><StatusIcon status={toast.status} size={18} /></span>
      {game?.cover && <Cover game={game} className="thumb" />}
      <span><b>{toast.what}</b> → <span className="to">{statusLabel(toast.status)}</span></span>
    </>
  );
}

export const useToast = () => useContext(Ctx);

/** Shows a binding error as a toast. */
export function useReportError() {
  const toast = useToast();
  return useCallback((err: unknown) => toast({ message: errorText(err), error: true }), [toast]);
}

/** Applies a status at once with an undo toast (L3). Finishing one game opens its card (ADR 0016). */
export function useSetStatus() {
  const toast = useToast();
  const showCard = useShowFinishedCard();
  const refresh = useRefresh();
  const report = useReportError();
  return useCallback(
    async (appids: number[], status: StatusValue, what: string) => {
      try {
        const before: State[] = list(await LibraryService.SetStatus(appids, status));
        refresh();
        toast({
          message: `${what} → ${statusLabel(status)}`,
          status,
          what,
          appid: appids.length === 1 ? appids[0] : undefined,
          undo: () => LibraryService.Restore(before).then(refresh, report),
        });
        if (status === Status.Finished && appids.length === 1 && before[0]?.status !== Status.Finished) showCard(appids[0], true);
      } catch (err) {
        report(err);
      }
    },
    [toast, refresh, report, showCard],
  );
}
