// Draws the shareable Stats and Tier list cards (docs/spec/ui-mocks/share-cards): games
// finished, the status breakdown and the tier list. They share the finished card's frame,
// logo and fonts, and are drawn at twice their size like it.
import { finishedDateText, Status, todayISO, type Game } from "./api";
import logoUrl from "./assets/logo.svg";
import { BODY, check, clip, DISPLAY, drawCover, GREEN, loadImage } from "./finishedCard";
import { STATUSES, type StatusValue } from "./statuses";

const W = 1200;
const SCALE = 2;
const GOLD = "#e8b04b";
const TEXT = "#ecedf0";
const MUTED = "#9aa1b0";
const FAINT = "#7d8494";
const LINE = "rgba(255,255,255,0.1)";

/** The status colours (styles.css --st-*), for drawing outside CSS. */
export const STATUS_HEX: Record<string, string> = {
  unplayed: "#8b93a5",
  to_play: "#5b9cf5",
  playing: "#e8b04b",
  shelved: "#a685e8",
  finished: "#5cc98a",
  not_for_me: "#e66a5c",
  endless: "#3fc1c0",
  dead: "#9c7b63",
};

/** The tier colours (styles.css .tier-*). */
const TIER_HEX: Record<string, string> = { S: "#e5707a", A: "#ec9a5c", B: "#e8c35a", C: "#8fcf6e", D: "#5fb3d9", F: "#9a8fd8" };

const rgba = (hex: string, a: number) => {
  const n = parseInt(hex.slice(1), 16);
  return `rgba(${n >> 16},${(n >> 8) & 255},${n & 255},${a})`;
};

const pctText = (n: number, total: number) => {
  const p = total ? (n / total) * 100 : 0;
  return `${p < 10 ? p.toFixed(1) : Math.round(p)}%`;
};

async function fonts() {
  await Promise.all([document.fonts.load(`700 40px ${DISPLAY}`), document.fonts.load(`600 15px ${BODY}`), document.fonts.load(`500 15px ${BODY}`)]);
}

/** A canvas of W × h at SCALE with the dark background, coloured glows and the inner frame. */
function frame(h: number, glows: [x: number, y: number, r: number, colour: string, alpha: number][]) {
  const canvas = document.createElement("canvas");
  canvas.width = W * SCALE;
  canvas.height = h * SCALE;
  const ctx = canvas.getContext("2d")!;
  ctx.scale(SCALE, SCALE);
  ctx.fillStyle = "#0c0e12";
  ctx.fillRect(0, 0, W, h);
  for (const [x, y, r, c, a] of glows) {
    const g = ctx.createRadialGradient(x, y, 0, x, y, r);
    g.addColorStop(0, rgba(c, a));
    g.addColorStop(1, rgba(c, 0));
    ctx.fillStyle = g;
    ctx.fillRect(0, 0, W, h);
  }
  ctx.strokeStyle = "rgba(255,255,255,0.08)";
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(22.5, 22.5, W - 45, h - 45, 18);
  ctx.stroke();
  return { canvas, ctx };
}

/** The logo top left and a pill badge top right with an icon drawn by icon(cx, cy). */
async function header(ctx: CanvasRenderingContext2D, label: string, colour: string, icon: (cx: number, cy: number) => void) {
  const logo = await loadImage(logoUrl);
  if (logo) ctx.drawImage(logo, 60, 50, (52 * 411) / 96, 52);
  ctx.font = `700 18px ${DISPLAY}`;
  ctx.letterSpacing = "2px";
  const bw = ctx.measureText(label).width + 66;
  const bx = W - 60 - bw;
  ctx.fillStyle = rgba(colour, 0.14);
  ctx.strokeStyle = rgba(colour, 0.5);
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(bx, 56, bw, 40, 20);
  ctx.fill();
  ctx.stroke();
  ctx.strokeStyle = colour;
  icon(bx + 26, 76);
  ctx.fillStyle = colour;
  ctx.textBaseline = "middle";
  ctx.fillText(label, bx + 44, 77);
  ctx.letterSpacing = "0px";
}

