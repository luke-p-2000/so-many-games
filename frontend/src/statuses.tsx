import { Status } from "./api";

export type StatusValue = Exclude<Status, Status.$zero>;

export const STATUSES: { value: StatusValue; label: string }[] = [
  { value: Status.Unplayed, label: "Unplayed" },
  { value: Status.ToPlay, label: "To play" },
  { value: Status.Playing, label: "Playing" },
  { value: Status.Shelved, label: "Shelved" },
  { value: Status.Finished, label: "Finished" },
  { value: Status.NotForMe, label: "Not for me" },
  { value: Status.Endless, label: "Endless" },
  { value: Status.Dead, label: "Dead" },
];

export const statusLabel = (s: Status) => STATUSES.find((x) => x.value === s)?.label ?? s;

/** 16px stroke icons; colour comes from currentColor. */
export function StatusIcon({ status, size = 16 }: { status: Status; size?: number }) {
  const common = { width: size, height: size, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 2, strokeLinecap: "round" as const, strokeLinejoin: "round" as const, "aria-hidden": true };
  switch (status) {
    case Status.Unplayed:
      return <svg {...common}><circle cx="12" cy="12" r="8" strokeDasharray="3 3.3" /></svg>;
    case Status.ToPlay:
      return <svg {...common}><path d="M7 4h10v16l-5-4-5 4z" /></svg>;
    case Status.Playing:
      return <svg {...common}><path d="M8 5l11 7-11 7z" /></svg>;
    case Status.Shelved:
      return <svg {...common}><path d="M9 5v14M15 5v14" /></svg>;
    case Status.Finished:
      return <svg {...common}><path d="M5 12.5l4.5 4.5L19 7.5" /></svg>;
    case Status.NotForMe:
      return <svg {...common}><circle cx="12" cy="12" r="8" /><path d="M6.5 17.5l11-11" /></svg>;
    case Status.Endless:
      return <svg {...common}><path d="M12 12c-2-2.7-3.6-4-5.5-4a4 4 0 000 8c1.9 0 3.5-1.3 5.5-4zm0 0c2 2.7 3.6 4 5.5 4a4 4 0 000-8c-1.9 0-3.5 1.3-5.5 4z" /></svg>;
    case Status.Dead:
      // headstone
      return <svg {...common}><path d="M6 20V10a6 6 0 0112 0v10" /><path d="M4 20h16M12 9v6M9.5 11.5h5" /></svg>;
    default:
      return null;
  }
}
