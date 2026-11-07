export type Route =
  | "home"
  | "library"
  | "recommendations"
  | "wheel"
  | "stats"
  | "settings"
  | "classify"
  | "pickbackup"
  | "hidden"
  | "dead";

/** Sidebar entries; Classify is added while unclassified games remain (L8). */
export const NAV: { route: Route; label: string }[] = [
  { route: "home", label: "Home" },
  { route: "library", label: "Library" },
  { route: "recommendations", label: "Recommendations" },
  { route: "wheel", label: "Wheel" },
  { route: "stats", label: "Stats" },
  { route: "settings", label: "Settings" },
];

/** Screens reached from inside another highlight that screen in the sidebar. */
export const PARENT: Partial<Record<Route, Route>> = { pickbackup: "library", hidden: "settings", dead: "settings" };
