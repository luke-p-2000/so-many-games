import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { errorText, LibraryService, list, type State } from "./api";
import { useRefresh } from "./data";
import { statusLabel, type StatusValue } from "./statuses";

type Toast = { message: string; undo?: () => void; error?: boolean };
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
        <div className={`toast${toast.error ? " error" : ""}`} role="status" aria-live="polite">
          <span>{toast.message}</span>
          {toast.undo && (
            <button className="btn small" onClick={() => { toast.undo?.(); setToast(null); }}>Undo</button>
          )}
          <button className="btn ghost small" aria-label="Dismiss" onClick={() => setToast(null)}>✕</button>
        </div>
      )}
    </Ctx.Provider>
  );
}

export const useToast = () => useContext(Ctx);

/** Shows a binding error as a toast. */
export function useReportError() {
  const toast = useToast();
  return useCallback((err: unknown) => toast({ message: errorText(err), error: true }), [toast]);
}

/** Applies a status at once with an undo toast (L3). */
export function useSetStatus() {
  const toast = useToast();
  const refresh = useRefresh();
  const report = useReportError();
  return useCallback(
    async (appids: number[], status: StatusValue, what: string) => {
      try {
        const before: State[] = list(await LibraryService.SetStatus(appids, status));
        refresh();
        toast({
          message: `${what} → ${statusLabel(status)}`,
          undo: () => LibraryService.Restore(before).then(refresh, report),
        });
      } catch (err) {
        report(err);
      }
    },
    [toast, refresh, report],
  );
}
