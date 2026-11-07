# 0005 Time-to-beat is deferred

Status: Accepted · 2026-11-07 (owner decision)

## Context
HowLongToBeat has no public API. Its search sits behind a token tied to IP and user agent, an anti-bot gate the app must not get around. Steam has no time-to-beat data.

## Decision
Time-to-beat moves to "Deferred and future features" until a sanctioned API is found, such as IGDB with the owner's own credentials. v1 has:
- no time-to-beat provider or interface;
- no `time_to_beat_h` column;
- no "~Nh to beat" text on cards;
- no time-to-beat wheel filter.

## Consequences
S8 moves to the deferred list. H1 and W1 lose their time-to-beat parts, and the mocks' "~Nh" text is ignored.
