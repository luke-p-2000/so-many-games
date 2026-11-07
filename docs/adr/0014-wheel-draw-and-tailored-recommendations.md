# 0014 Wheel draw, multi-tag filter and Tailored for you

Status: Accepted · 2026-11-07 (owner feedback after the second build)

## Wheel
- The wheel is drawn as soon as the filters, bias switch or size change, so it always shows the games it holds. Spin picks one of the games on the wheel; it no longer redraws the wheel. Spin again spins the same wheel, and **New games** redraws it when the pool is larger than the wheel.
- The All reel scrolls past the marker and stops on the winner, like the wheel.
- The tag filter takes several tags. A game must have every selected tag, as all wheel filters combine by AND. Changing this to "any of" is a one-line change in `wheel.Filter` if AND turns out too narrow.

## Recommendations
- **Tailored for you** is the first section. It ranks games that share at least 2 tags with the taste profile, most shared tags first and Wilson score among equal counts. The games it is showing are left out of the "Because you play ..." rows, so no game appears twice on screen. Home shows the first two sections, so Tailored for you comes first there too.
- Each section holds up to 30 ranked games and shows as many as fit on one line (at most 6). Its **Refresh** button shows the next batch, wrapping round to the start so every batch fills the line.
- `rec_shown` is logged for the cards actually on screen, once per game per run, rather than for every game a section holds.
- The To play button on recommendation cards (ADR 0013) is removed; the compact status control covers it.

## Naming
- The Pick back up view (R7) is labelled **Resume shelved games**, with a tooltip saying what it lists. The requirement id and route keep the old name.
