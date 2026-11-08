import type { Route } from "../routes";

/** 18px stroke icons for the sidebar; colour comes from currentColor. */
const PATHS: Partial<Record<Route, string[]>> = {
  home: ["M4 11l8-7 8 7", "M6 10v10h12V10", "M10 20v-5h4v5"],
  classify: ["M4 6h10M4 12h10M4 18h6", "M16 16l2.5 2.5L22 14"],
  library: ["M4 4h6v6H4z", "M14 4h6v6h-6z", "M4 14h6v6H4z", "M14 14h6v6h-6z"],
  recommendations: ["M12 3l2.2 5.3L20 9l-4.4 3.8L17 19l-5-3-5 3 1.4-6.2L4 9l5.8-.7z"],
  wheel: ["M12 3.5a8.5 8.5 0 100 17 8.5 8.5 0 000-17z", "M12 3.5v17M3.5 12h17M6 6l12 12M18 6L6 18"],
  stats: ["M5 20V11M12 20V5M19 20v-7"],
  tiers: ["M4 6h16M4 12h11M4 18h6"],
  settings: ["M12 9a3 3 0 100 6 3 3 0 000-6z", "M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M5.6 18.4l2.1-2.1M16.3 7.7l2.1-2.1"],
};

export function NavIcon({ route }: { route: Route }) {
  const paths = PATHS[route];
  if (!paths) return null;
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      {paths.map((d) => <path key={d} d={d} />)}
    </svg>
  );
}
