// Draws the shareable finished-game card (ADR 0016, docs/spec/ui-mocks/FinishedCard.png)
// on a canvas: cover art, a finished checkmark, the date, hours played and "#N finished".
import { finishedDateText, hours, type Game } from "./api";
import logoUrl from "./assets/logo.svg";

const W = 1200;
const H = 630;
const SCALE = 2; // 2400 × 1260 pixels
export const GREEN = "#5cc98a";
export const DISPLAY = '"Space Grotesk", "Inter", sans-serif';
export const BODY = '"Inter", system-ui, sans-serif';

export function loadImage(src: string): Promise<HTMLImageElement | null> {
  return new Promise((resolve) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = src;
  });
}

/** Draws img to fill the box, cropping the overflow like CSS object-fit: cover. */
export function drawCover(ctx: CanvasRenderingContext2D, img: HTMLImageElement, x: number, y: number, w: number, h: number) {
  const scale = Math.max(w / img.width, h / img.height);
  const sw = w / scale;
  const sh = h / scale;
  ctx.drawImage(img, (img.width - sw) / 2, (img.height - sh) / 2, sw, sh, x, y, w, h);
}

export function check(ctx: CanvasRenderingContext2D, cx: number, cy: number, size: number, width: number) {
  // The Finished status icon's path (24-unit box), centred on cx, cy.
  const k = size / 24;
  ctx.save();
  ctx.translate(cx - 12 * k, cy - 12 * k);
  ctx.scale(k, k);
  ctx.beginPath();
  ctx.moveTo(5, 12.5);
  ctx.lineTo(9.5, 17);
  ctx.lineTo(19, 7.5);
  ctx.lineWidth = width / k;
  ctx.lineCap = "round";
  ctx.lineJoin = "round";
  ctx.stroke();
  ctx.restore();
}

/** Greedy word wrap to width at the current font. */
function wrap(ctx: CanvasRenderingContext2D, words: string[], width: number): string[] {
  const lines: string[] = [];
  let line = "";
  for (const w of words) {
    const next = line ? `${line} ${w}` : w;
    if (line && ctx.measureText(next).width > width) {
      lines.push(line);
      line = w;
    } else line = next;
  }
  return line ? [...lines, line] : lines;
}

/** Shortens text with an ellipsis until it fits width. */
export function clip(ctx: CanvasRenderingContext2D, text: string, width: number): string {
  if (ctx.measureText(text).width <= width) return text;
  while (text.length > 1 && ctx.measureText(`${text}…`).width > width) text = text.slice(0, -1);
  return `${text.trimEnd()}…`;
}

/** Splits the title into at most two lines that fit width, shrinking the font if needed. */
export function fitTitle(ctx: CanvasRenderingContext2D, title: string, width: number): { size: number; lines: string[] } {
  const words = title.split(/\s+/).filter(Boolean);
  for (let size = 58; size >= 34; size -= 4) {
    ctx.font = `700 ${size}px ${DISPLAY}`;
    const lines = wrap(ctx, words, width);
    if (lines.length <= 2 && lines.every((l) => ctx.measureText(l).width <= width)) return { size, lines };
  }
  // Too long even at the smallest size: two lines, the second ending in an ellipsis.
  const lines = wrap(ctx, words, width);
  return { size: 34, lines: [clip(ctx, lines[0], width), ...(lines.length > 1 ? [clip(ctx, lines.slice(1).join(" "), width)] : [])] };
}

