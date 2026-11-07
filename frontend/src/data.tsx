// Server state: a version counter that every query depends on. Mutations and backend
// events bump it, so all screens refetch from Go (design: "small query cache").
import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { Events } from "@wailsio/runtime";
import { EVENTS, errorText, LibraryService, list, type Game } from "./api";

type DataCtx = { version: number; refresh: () => void };
const Ctx = createContext<DataCtx>({ version: 0, refresh: () => {} });

export function DataProvider({ children }: { children: ReactNode }) {
  const [version, setVersion] = useState(0);
  const refresh = useCallback(() => setVersion((v) => v + 1), []);
  useEffect(() => Events.On(EVENTS.libraryChanged, refresh), [refresh]);
  return <Ctx.Provider value={{ version, refresh }}>{children}</Ctx.Provider>;
}

export const useRefresh = () => useContext(Ctx).refresh;

export type Query<T> = { data: T | undefined; error: string | null; loading: boolean };

/** Runs load on mount, on every refresh and when deps change; keeps the last data while reloading. */
export function useQuery<T>(load: () => Promise<T>, deps: unknown[] = []): Query<T> {
  const { version } = useContext(Ctx);
  const [state, setState] = useState<Query<T>>({ data: undefined, error: null, loading: true });
  const loadRef = useRef(load);
  loadRef.current = load;
  useEffect(() => {
    let live = true;
    setState((s) => ({ ...s, loading: true }));
    loadRef.current().then(
      (data) => live && setState({ data, error: null, loading: false }),
      (err) => live && setState((s) => ({ data: s.data, error: errorText(err), loading: false })),
    );
    return () => {
      live = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [version, ...deps]);
  return state;
}

export const useGames = () => useQuery<Game[]>(() => LibraryService.Games().then(list));

/** Subscribes to a backend event for the component's lifetime. */
export function useEvent<T>(name: string, handler: (data: T) => void) {
  const ref = useRef(handler);
  ref.current = handler;
  useEffect(() => Events.On(name, (ev: { data: T }) => ref.current(ev.data)), [name]);
}