/** Horizontal lines of decreasing length, the badge icon for lists. */
function linesIcon(ctx: CanvasRenderingContext2D, cx: number, cy: number, lengths: number[]) {
  ctx.lineWidth = 2.4;
  ctx.lineCap = "round";
  ctx.beginPath();
  lengths.forEach((l, i) => {
    const y = cy - 6 + i * 6;
    ctx.moveTo(cx - 8, y);
    ctx.lineTo(cx - 8 + l, y);
  });
  ctx.stroke();
}

function kicker(ctx: CanvasRenderingContext2D, text: string, x: number, y: number) {
  ctx.textBaseline = "middle";
  ctx.font = `600 15px ${BODY}`;
  ctx.letterSpacing = "1.2px";
  ctx.fillStyle = FAINT;
  ctx.fillText(text.toUpperCase(), x, y);
  ctx.letterSpacing = "0px";
}

/** Cover art cropped into a rounded box, or the game's initials on a gradient. */
function coverBox(ctx: CanvasRenderingContext2D, img: HTMLImageElement | null, name: string, x: number, y: number, w: number, h: number, r: number) {
  ctx.save();
  ctx.beginPath();
  ctx.roundRect(x, y, w, h, r);
  ctx.clip();
  if (img) drawCover(ctx, img, x, y, w, h);
  else {
    const g = ctx.createLinearGradient(x, y, x + w, y + h);
    g.addColorStop(0, "#2b3140");
    g.addColorStop(1, "#1a1d24");
    ctx.fillStyle = g;
    ctx.fillRect(x, y, w, h);
    ctx.fillStyle = "#9aa1b0";
    ctx.font = `700 ${Math.round(h * 0.3)}px ${DISPLAY}`;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]).join("").toUpperCase(), x + w / 2, y + h / 2 + 1);
    ctx.textAlign = "left";
  }
  ctx.restore();
  ctx.strokeStyle = "rgba(255,255,255,0.1)";
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(x + 0.5, y + 0.5, w - 1, h - 1, r);
  ctx.stroke();
}

const coverOf = (g: Game) => (g.cover ? loadImage(`/covers/${g.cover}`) : Promise.resolve(null));

// ---- 1. Games finished ----

export type FinishedProgress = { finished: number; total: number; latest: Game | null; thisYear: number; year: number };

