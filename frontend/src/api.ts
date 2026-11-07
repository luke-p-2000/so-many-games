// One import point for the generated Wails bindings plus small shared helpers.
import type { Recommendations } from "../bindings/somanygames/internal/app/models";
import type { Ranked } from "../bindings/somanygames/internal/recommend/models";
export {
  DataService,
  LibraryService,
  MetaService,
  RecService,
  SettingsService,
  SetupService,
  StatsService,
  WheelService,
} from "../bindings/somanygames/internal/app";
export type { Banner, Connection, Recommendations, SyncState, Wheel as WheelDraw } from "../bindings/somanygames/internal/app/models";
export type { Game, State } from "../bindings/somanygames/internal/library/models";
export { Status } from "../bindings/somanygames/internal/library/models";
export type { Failure, Progress } from "../bindings/somanygames/internal/meta/models";
export type { BackUp, Profile, Ranked, Row } from "../bindings/somanygames/internal/recommend/models";
export type { Bar, Stats } from "../bindings/somanygames/internal/stats/models";
export type { Settings } from "../bindings/somanygames/internal/store/models";
export type { Summary } from "../bindings/somanygames/internal/transfer/models";
export type { Filters } from "../bindings/somanygames/internal/wheel/models";

/** Go nil slices arrive as null; read every list through this. */
export const list = <T,>(xs: T[] | null | undefined): T[] => xs ?? [];

/** Backend event names (internal/app/core.go). */
export const EVENTS = {
  libraryChanged: "library:changed",
  sync: "sync:state",
  fetchProgress: "fetch:progress",
} as const;

/** Error text from a rejected binding call. Wails wraps Go errors in an object with `message`. */
export function errorText(err: unknown): string {
  if (err && typeof err === "object" && "message" in err) return String((err as { message: unknown }).message);
  return String(err);
}

export function hours(minutes: number): string {
  const h = minutes / 60;
  if (h === 0) return "0 h";
  if (h < 10) return `${h.toFixed(1).replace(/\.0$/, "")} h`;
  return `${Math.round(h).toLocaleString()} h`;
}

export function daysSince(iso: string | null, now = Date.now()): number | null {
  if (!iso) return null;
  return Math.floor((now - Date.parse(iso)) / 86_400_000);
}

export function lastPlayedText(iso: string | null): string {
  const d = daysSince(iso);
  if (d === null) return "never played";
  if (d <= 0) return "played today";
  if (d === 1) return "played yesterday";
  return `last played ${d.toLocaleString()} days ago`;
}

export function reviewText(pct: number | null, count: number | null): string {
  if (pct === null || count === null) return "";
  return `${pct}% · ${count.toLocaleString()} reviews`;
}

export function shortReview(pct: number | null, count: number | null): string {
  if (pct === null || count === null) return "";
  const c = count >= 1e6 ? `${(count / 1e6).toFixed(1).replace(/\.0$/, "")}M` : count >= 1000 ? `${Math.round(count / 1000)}k` : String(count);
  return `${pct}% (${c})`;
}

/** A titled recommendation section: Tailored for you first, then one per profile tag (R5). */
export type RecSectionData = { key: string; title: string; note?: string; games: Ranked[] };

export function recSections(r: Recommendations | undefined): RecSectionData[] {
  if (!r) return [];
  const tailored = list(r.tailored).length ? [{ key: "tailored", title: "Tailored for you", note: "Games that share the most of your taste tags", games: list(r.tailored) }] : [];
  return [...tailored, ...list(r.rows).map((row) => ({ key: row.tag, title: `Because you play ${row.tag}`, games: list(row.games) }))];
}
