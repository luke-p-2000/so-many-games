package steam

import (
	"context"
	"errors"
	"testing"
)

func TestResolveSteamIDWithoutNetwork(t *testing.T) {
	w := &WebAPI{} // no HTTP client: these inputs must not need one
	for _, in := range []string{
		"https://steamcommunity.com/profiles/76561190000000001/",
		"steamcommunity.com/profiles/76561190000000001",
		"76561190000000001",
	} {
		id, err := w.ResolveSteamID(context.Background(), "key", in)
		if err != nil || id != "76561190000000001" {
			t.Errorf("%q -> %q, %v", in, id, err)
		}
	}
}

func TestScrubRemovesKey(t *testing.T) {
	err := scrub(errors.New(`Get "https://x/?key=ABC123&steamid=1": timeout`), "ABC123")
	if got := err.Error(); got != `Get "https://x/?key=REDACTED&steamid=1": timeout` {
		t.Fatalf("scrub = %s", got)
	}
}

func TestMapStatusBadKey(t *testing.T) {
	if !errors.Is(mapStatus(StatusError{403}), ErrBadKey) {
		t.Fatal("403 should map to ErrBadKey")
	}
}
