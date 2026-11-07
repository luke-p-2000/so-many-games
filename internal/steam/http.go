// Package steam talks to Steam: the HTTP client shared by every Steam call, the owned-games
// provider, and the API key store. All Steam traffic follows ADR 0011.
package steam

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// UserAgent identifies the app honestly (ADR 0011); main adds the build's version.
var UserAgent = "SoManyGames/dev"

// ErrRateLimited means Steam kept answering 429 after the backoff retries.
var ErrRateLimited = errors.New("Steam is rate limiting requests; try again later")

// HTTP paces requests per host, backs off on 429 and scrubs secrets from errors.
type HTTP struct {
	client  *http.Client
	gap     time.Duration
	backoff time.Duration

	mu   sync.Mutex
	next map[string]time.Time
}

// NewHTTP returns the client used for all Steam requests: at least 1.5 s between calls to
// the same host, and 30 s, 60 s backoff on 429.
func NewHTTP() *HTTP {
	return &HTTP{
		client:  &http.Client{Timeout: 20 * time.Second},
		gap:     1500 * time.Millisecond,
		backoff: 30 * time.Second,
		next:    map[string]time.Time{},
	}
}

// StatusError is a non-200 reply.
type StatusError struct{ Code int }

func (e StatusError) Error() string { return fmt.Sprintf("Steam replied HTTP %d", e.Code) }

// Get fetches url and returns the body. secret, if set, is removed from any error text.
func (h *HTTP) Get(ctx context.Context, url, secret string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		body, status, err := h.once(ctx, url)
		if err != nil {
			return nil, scrub(err, secret)
		}
		switch {
		case status == http.StatusOK:
			return body, nil
		case status == http.StatusTooManyRequests && attempt < 2:
			if err := sleep(ctx, h.backoff*time.Duration(attempt+1)); err != nil {
				return nil, err
			}
		case status == http.StatusTooManyRequests:
			return nil, ErrRateLimited
		default:
			return nil, StatusError{status}
		}
	}
}

func (h *HTTP) once(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", UserAgent)
	if err := h.wait(ctx, req.URL.Host); err != nil {
		return nil, 0, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// wait reserves the next slot for host so concurrent callers stay paced too.
func (h *HTTP) wait(ctx context.Context, host string) error {
	h.mu.Lock()
	now := time.Now()
	at := h.next[host]
	if at.Before(now) {
		at = now
	}
	h.next[host] = at.Add(h.gap)
	h.mu.Unlock()
	return sleep(ctx, time.Until(at))
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func scrub(err error, secret string) error {
	if secret == "" || !strings.Contains(err.Error(), secret) {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), secret, "REDACTED"))
}