export async function drawFinishedProgressCard(p: FinishedProgress): Promise<HTMLCanvasElement> {
  await fonts();
  const H = 630;
  const { canvas, ctx } = frame(H, [[1110, 70, 340, GREEN, 0.18], [60, 690, 300, GOLD, 0.07]]);
  await header(ctx, "GAMES FINISHED", GREEN, (cx, cy) => check(ctx, cx, cy, 20, 3));

  // The count and its share of the library.
  ctx.textBaseline = "alphabetic";
  ctx.save();
  ctx.shadowColor = "rgba(92,201,138,0.4)";
  ctx.shadowBlur = 40;
  ctx.font = `700 148px ${DISPLAY}`;
  ctx.letterSpacing = "-4px";
  ctx.fillStyle = GREEN;
  const count = p.finished.toLocaleString();
  ctx.fillText(count, 56, 302);
  const cw = ctx.measureText(count).width;
  ctx.restore();
  ctx.letterSpacing = "0px";
  ctx.font = `700 40px ${DISPLAY}`;
  ctx.fillStyle = MUTED;
  ctx.fillText(`/ ${p.total.toLocaleString()}`, 56 + cw + 18, 262);
  ctx.textBaseline = "alphabetic";
  ctx.font = `600 15px ${BODY}`;
  ctx.letterSpacing = "1.2px";
  ctx.fillStyle = FAINT;
  ctx.fillText("GAMES IN MY LIBRARY", 56 + cw + 18, 296);
  ctx.letterSpacing = "0px";

  ctx.textAlign = "right";
  ctx.font = `700 64px ${DISPLAY}`;
  ctx.fillStyle = TEXT;
  ctx.fillText(pctText(p.finished, p.total), W - 60, 266);
  ctx.font = `500 18px ${BODY}`;
  ctx.fillStyle = MUTED;
  ctx.fillText("of my library finished", W - 60, 296);
  ctx.textAlign = "left";

  // The progress bar: a striped, glowing fill with quarter ticks under the track.
  const bx = 60, by = 346, bw = W - 120, bh = 30;
  ctx.fillStyle = "#1d2129";
  ctx.beginPath();
  ctx.roundRect(bx, by, bw, bh, bh / 2);
  ctx.fill();
  ctx.strokeStyle = "rgba(255,255,255,0.06)";
  ctx.stroke();
  const fw = p.finished && p.total ? Math.max(bh, (bw * p.finished) / p.total) : 0;
  if (fw) {
    const g = ctx.createLinearGradient(bx, 0, bx + bw, 0);
    g.addColorStop(0, "#3e9f68");
    g.addColorStop(0.7, GREEN);
    g.addColorStop(1, "#8be3b0");
    ctx.save();
    ctx.shadowColor = "rgba(92,201,138,0.5)";
    ctx.shadowBlur = 28;
    ctx.fillStyle = g;
    ctx.beginPath();
    ctx.roundRect(bx, by, fw, bh, bh / 2);
    ctx.fill();
    ctx.restore();
    ctx.save();
    ctx.beginPath();
    ctx.roundRect(bx, by, fw, bh, bh / 2);
    ctx.clip();
    ctx.strokeStyle = "rgba(255,255,255,0.1)";
    ctx.lineWidth = 12;
    ctx.beginPath();
    for (let x = bx - bh; x < bx + fw + bh; x += 28) {
      ctx.moveTo(x, by + bh + 2);
      ctx.lineTo(x + bh * 0.47, by - 2);
    }
    ctx.stroke();
    ctx.restore();
  }
  ctx.font = `600 13px ${BODY}`;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  for (const q of [25, 50, 75]) {
    const x = Math.round(bx + (bw * q) / 100) + 0.5;
    ctx.strokeStyle = "rgba(255,255,255,0.18)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(x, by + bh + 6);
    ctx.lineTo(x, by + bh + 12);
    ctx.stroke();
    ctx.fillStyle = FAINT;
    ctx.fillText(`${q}%`, x, by + bh + 24);
  }
  ctx.textAlign = "left";

  // Footer: the latest finish, finishes this year and how many are left.
  ctx.strokeStyle = LINE;
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(60, 491.5);
  ctx.lineTo(W - 60, 491.5);
  ctx.stroke();
  const cols = [60, 510, 825];
  for (const x of cols.slice(1)) {
    ctx.beginPath();
    ctx.moveTo(x + 0.5, 514);
    ctx.lineTo(x + 0.5, 584);
    ctx.stroke();
  }
  kicker(ctx, "Latest finish", cols[0], 525);
  if (p.latest) {
    coverBox(ctx, await coverOf(p.latest), p.latest.name, cols[0], 542, 96, 45, 6);
    ctx.textBaseline = "middle";
    ctx.font = `700 24px ${DISPLAY}`;
    ctx.fillStyle = TEXT;
    ctx.fillText(clip(ctx, p.latest.name, cols[1] - 32 - (cols[0] + 110)), cols[0] + 110, 555);
    ctx.font = `500 15px ${BODY}`;
    ctx.fillStyle = MUTED;
    ctx.fillText(finishedDateText(p.latest.finishedAt), cols[0] + 110, 579);
  } else {
    ctx.font = `700 32px ${DISPLAY}`;
    ctx.fillStyle = TEXT;
    ctx.fillText("—", cols[0], 562);
  }
  const stat = (label: string, value: string, colour: string, x: number) => {
    kicker(ctx, label, x, 525);
    ctx.font = `700 32px ${DISPLAY}`;
    ctx.fillStyle = colour;
    ctx.fillText(value, x, 562);
  };
  stat(`Finished in ${p.year}`, p.thisYear.toLocaleString(), GREEN, cols[1] + 32);
  stat("Still to go", (p.total - p.finished).toLocaleString(), TEXT, cols[2] + 32);
  return canvas;
}

