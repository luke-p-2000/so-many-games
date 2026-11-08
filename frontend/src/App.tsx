import { useState } from "react";
import { SetupService } from "./api";
import logo from "./assets/logo.svg";
import { NavIcon } from "./components/NavIcon";
import { useGames, useQuery } from "./data";
import { useTotals } from "./totals";
import { NAV, PARENT, type Route } from "./routes";
import { Classify } from "./screens/Classify";
import { Dead } from "./screens/Dead";
import { Hidden } from "./screens/Hidden";
import { Home } from "./screens/Home";
import { Library } from "./screens/Library";
import { Onboarding } from "./screens/Onboarding";
import { PickBackUp } from "./screens/PickBackUp";
import { RecentlyFinished } from "./screens/RecentlyFinished";
import { Recommendations } from "./screens/Recommendations";
import { Settings } from "./screens/Settings";
import { Stats } from "./screens/Stats";
import { TierList } from "./screens/TierList";
import { Wheel } from "./screens/Wheel";

export default function App() {
  const conn = useQuery(() => SetupService.Status());
  const [route, setRoute] = useState<Route>("home");
  const [justConnected, setJustConnected] = useState(false);

  if (conn.loading && !conn.data) return null;
  const connected = conn.data?.connected || justConnected;
  return (
    <div className="shell">
      <Sidebar route={route} go={setRoute} connected={connected} />
      <main className="content">
        {connected ? <Screen route={route} go={setRoute} /> : <Onboarding onConnected={() => { setJustConnected(true); setRoute("home"); }} />}
      </main>
      <Version />
    </div>
  );
}

function Version() {
  const v = useQuery(() => SetupService.Version()).data;
  if (!v) return null;
  return <div className="app-version">{v === "dev" ? "dev build" : `v${v}`}</div>;
}

function Screen({ route, go }: { route: Route; go: (r: Route) => void }) {
  switch (route) {
    case "home": return <Home go={go} />;
    case "library": return <Library />;
    case "recommendations": return <Recommendations go={go} />;
    case "wheel": return <Wheel />;
    case "stats": return <Stats />;
    case "settings": return <Settings go={go} />;
    case "classify": return <Classify go={go} />;
    case "pickbackup": return <PickBackUp />;
    case "finished": return <RecentlyFinished />;
    case "tiers": return <TierList />;
    case "hidden": return <Hidden />;
    case "dead": return <Dead />;
  }
}

function Sidebar({ route, go, connected }: { route: Route; go: (r: Route) => void; connected: boolean }) {
  const games = useGames();
  const unclassified = (games.data ?? []).filter((g) => !g.classified && !g.hidden).length;
  const items = [...NAV];
  if (unclassified > 0) items.splice(1, 0, { route: "classify", label: "Classify" });
  const active = PARENT[route] ?? route;
  return (
    <nav className="sidebar" aria-label="Main">
      <img className="brand" src={logo} alt="So Many Games" />
      {connected && items.map((item) => (
        <button key={item.route} className="nav-item" aria-current={active === item.route ? "page" : undefined} onClick={() => go(item.route)}>
          <NavIcon route={item.route} />
          <span className="nav-label">{item.label}</span>
          {item.route === "classify" && <span className="nav-badge">{unclassified}</span>}
        </button>
      ))}
      <div className="sidebar-foot">Not affiliated with Valve.</div>
      {connected && <SidebarTotals go={go} />}
    </nav>
  );
}

/** Finished and backlog at a glance, under the footer; opens Stats. */
function SidebarTotals({ go }: { go: (r: Route) => void }) {
  const t = useTotals();
  if (!t) return null;
  return (
    <button className="side-totals" onClick={() => go("stats")} title="Open Stats">
      <span className="row"><span>Finished</span><b>{t.finished} <small>/ {t.total}</small></b></span>
      <span className="meter" aria-hidden><span style={{ width: `${t.finishedPct}%` }} /></span>
      <span className="row"><span>Backlog</span><b>{t.backlog}</b></span>
    </button>
  );
}
