// A Share button that opens a dialog previewing a drawn card, with Copy image and Save PNG…
// (the Stats and Tier list cards; the finished-game card has its own dialog).
import { useEffect, useRef, useState } from "react";
import { DataService, errorText } from "../api";
import { cardBlob } from "../finishedCard";
import { useReportError, useToast } from "../toast";

type Props = {
  /** Dialog heading, e.g. "Games finished card". */
  title: string;
  /** Suggested file name for Save PNG…. */
  fileName: string;
  /** Draws the card; called when the dialog opens and whenever drawKey changes. */
  draw: () => Promise<HTMLCanvasElement>;
  drawKey: string;
  /** Cards taller than 1200 × 630 (the tier list) scroll inside the preview. */
  tall?: boolean;
  disabled?: boolean;
};

export function ShareButton(props: Props) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" className="btn small" onClick={() => setOpen(true)} disabled={props.disabled} title="Image to copy or save">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
          <path d="M12 15V4M7.5 8.5L12 4l4.5 4.5" /><path d="M5 13v5a2 2 0 002 2h10a2 2 0 002-2v-5" />
        </svg>
        Share
      </button>
      {open && <ShareDialog {...props} onClose={() => setOpen(false)} />}
    </>
  );
}

function ShareDialog({ title, fileName, draw, drawKey, tall, onClose }: Props & { onClose: () => void }) {
  const report = useReportError();
  const toast = useToast();
  const ref = useRef<HTMLDialogElement>(null);
  const canvas = useRef<HTMLCanvasElement | null>(null);
  const [preview, setPreview] = useState("");

  useEffect(() => {
    if (ref.current && !ref.current.open) ref.current.showModal();
  }, []);
  useEffect(() => {
    let live = true;
    draw().then((c) => {
      if (!live) return;
      canvas.current = c;
      setPreview(c.toDataURL("image/png"));
    }, report);
    return () => {
      live = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [drawKey]);

  const copy = async () => {
    try {
      if (!canvas.current) return;
      const blob = await cardBlob(canvas.current);
      await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
      toast({ message: "Card copied to the clipboard" });
    } catch (err) {
      report(err);
    }
  };
  const save = async () => {
    if (!canvas.current) return;
    try {
      const path = await DataService.SaveImage(fileName, canvas.current.toDataURL("image/png").split(",")[1]);
      toast({ message: `Saved ${path}` });
    } catch (err) {
      if (errorText(err) !== "cancelled") report(err);
    }
  };

  return (
    <dialog ref={ref} className="modal finished-dialog" aria-label={title} onCancel={(e) => { e.preventDefault(); onClose(); }}>
      <div className="modal-head">
        <h2>{title}</h2>
        <button className="btn ghost small" aria-label="Close" onClick={onClose}>✕</button>
      </div>
      <div className={`card-preview${tall ? " tall" : ""}`}>{preview ? <img src={preview} alt={title} /> : <span className="faint">Drawing the card…</span>}</div>
      <div className="modal-foot">
        <button className="btn ghost" onClick={onClose}>Close</button>
        <button className="btn" onClick={save} disabled={!preview}>Save PNG…</button>
        <button className="btn primary" onClick={copy} disabled={!preview}>Copy image</button>
      </div>
    </dialog>
  );
}