// ---- 2. Status breakdown ----

/** Draws a status icon (24-unit paths from StatusIcon) centred on cx, cy. */
function statusIcon(ctx: CanvasRenderingContext2D, status: StatusValue, cx: number, cy: number, size: number) {
  const k = size / 24;
  ctx.save();
  ctx.translate(cx - 12 * k, cy - 12 * k);
  ctx.scale(k, k);
  ctx.lineWidth = 2;
  ctx.lineCap = "round";
  ctx.lineJoin = "round";
  const circle = () => {
    ctx.beginPath();
    ctx.arc(12, 12, 8, 0, Math.PI * 2);
    ctx.stroke();
  };
  const paths: Record<string, string[]> = {
    to_play: ["M7 4h10v16l-5-4-5 4z"],
    playing: ["M8 5l11 7-11 7z"],
    shelved: ["M9 5v14M15 5v14"],
    finished: ["M5 12.5l4.5 4.5L19 7.5"],
    not_for_me: ["M6.5 17.5l11-11"],
    endless: ["M12 12c-2-2.7-3.6-4-5.5-4a4 4 0 000 8c1.9 0 3.5-1.3 5.5-4zm0 0c2 2.7 3.6 4 5.5 4a4 4 0 000-8c-1.9 0-3.5 1.3-5.5 4z"],
    dead: ["M6 20V10a6 6 0 0112 0v10", "M4 20h16M12 9v6M9.5 11.5h5"],
  };
  if (status === Status.Unplayed) {
    ctx.setLineDash([3, 3.3]);
    circle();
  }
  if (status === Status.NotForMe) circle();
  for (const d of paths[status] ?? []) ctx.stroke(new Path2D(d));
  ctx.restore();
}

export type Breakdown = { counts: Map<string, number>; backlogSize: number; backlogHours: number };

