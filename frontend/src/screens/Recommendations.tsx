import { useState } from "react";
import { list, RecService, recSections, type Recommendations as Recs } from "../api";
import { FetchBanner } from "../components/Banners";
import { Segmented } from "../components/bits";
import { RecSections } from "../components/RecSections";
import { useQuery } from "../data";
import type { Route } from "../routes";

type View = "all" | "recent";

export function Recommendations({ go }: { go: (r: Route) => void }) {
  const [view, setView] = useState<View>("all");
  const recs = useQuery<Recs>(() => RecService.Rows(view), [view]);

  const empty = recSections(recs.data).length === 0;
  const profile = list(recs.data?.profile.tags);
  return (
    <div className="page">
      <div className="page-head">
        <h1>Recommendations</h1>
        <Segmented label="Taste profile" value={view} onChange={setView} options={[{ value: "all", label: "All-time" }, { value: "recent", label: "Recent" }]} />
      </div>
      <FetchBanner />
      {profile.length > 0 && (
        <div className="toolbar">
          <span className="subtle">Your taste:</span>
          <div className="chips">
            {profile.map((t) => <span key={t.name} className={`chip${t.pinned ? "" : " accent"}`}>{t.name}{t.pinned ? " (pinned)" : ""}</span>)}
          </div>
          <button className="link" onClick={() => go("settings")}>Edit tag lists</button>
        </div>
      )}
      {recs.error && <p className="warn-text">{recs.error}</p>}
      {!recs.loading && empty && (
        <div className="panel empty">
          No recommendations yet. They need store data for your games, played hours to build a taste profile, and
          Unplayed, To play or lightly played Shelved and Endless games with at least the minimum number of reviews.
        </div>
      )}
      <RecSections data={recs.data} />
    </div>
  );
}