/** Renders the card for a Finished game; number is its place in finish order. */
export async function drawFinishedCard(game: Game, number: number | undefined): Promise<HTMLCanvasElement> {
  await Promise.all([document.fonts.load(`700 58px ${DISPLAY}`), document.fonts.load(`600 15px ${BODY}`)]);
  const [cover, logo] = await Promise.all([game.cover ? loadImage(`/covers/${game.cover}`) : Promise.resolve(null), loadImage(logoUrl)]);

  const canvas = document.createElement("canvas");
  canvas.width = W * SCALE;
  canvas.height = H * SCALE;
  const ctx = canvas.getContext("2d")!;
  ctx.scale(SCALE, SCALE);

  // Background: the cover blurred behind a dark wash, with a green glow on the right.
  ctx.fillStyle = "#0c0e12";
  ctx.fillRect(0, 0, W, H);
  if (cover) {
    ctx.save();
    ctx.filter = "blur(38px) saturate(1.2)";
    ctx.globalAlpha = 0.35;
    drawCover(ctx, cover, -60, -60, W + 120, H + 120);
    ctx.restore();
  }
  const glow = ctx.createRadialGradient(984, 265, 0, 984, 265, 520);
  glow.addColorStop(0, "rgba(92,201,138,0.16)");
  glow.addColorStop(1, "rgba(92,201,138,0)");
  ctx.fillStyle = glow;
  ctx.fillRect(0, 0, W, H);
  const wash = ctx.createLinearGradient(0, 0, 0, H);
  wash.addColorStop(0, "rgba(12,14,18,0.55)");
  wash.addColorStop(1, "rgba(12,14,18,0.92)");
  ctx.fillStyle = wash;
  ctx.fillRect(0, 0, W, H);
  ctx.strokeStyle = "rgba(255,255,255,0.08)";
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(22.5, 22.5, W - 45, H - 45, 18);
  ctx.stroke();

  // Top row: the So Many Games logo and a FINISHED badge.
  if (logo) ctx.drawImage(logo, 60, 50, (52 * 411) / 96, 52);
  ctx.font = `700 18px ${DISPLAY}`;
  ctx.letterSpacing = "2px";
  const label = "FINISHED";
  const bw = ctx.measureText(label).width + 66;
  const bx = W - 60 - bw;
  ctx.fillStyle = "rgba(92,201,138,0.14)";
  ctx.strokeStyle = "rgba(92,201,138,0.5)";
  ctx.beginPath();
  ctx.roundRect(bx, 56, bw, 40, 20);
  ctx.fill();
  ctx.stroke();
  ctx.strokeStyle = GREEN;
  check(ctx, bx + 26, 76, 20, 3);
  ctx.fillStyle = GREEN;
  ctx.textBaseline = "middle";
  ctx.fillText(label, bx + 44, 77);
  ctx.letterSpacing = "0px";

  // Cover art.
  const cx = 60, cy = 165, cw = 560, ch = 262;
  ctx.save();
  ctx.shadowColor = "rgba(0,0,0,0.6)";
  ctx.shadowBlur = 50;
  ctx.shadowOffsetY = 20;
  ctx.fillStyle = "#1a1d24";
  ctx.beginPath();
  ctx.roundRect(cx, cy, cw, ch, 14);
  ctx.fill();
  ctx.restore();
  ctx.save();
  ctx.beginPath();
  ctx.roundRect(cx, cy, cw, ch, 14);
  ctx.clip();
  if (cover) drawCover(ctx, cover, cx, cy, cw, ch);
  else {
    const g = ctx.createLinearGradient(cx, cy, cx + cw, cy + ch);
    g.addColorStop(0, "#2b3140");
    g.addColorStop(1, "#1a1d24");
    ctx.fillStyle = g;
    ctx.fillRect(cx, cy, cw, ch);
    ctx.fillStyle = "#7d8494";
    ctx.font = `700 72px ${DISPLAY}`;
    ctx.textAlign = "center";
    const initials = game.name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]).join("").toUpperCase();
    ctx.fillText(initials, cx + cw / 2, cy + ch / 2);
    ctx.textAlign = "left";
  }
  ctx.restore();
  ctx.strokeStyle = "rgba(255,255,255,0.1)";
  ctx.beginPath();
  ctx.roundRect(cx + 0.5, cy + 0.5, cw - 1, ch - 1, 14);
  ctx.stroke();

  // Title and the finished checkmark, centred beside the cover.
  const tx = 668;
  const { size, lines } = fitTitle(ctx, game.name, W - 60 - tx);
  const lineH = size * 1.05;
  const blockH = lines.length * lineH + 22 + 64;
  let y = cy + ch / 2 - blockH / 2;
  ctx.fillStyle = "#ecedf0";
  ctx.font = `700 ${size}px ${DISPLAY}`;
  ctx.textBaseline = "top";
  for (const l of lines) {
    ctx.fillText(l, tx, y);
    y += lineH;
  }
  y += 22;
  ctx.save();
  ctx.shadowColor = "rgba(92,201,138,0.5)";
  ctx.shadowBlur = 26;
  ctx.strokeStyle = GREEN;
  ctx.lineWidth = 3;
  ctx.beginPath();
  ctx.arc(tx + 32, y + 32, 30.5, 0, Math.PI * 2);
  ctx.stroke();
  check(ctx, tx + 32, y + 32, 34, 2.6);
  ctx.restore();
  ctx.save();
  ctx.shadowColor = "rgba(92,201,138,0.55)";
  ctx.shadowBlur = 18;
  ctx.fillStyle = "#eafff1";
  ctx.font = `700 34px ${DISPLAY}`;
  ctx.textBaseline = "middle";
  ctx.fillText("Game finished!", tx + 80, y + 33);
  ctx.restore();

  // Bottom row: finished date, hours played and the finish number.
  ctx.strokeStyle = "rgba(255,255,255,0.1)";
  ctx.beginPath();
  ctx.moveTo(60, 491.5);
  ctx.lineTo(W - 60, 491.5);
  ctx.stroke();
  const stats: [string, string, boolean][] = [
    ["FINISHED", finishedDateText(game.finishedAt), false],
    ["PLAYED", hours(game.minutes), false],
    ["FINISHED GAME", number ? `#${number}` : "—", true],
  ];
  stats.forEach(([l, v, green], i) => {
    const x = 60 + i * 360 + (i ? 32 : 0);
    if (i) {
      ctx.beginPath();
      ctx.moveTo(60 + i * 360 + 0.5, 514);
      ctx.lineTo(60 + i * 360 + 0.5, 584);
      ctx.stroke();
    }
    ctx.textBaseline = "middle";
    ctx.font = `600 15px ${BODY}`;
    ctx.letterSpacing = "1.2px";
    ctx.fillStyle = "#7d8494";
    ctx.fillText(l, x, 525);
    ctx.letterSpacing = "0px";
    ctx.font = `700 32px ${DISPLAY}`;
    ctx.fillStyle = green ? GREEN : "#ecedf0";
    ctx.fillText(v, x, 562);
  });
  return canvas;
}

/** The card as a PNG blob. */
export function cardBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => canvas.toBlob((b) => (b ? resolve(b) : reject(new Error("could not draw the card"))), "image/png"));
}

/** A file name such as "Portal 2 finished.png", without characters Windows refuses. */
export function cardFileName(game: Game): string {
  return `${game.name.replace(/[<>:"/\\|?*]+/g, "").trim() || "Game"} finished.png`;
}
