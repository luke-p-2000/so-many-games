import { useEffect, useMemo, useState, type DragEvent } from "react";
import { byFinishedNewest, LibraryService, list, SettingsService, Status, type Game, type TierEntry } from "../api";
import { Cover } from "../components/bits";
import { ShareButton } from "../components/ShareCard";
import { useGames, useQuery, useRefresh } from "../data";
import { drawTierListCard } from "../shareCards";
import { useReportError } from "../toast";

const TIERS = ["S", "A", "B", "C", "D", "F"] as const;
type Tier = (typeof TIERS)[number];
type Target = { tier: Tier | null; index: number };

/**
 * T2: drag Finished games (and optionally Endless ones) from the unranked panel into tiers.
 * Every drop saves the whole list; placements of games no longer shown are kept.
 */
export function TierList() {
  const games = useGames();
  const settings = useQuery(() => SettingsService.Get());
  const saved = useQuery<TierEntry[]>(() => LibraryService.Tiers().then(list));
  const refresh = useRefresh();
  const report = useReportError();
  const [local, setLocal] = useState<TierEntry[] | null>(null);
  const [dragging, setDragging] = useState<number | null>(null);
  const [target, setTarget] = useState<Target | null>(null);
  const [search, setSearch] = useState("");

  // Unsaved drops show at once; a fresh load from Go (after any refresh) takes over again.
  useEffect(() => setLocal(null), [saved.data]);

  const includeEndless = settings.data?.tier_endless ?? false;
  const entries = local ?? saved.data ?? [];

  const eligible = useMemo(() => {
    const all = games.data ?? [];
    const endless = includeEndless ? all.filter((g) => g.status === Status.Endless && g.classified && !g.hidden).sort((a, b) => a.name.localeCompare(b.name)) : [];
    return [...byFinishedNewest(all), ...endless];
  }, [games.data, includeEndless]);
  const byId = useMemo(() => new Map(eligible.map((g) => [g.appid, g])), [eligible]);

  const rows = useMemo(() => {
    const out = new Map<Tier, Game[]>(TIERS.map((t) => [t, []]));
    for (const e of [...entries].sort((a, b) => a.position - b.position)) {
      const g = byId.get(e.appid);
      if (g) out.get(e.tier as Tier)?.push(g);
    }
    return out;
  }, [entries, byId]);
  const placed = new Set(entries.map((e) => e.appid));
  const rankedCount = [...rows.values()].reduce((a, gs) => a + gs.length, 0);
  const q = search.trim().toLowerCase();
  const unranked = eligible.filter((g) => !placed.has(g.appid) && (!q || g.name.toLowerCase().includes(q)));

  /** Moves a game to index among the shown cards of tier, or out of the list (tier null). */
  const move = (appid: number, to: Target) => {
    const rest = entries.filter((e) => e.appid !== appid);
    let next = rest;
    if (to.tier) {
      const tier = to.tier;
      const shown = (rows.get(tier) ?? []).filter((g) => g.appid !== appid);
      const before = shown[to.index]?.appid; // insert before this card, or at the end
      const inTier = rest.filter((e) => e.tier === tier).sort((a, b) => a.position - b.position).map((e) => e.appid);
      const at = before === undefined ? inTier.length : inTier.indexOf(before);
      inTier.splice(at, 0, appid);
      next = [...rest.filter((e) => e.tier !== tier), ...inTier.map((id, position) => ({ appid: id, tier, position }))];
    }
    setLocal(next);
    LibraryService.SetTiers(next).catch((err) => {
      report(err);
      setLocal(null);
      refresh();
    });
  };

  const drop = (e: DragEvent) => {
    e.preventDefault();
    const appid = Number(e.dataTransfer.getData("text/plain"));
    if (appid && target) move(appid, target);
    setTarget(null);
    setDragging(null);
  };

  /** Where in a row of wrapped cards the pointer would insert. */
  const indexAt = (e: DragEvent<HTMLElement>) => {
    const cards = [...e.currentTarget.querySelectorAll<HTMLElement>("[data-card]")];
    const i = cards.findIndex((c) => {
      const r = c.getBoundingClientRect();
      return e.clientY < r.bottom && (e.clientY < r.top || e.clientX < r.left + r.width / 2);
    });
    return i === -1 ? cards.length : i;
  };

  const card = (g: Game, drop?: "before") => (
    <div
      key={g.appid}
      data-card
      className={`tier-card${dragging === g.appid ? " dragging" : ""}${drop ? " drop-before" : ""}`}
      draggable
      title={g.name}
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", String(g.appid));
        e.dataTransfer.effectAllowed = "move";
        setDragging(g.appid);
      }}
      onDragEnd={() => {
        setDragging(null);
        setTarget(null);
      }}
    >
      <Cover game={g} />
      <span className="name">{g.name}</span>
    </div>
  );

  return (
    <div className="page tier-page">
      <div className="page-head">
        <h1>Tier list</h1>
        <ShareButton
          title="Tier list card"
          fileName="So Many Games tier list.png"
          tall
          disabled={rankedCount === 0}
          draw={() => drawTierListCard(TIERS.map((t) => [t, rows.get(t) ?? []]))}
          drawKey={TIERS.map((t) => (rows.get(t) ?? []).map((g) => `${g.appid}:${g.cover}`).join(",")).join("|")}
        />
      </div>
      <p className="subtle" style={{ marginTop: -8 }}>Drag games from Unranked into a tier, between tiers, or back to Unranked.</p>

      <div className="tier-layout">
        <div className="tier-board">
          {TIERS.map((t) => {
            const cards = rows.get(t) ?? [];
            const over = target?.tier === t;
            return (
              <section
                key={t}
                className={`tier-row${over ? " over" : ""}`}
                aria-label={`Tier ${t}`}
                onDragOver={(e) => {
                  e.preventDefault();
                  const index = indexAt(e);
                  if (target?.tier !== t || target.index !== index) setTarget({ tier: t, index });
                }}
                onDragLeave={(e) => !e.currentTarget.contains(e.relatedTarget as Node) && setTarget(null)}
                onDrop={drop}
              >
                <div className={`tier-label tier-${t}`}>{t}</div>
                <div className={`tier-cards${over && target.index >= cards.length ? " drop-end" : ""}`}>
                  {cards.map((g, i) => card(g, over && target.index === i ? "before" : undefined))}
                </div>
              </section>
            );
          })}
        </div>

        <aside
          className={`panel tier-inventory${target?.tier === null ? " over" : ""}`}
          aria-label="Unranked games"
          onDragOver={(e) => {
            e.preventDefault();
            if (target?.tier !== null) setTarget({ tier: null, index: 0 });
          }}
          onDragLeave={(e) => !e.currentTarget.contains(e.relatedTarget as Node) && setTarget(null)}
          onDrop={drop}
        >
          {/* Laid out inside an absolute box so the panel takes the tier board's height, not its own. */}
          <div className="tier-inventory-inner">
            <div className="tier-inventory-head">
              <h3>Unranked <span className="faint">· {unranked.length}</span></h3>
              <label className="check faint">
                <input type="checkbox" checked={includeEndless} onChange={(e) => LibraryService.SetTierEndless(e.target.checked).then(refresh, report)} />
                Include Endless
              </label>
            </div>
            <input className="input" type="search" placeholder="Search" aria-label="Search unranked games" value={search} onChange={(e) => setSearch(e.target.value)} />
            <div className="tier-inventory-cards">
              {unranked.map((g) => card(g))}
              {!games.loading && unranked.length === 0 && <p className="faint">{eligible.length ? "Every game is ranked." : "No Finished games yet."}</p>}
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
}