export async function drawBreakdownCard(b: Breakdown): Promise<HTMLCanvasElement> {
  await fonts();
  const H = 630;
  const total = [...b.counts.values()].reduce((a, n) => a + n, 0);
  const { canvas, ctx } = frame(H, [[1110, 50, 400, "#5b9cf5", 0.12], [60, 700, 340, "#a685e8", 0.1]]);
  await header(ctx, "MY LIBRARY", GOLD, (cx, cy) => linesIcon(ctx, cx, cy, [16, 10, 6]));

  ctx.textBaseline = "alphabetic";
  ctx.font = `700 46px ${DISPLAY}`;
  ctx.letterSpacing = "-1px";
  ctx.fillStyle = TEXT;
  ctx.fillText("Status breakdown", 60, 196);
  ctx.letterSpacing = "0px";
  ctx.textAlign = "right";
  ctx.font = `500 20px ${BODY}`;
  ctx.fillStyle = MUTED;
  ctx.fillText(" games", W - 60, 194);
  const gw = ctx.measureText(" games").width;
  ctx.font = `700 28px ${DISPLAY}`;
  ctx.fillStyle = TEXT;
  ctx.fillText(total.toLocaleString(), W - 60 - gw, 194);
  ctx.textAlign = "left";

  // The stacked bar: one rounded segment per status with games, small ones kept visible.
  const shown = STATUSES.filter((s) => (b.counts.get(s.value) ?? 0) > 0);
  const gap = 4, sy = 222, sh = 34;
  const avail = W - 120 - gap * Math.max(0, shown.length - 1);
  const raw = shown.map((s) => Math.max(8, (avail * (b.counts.get(s.value) ?? 0)) / total));
  const k = avail / raw.reduce((a, n) => a + n, 0);
  let x = 60;
  shown.forEach((s, i) => {
    const w = raw[i] * k;
    const r = sh / 2;
    const first = i === 0, last = i === shown.length - 1;
    ctx.fillStyle = STATUS_HEX[s.value];
    ctx.beginPath();
    ctx.roundRect(x, sy, w, sh, [first ? r : 6, last ? r : 6, last ? r : 6, first ? r : 6]);
    ctx.fill();
    x += w + gap;
  });
  if (!shown.length) {
    ctx.fillStyle = "#1d2129";
    ctx.beginPath();
    ctx.roundRect(60, sy, W - 120, sh, sh / 2);
    ctx.fill();
  }

  // A tile per status.
  const tw = (W - 120 - 3 * 14) / 4, th = 82;
  STATUSES.forEach((s, i) => {
    const c = STATUS_HEX[s.value];
    const n = b.counts.get(s.value) ?? 0;
    const tx = 60 + (i % 4) * (tw + 14), ty = 288 + Math.floor(i / 4) * (th + 14);
    ctx.save();
    if (!n) ctx.globalAlpha = 0.45;
    ctx.beginPath();
    ctx.roundRect(tx, ty, tw, th, 14);
    ctx.fillStyle = "#15181e";
    ctx.fill();
    const g = ctx.createLinearGradient(tx, ty, tx + tw * 0.7, ty + th);
    g.addColorStop(0, rgba(c, 0.15));
    g.addColorStop(1, rgba(c, 0));
    ctx.fillStyle = g;
    ctx.fill();
    ctx.strokeStyle = rgba(c, 0.28);
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.roundRect(tx + 0.5, ty + 0.5, tw - 1, th - 1, 14);
    ctx.stroke();
    ctx.fillStyle = c;
    ctx.beginPath();
    ctx.roundRect(tx, ty + 14, 3, th - 28, [0, 3, 3, 0]);
    ctx.fill();
    ctx.fillStyle = rgba(c, 0.16);
    ctx.beginPath();
    ctx.arc(tx + 40, ty + th / 2, 22, 0, Math.PI * 2);
    ctx.fill();
    ctx.strokeStyle = c;
    statusIcon(ctx, s.value, tx + 40, ty + th / 2, 22);
    ctx.textBaseline = "middle";
    ctx.font = `600 15px ${BODY}`;
    ctx.fillStyle = MUTED;
    ctx.fillText(s.label, tx + 76, ty + 26);
    ctx.font = `700 32px ${DISPLAY}`;
    ctx.fillStyle = TEXT;
    const ns = n.toLocaleString();
    ctx.fillText(ns, tx + 76, ty + 55);
    const nw = ctx.measureText(ns).width;
    ctx.font = `600 15px ${BODY}`;
    ctx.fillStyle = c;
    ctx.fillText(pctText(n, total), tx + 76 + nw + 8, ty + 58);
    ctx.restore();
  });

  // Footer: the backlog line from the Stats screen.
  ctx.strokeStyle = LINE;
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(60, 511.5);
  ctx.lineTo(W - 60, 511.5);
  ctx.stroke();
  const runs = (parts: [string, boolean][], x: number, align: "left" | "right") => {
    const fontOf = (bold: boolean) => (bold ? `700 24px ${DISPLAY}` : `500 18px ${BODY}`);
    const widths = parts.map(([t, bold]) => { ctx.font = fontOf(bold); return ctx.measureText(t).width; });
    let px = align === "left" ? x : x - widths.reduce((a, n) => a + n, 0);
    ctx.textBaseline = "middle";
    parts.forEach(([t, bold], i) => {
      ctx.font = fontOf(bold);
      ctx.fillStyle = bold ? TEXT : MUTED;
      ctx.fillText(t, px, 552);
      px += widths[i];
    });
  };
  runs([["Backlog ", false], [b.backlogSize.toLocaleString(), true], [" games", false]], 60, "left");
  runs([[`${Math.round(b.backlogHours).toLocaleString()} h`, true], [" already played in the backlog", false]], W - 60, "right");
  return canvas;
}

// ---- 3. Tier list ----

export type TierRows = [tier: string, games: Game[]][];

