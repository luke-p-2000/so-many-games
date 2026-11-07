package steam

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Owned is one game from GetOwnedGames.
type Owned struct {
	AppID      int
	Name       string
	Minutes    int
	LastPlayed *time.Time // nil when never played
}

// Distinct, plain-language errors (design: Errors).
var (
	ErrBadKey          = errors.New("Steam rejected the API key; check it in Settings")
	ErrPrivateProfile  = errors.New(`Steam returned no games. Set "Game details" to Public in your Steam privacy settings`)
	ErrProfileNotFound = errors.New("no Steam profile matches that URL or name")
)

// WebAPI is the owned-games provider backed by the Steam Web API.
type WebAPI struct {
	HTTP *HTTP
	Keys KeyStore
}

const apiBase = "https://api.steampowered.com/"

// OwnedGames returns the library of steamID. It is the only call, with ResolveSteamID,
// that sends the API key (ADR 0011).
func (w *WebAPI) OwnedGames(ctx context.Context, steamID string) ([]Owned, error) {
	key, err := w.Keys.Get()
	if err != nil {
		return nil, err
	}
	return w.ownedGames(ctx, key, steamID)
}

// Check tests a key and SteamID before they are saved (S1).
func (w *WebAPI) Check(ctx context.Context, key, steamID string) error {
	_, err := w.ownedGames(ctx, key, steamID)
	return err
}

func (w *WebAPI) ownedGames(ctx context.Context, key, steamID string) ([]Owned, error) {
	q := url.Values{
		"key": {key}, "steamid": {steamID},
		"include_appinfo": {"1"}, "include_played_free_games": {"1"}, "skip_unvetted_apps": {"0"},
	}
	body, err := w.HTTP.Get(ctx, apiBase+"IPlayerService/GetOwnedGames/v1/?"+q.Encode(), key)
	if err != nil {
		return nil, mapStatus(err)
	}
	var resp struct {
		Response struct {
			Games []struct {
				AppID          int    `json:"appid"`
				Name           string `json:"name"`
				Minutes        int    `json:"playtime_forever"`
				LastPlayedUnix int64  `json:"rtime_last_played"`
			} `json:"games"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Response.Games) == 0 {
		return nil, ErrPrivateProfile
	}
	out := make([]Owned, 0, len(resp.Response.Games))
	for _, g := range resp.Response.Games {
		o := Owned{AppID: g.AppID, Name: g.Name, Minutes: g.Minutes}
		if g.LastPlayedUnix > 0 {
			t := time.Unix(g.LastPlayedUnix, 0).UTC()
			o.LastPlayed = &t
		}
		out = append(out, o)
	}
	return out, nil
}

var (
	profilesURL = regexp.MustCompile(`/profiles/(\d{17})/?$`)
	steamID64   = regexp.MustCompile(`^\d{17}$`)
	vanityURL   = regexp.MustCompile(`/id/([^/]+)/?$`)
)

// ResolveSteamID accepts a profile URL, a custom-URL name or a 64-bit SteamID.
func (w *WebAPI) ResolveSteamID(ctx context.Context, key, input string) (string, error) {
	input = strings.TrimSpace(input)
	if m := profilesURL.FindStringSubmatch(input); m != nil {
		return m[1], nil
	}
	if steamID64.MatchString(input) {
		return input, nil
	}
	vanity := input
	if m := vanityURL.FindStringSubmatch(input); m != nil {
		vanity = m[1]
	}
	if vanity == "" || strings.ContainsAny(vanity, "/:") {
		return "", ErrProfileNotFound
	}
	q := url.Values{"key": {key}, "vanityurl": {vanity}}
	body, err := w.HTTP.Get(ctx, apiBase+"ISteamUser/ResolveVanityURL/v1/?"+q.Encode(), key)
	if err != nil {
		return "", mapStatus(err)
	}
	var resp struct {
		Response struct {
			Success int    `json:"success"`
			SteamID string `json:"steamid"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if resp.Response.Success != 1 {
		return "", ErrProfileNotFound
	}
	return resp.Response.SteamID, nil
}

func mapStatus(err error) error {
	var se StatusError
	if errors.As(err, &se) && (se.Code == http.StatusUnauthorized || se.Code == http.StatusForbidden) {
		return ErrBadKey
	}
	return err
}
