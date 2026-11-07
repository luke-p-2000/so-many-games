// Package library owns game statuses and the rules that change them: automatic changes on
// sync (L2), the first-import Classify flow (L7, L8) and hidden entries (L6).
package library

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Status is one of the eight statuses in L1 (Dead added by ADR 0013).
type Status string

const (
	Unplayed Status = "unplayed"
	ToPlay   Status = "to_play"
	Playing  Status = "playing"
	Shelved  Status = "shelved"
	Finished Status = "finished"
	NotForMe Status = "not_for_me"
	Endless  Status = "endless"
	Dead     Status = "dead" // unplayable: servers shut down or broken (ADR 0013)
)

// Statuses is the display order used by every status control.
var Statuses = []Status{Unplayed, ToPlay, Playing, Shelved, Finished, NotForMe, Endless, Dead}

// Valid reports whether s is one of the statuses.
func (s Status) Valid() bool { return slices.Contains(Statuses, s) }

// ParseStatus validates a status coming from the UI or an import file.
func ParseStatus(s string) (Status, error) {
	if st := Status(s); st.Valid() {
		return st, nil
	}
	return "", fmt.Errorf("unknown status %q", s)
}

// AfterSync applies L2 to a game already in the library (ADR 0002): new playtime moves
// Unplayed, To play and Shelved to Playing; nothing else (Finished, Not for me, Endless,
// Dead) changes automatically.
func AfterSync(current Status, prevMinutes, minutes int) Status {
	if minutes > prevMinutes && (current == Unplayed || current == ToPlay || current == Shelved) {
		return Playing
	}
	return current
}

// ForNewGame is the status of a game first seen in a later sync (L8: no classification).
func ForNewGame(minutes int) Status {
	if minutes == 0 {
		return Unplayed
	}
	return Playing
}

// ClassifyDefault is the staged status on the Classify screen (L7): Playing when played
// within the window, otherwise Shelved.
func ClassifyDefault(lastPlayed *time.Time, now time.Time, windowDays int) Status {
	if PlayedWithin(lastPlayed, now, windowDays) {
		return Playing
	}
	return Shelved
}

// PlayedWithin reports whether lastPlayed falls in the last days days.
func PlayedWithin(lastPlayed *time.Time, now time.Time, days int) bool {
	return lastPlayed != nil && now.Sub(*lastPlayed) <= time.Duration(days)*24*time.Hour
}

// HiddenReason returns why name should be hidden by pattern or store type (L6), or "".
// Patterns match case-insensitively as whole words, so "Demo" hides "X Demo" but not "Demolition".
func HiddenReason(name string, patterns []string, isGame *bool) string {
	return hiddenMatcher(patterns)(name, isGame)
}

// hiddenMatcher compiles the patterns once for checking many games.
func hiddenMatcher(patterns []string) func(name string, isGame *bool) string {
	type rule struct {
		pattern string
		re      *regexp.Regexp
	}
	var rules []rule
	for _, p := range patterns {
		if p = strings.TrimSpace(p); p != "" {
			rules = append(rules, rule{p, regexp.MustCompile(`(?i)(^|\W)` + regexp.QuoteMeta(p) + `($|\W)`)})
		}
	}
	return func(name string, isGame *bool) string {
		for _, r := range rules {
			if r.re.MatchString(name) {
				return r.pattern
			}
		}
		if isGame != nil && !*isGame {
			return "not a game"
		}
		return ""
	}
}