export async function drawTierListCard(rows: TierRows): Promise<HTMLCanvasElement> {
  await fonts();
  const LABEL = 88, CW = 152, CH = 98, GAP = 10, PAD = 10, MIN_ROW = 104;
  const perLine = Math.max(1, Math.floor((W - 120 - LABEL - PAD * 2 + GAP) / (CW + GAP)));
  const heights = rows.map(([, gs]) => {
    const lines = Math.max(1, Math.ceil(gs.length / perLine));
    return Math.max(MIN_ROW, PAD * 2 + lines * CH + (lines - 1) * GAP);
  });
  const top = 136;
  const boardEnd = top + heights.reduce((a, h) => a + h, 0) + 8 * (rows.length - 1);
  const H = boardEnd + 26 + 20 + 50;

  const all = rows.flatMap(([, gs]) => gs);
  const covers = new Map(await Promise.all(all.map(async (g) => [g.appid, await coverOf(g)] as const)));
  const { canvas, ctx } = frame(H, [[1140, 40, 420, TIER_HEX.S, 0.1], [60, H + 40, 380, TIER_HEX.D, 0.07]]);
  await header(ctx, "TIER LIST", TIER_HEX.S, (cx, cy) => linesIcon(ctx, cx, cy, [16, 12, 7]));

  let y = top;
  rows.forEach(([tier, gs], i) => {
    const h = heights[i];
    ctx.save();
    ctx.beginPath();
    ctx.roundRect(60, y, W - 120, h, 12);
    ctx.clip();
    ctx.fillStyle = "#171a21";
    ctx.fillRect(60, y, W - 120, h);
    ctx.fillStyle = TIER_HEX[tier] ?? "#888";
    ctx.fillRect(60, y, LABEL, h);
    ctx.restore();
    ctx.strokeStyle = "rgba(255,255,255,0.08)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.roundRect(60.5, y + 0.5, W - 121, h - 1, 12);
    ctx.stroke();
    ctx.fillStyle = "#14161b";
    ctx.font = `700 40px ${DISPLAY}`;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(tier, 60 + LABEL / 2, y + h / 2 + 2);
    ctx.textAlign = "left";
    if (!gs.length) {
      ctx.fillStyle = "#5b6170";
      ctx.font = `500 15px ${BODY}`;
      ctx.fillText("—", 60 + LABEL + PAD + 6, y + h / 2);
    }
    gs.forEach((g, j) => {
      const cx = 60 + LABEL + PAD + (j % perLine) * (CW + GAP);
      const cy = y + PAD + Math.floor(j / perLine) * (CH + GAP);
      ctx.fillStyle = "#20242c";
      ctx.beginPath();
      ctx.roundRect(cx, cy, CW, CH, 8);
      ctx.fill();
      ctx.strokeStyle = "rgba(255,255,255,0.08)";
      ctx.beginPath();
      ctx.roundRect(cx + 0.5, cy + 0.5, CW - 1, CH - 1, 8);
      ctx.stroke();
      coverBox(ctx, covers.get(g.appid) ?? null, g.name, cx + 6, cy + 6, CW - 12, 65, 5);
      ctx.font = `600 12px ${BODY}`;
      ctx.fillStyle = TEXT;
      ctx.textBaseline = "middle";
      ctx.fillText(clip(ctx, g.name, CW - 12), cx + 6, cy + 6 + 65 + 5 + 8);
    });
    y += h + 8;
  });

  // Footer: how many are ranked and today's date.
  const fy = boardEnd + 26 + 10;
  ctx.textBaseline = "middle";
  ctx.font = `700 16px ${DISPLAY}`;
  ctx.fillStyle = TEXT;
  const n = all.length.toLocaleString();
  ctx.fillText(n, 60, fy);
  const nw = ctx.measureText(n).width;
  ctx.font = `500 16px ${BODY}`;
  ctx.fillStyle = MUTED;
  ctx.fillText(` ${all.length === 1 ? "game" : "games"} ranked`, 60 + nw, fy);
  ctx.textAlign = "right";
  ctx.fillText(finishedDateText(todayISO()), W - 60, fy);
  ctx.textAlign = "left";
  return canvas;
}
