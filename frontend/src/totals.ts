import { list, StatsService, Status, type Stats } from "./api";
import { useQuery } from "./data";

/** Library totals for the sidebar card and Home's glance strip (classified, visible games). */
export function useTotals() {
  const s = useQuery<Stats>(() => StatsService.Stats()).data;
  if (!s) return null;
  const counts = new Map(list(s.statusCounts).map((b) => [b.label, b.value]));
  const total = [...counts.values()].reduce((a, b) => a + b, 0);
  const finished = counts.get(Status.Finished) ?? 0;
  return { total, finished, playing: counts.get(Status.Playing) ?? 0, backlog: s.backlogSize, finishedPct: total ? (finished / total) * 100 : 0 };
}

export const pctText = (p: number) => `${p < 10 ? p.toFixed(1) : Math.round(p)}%`;
