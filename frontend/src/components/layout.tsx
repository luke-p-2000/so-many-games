// Layout helpers that keep rows on one line instead of wrapping raggedly.
import { useLayoutEffect, useRef, useState } from "react";

/** How many items of at least minWidth fit across the element, capped at max. */
export function useFitCount<T extends HTMLElement>(minWidth: number, gap: number, max: number) {
  const ref = useRef<T>(null);
  const [count, setCount] = useState(max);
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const measure = () => setCount(Math.max(1, Math.min(max, Math.floor((el.clientWidth + gap) / (minWidth + gap)))));
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [minWidth, gap, max]);
  return { ref, count };
}

/** Chips limited to two lines; the rest sit behind a "+N" chip that shows them on hover or focus. */
export function ClampedChips({ items, accent = false }: { items: string[]; accent?: boolean }) {
  const box = useRef<HTMLDivElement>(null);
  const [shown, setShown] = useState(items.length);
  const key = items.join("|"); // stable across renders that pass a new array

  // Render everything and keep the chips that start on the first two lines. A second pass
  // then drops one more at a time while the "+N" chip itself would land on a third line.
  const lineOf = (el: HTMLElement, first: HTMLElement) => Math.round((el.offsetTop - first.offsetTop) / (first.offsetHeight + 6));
  useLayoutEffect(() => {
    const measure = () => {
      setShown(items.length);
      requestAnimationFrame(() => {
        const chips = [...(box.current?.querySelectorAll<HTMLElement>("[data-chip]") ?? [])];
        if (!chips.length) return;
        const third = chips.findIndex((c) => lineOf(c, chips[0]) >= 2);
        setShown(third === -1 ? items.length : third);
      });
    };
    measure();
    // Hiding chips changes the height, so only a width change triggers a re-measure.
    let width = box.current?.clientWidth;
    const ro = new ResizeObserver(() => {
      const w = box.current?.clientWidth;
      if (w !== width) {
        width = w;
        measure();
      }
    });
    if (box.current) ro.observe(box.current);
    return () => ro.disconnect();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  useLayoutEffect(() => {
    const first = box.current?.querySelector<HTMLElement>("[data-chip]");
    const more = box.current?.querySelector<HTMLElement>(".chip.more");
    if (first && more && shown > 1 && lineOf(more, first) >= 2) setShown(shown - 1);
  }, [shown]);

  const rest = items.slice(shown);
  const cls = `chip${accent ? " accent" : ""}`;
  return (
    <div className="chips clamped" ref={box}>
      {items.slice(0, shown).map((t) => <span key={t} data-chip className={cls}>{t}</span>)}
      {rest.length > 0 && (
        <span className="chip more" tabIndex={0} aria-label={`${rest.length} more: ${rest.join(", ")}`}>
          +{rest.length}
          <span className="more-pop" role="tooltip">
            {rest.map((t) => <span key={t} className={cls}>{t}</span>)}
          </span>
        </span>
      )}
    </div>
  );
}

