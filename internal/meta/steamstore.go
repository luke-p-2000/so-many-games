// Package meta fetches store metadata (S5–S7, S9) through a swappable Source and keeps
// the one-at-a-time fetch queue with pause, resume and retry.
package meta

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"somanygames/internal/steam"
)

// Tag is a store tag with its vote weight.
type Tag struct {
	Name   string
	Weight int
}

// Item is what the store knows about one app.
type Item struct {
	Tags        []Tag
	ReviewPct   *int
	ReviewCount *int
	ReleaseDate string // YYYY-MM-DD or ""
	IsGame      bool
	Categories  []string // "single-player", "co-op", "pvp"
	CoverURL    string
}

// Source is the metadata provider (ADR 0006). Items returns only the apps the store found.
type Source interface {
	Items(ctx context.Context, appids []int) (map[int]Item, error)
	Genres(ctx context.Context, appid int) ([]string, error)
	Cover(ctx context.Context, url string) ([]byte, error)
}

// SteamStore implements Source with keyless Steam store endpoints:
// IStoreBrowseService/GetItems for nearly everything and appdetails for genres.
type SteamStore struct {
	HTTP *steam.HTTP

	mu       sync.Mutex
	tagNames map[int]string
}

// BatchSize is the GetItems batch size confirmed by the spike.
const BatchSize = 50

// Store category ids mapped to the wheel's player flags (W1).
var categoryFlags = map[int]string{
	2: "single-player",
	9: "co-op", 38: "co-op", 39: "co-op", 48: "co-op",
	49: "pvp", 36: "pvp", 37: "pvp", 47: "pvp",
}

const coverCDN = "https://shared.akamai.steamstatic.com/store_item_assets/"

type browseItem struct {
	ID      int `json:"id"`
	Success int `json:"success"`
	Type    int `json:"type"`
	Tags    []struct {
		TagID  int `json:"tagid"`
		Weight int `json:"weight"`
	} `json:"tags"`
	Reviews *struct {
		Summary *struct {
			Count int `json:"review_count"`
			Pct   int `json:"percent_positive"`
		} `json:"summary_filtered"`
	} `json:"reviews"`
	Release *struct {
		Date int64 `json:"steam_release_date"`
	} `json:"release"`
	Categories map[string][]int `json:"categories"`
	Assets     *struct {
		Format string `json:"asset_url_format"`
		Header string `json:"header"`
	} `json:"assets"`
}

func (s *SteamStore) Items(ctx context.Context, appids []int) (map[int]Item, error) {
	names, err := s.names(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]map[string]int, len(appids))
	for i, id := range appids {
		ids[i] = map[string]int{"appid": id}
	}
	input, _ := json.Marshal(map[string]any{
		"ids":     ids,
		"context": map[string]any{"language": "english", "country_code": "US", "steam_realm": 1},
		"data_request": map[string]any{
			"include_basic_info": true, "include_tag_count": 20, "include_reviews": true,
			"include_release": true, "include_assets": true, "include_categories": true,
		},
	})
	body, err := s.HTTP.Get(ctx, "https://api.steampowered.com/IStoreBrowseService/GetItems/v1/?input_json="+url.QueryEscape(string(input)), "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Response struct {
			Items []browseItem `json:"store_items"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	out := map[int]Item{}
	for _, b := range resp.Response.Items {
		if b.Success != 1 || b.ID == 0 {
			continue // not found comes back as success 15 with id 0
		}
		out[b.ID] = b.toItem(names)
	}
	return out, nil
}

func (b browseItem) toItem(names map[int]string) Item {
	it := Item{IsGame: b.Type == 0}
	for _, t := range b.Tags {
		if n, ok := names[t.TagID]; ok {
			it.Tags = append(it.Tags, Tag{n, t.Weight})
		}
	}
	if b.Reviews != nil && b.Reviews.Summary != nil {
		it.ReviewPct, it.ReviewCount = &b.Reviews.Summary.Pct, &b.Reviews.Summary.Count
	}
	if b.Release != nil && b.Release.Date > 0 {
		it.ReleaseDate = time.Unix(b.Release.Date, 0).UTC().Format("2006-01-02")
	}
	seen := map[string]bool{}
	for _, list := range b.Categories {
		for _, id := range list {
			if f, ok := categoryFlags[id]; ok && !seen[f] {
				seen[f] = true
				it.Categories = append(it.Categories, f)
			}
		}
	}
	if b.Assets != nil && b.Assets.Header != "" && b.Assets.Format != "" {
		it.CoverURL = coverCDN + strings.Replace(b.Assets.Format, "${FILENAME}", b.Assets.Header, 1)
	}
	return it
}

// names loads the tag-id-to-name list once per run; a failure is retried on the next call.
func (s *SteamStore) names(ctx context.Context) (map[int]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tagNames != nil {
		return s.tagNames, nil
	}
	body, err := s.HTTP.Get(ctx, "https://api.steampowered.com/IStoreService/GetTagList/v1/?language=english", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Response struct {
			Tags []struct {
				ID   int    `json:"tagid"`
				Name string `json:"name"`
			} `json:"tags"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	names := map[int]string{}
	for _, t := range resp.Response.Tags {
		names[t.ID] = t.Name
	}
	s.tagNames = names
	return names, nil
}

// ErrNotFound means the store has no page for the app.
var ErrNotFound = errors.New("Steam has no store page for this game (delisted or region-locked)")

func (s *SteamStore) Genres(ctx context.Context, appid int) ([]string, error) {
	id := strconv.Itoa(appid)
	body, err := s.HTTP.Get(ctx, "https://store.steampowered.com/api/appdetails?l=english&filters=genres&appids="+id, "")
	if err != nil {
		return nil, err
	}
	var resp map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			Genres []struct {
				Description string `json:"description"`
			} `json:"genres"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	r, ok := resp[id]
	if !ok || !r.Success {
		return nil, ErrNotFound
	}
	var out []string
	for _, g := range r.Data.Genres {
		out = append(out, g.Description)
	}
	return out, nil
}

func (s *SteamStore) Cover(ctx context.Context, url string) ([]byte, error) {
	return s.HTTP.Get(ctx, url, "")
}
