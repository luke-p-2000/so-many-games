package update

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo is where releases are published.
const Repo = "luke-p-2000/so-many-games"

// PublicKey verifies SHA256SUMS.sig. Its private half is the SOMANYGAMES_SIGNING_KEY secret
// (see cmd/release and ADR 0021).
const PublicKey = "tqbpq+H2wWj3Ru7hNFjsXZrMBG+c2ZaIknjLas+sq4w="

// DecodedKey is PublicKey as bytes.
func DecodedKey() []byte {
	b, _ := base64.StdEncoding.DecodeString(PublicKey)
	return b
}

// GitHub reads the latest release from the GitHub REST API. /releases/latest skips drafts and
// prereleases.
type GitHub struct {
	HTTP      *http.Client
	UserAgent string
	BaseURL   string // https://api.github.com, or a test server
	Repo      string
}

// ErrRateLimited is GitHub's hourly limit for unauthenticated calls.
var ErrRateLimited = errors.New("GitHub is limiting update checks; try again in an hour")

func (g *GitHub) Latest(ctx context.Context) (Release, error) {
	url := strings.TrimSuffix(g.BaseURL, "/") + "/repos/" + g.Repo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", g.UserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := g.HTTP.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("couldn't reach GitHub: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusTooManyRequests:
		return Release{}, ErrRateLimited
	case http.StatusNotFound:
		return Release{}, errors.New("no releases are published yet")
	default:
		return Release{}, fmt.Errorf("GitHub replied HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return Release{}, fmt.Errorf("couldn't read GitHub's reply: %w", err)
	}
	rel := Release{Version: body.TagName, Notes: body.Body, PageURL: body.HTMLURL, Assets: map[string]Asset{}}
	for _, a := range body.Assets {
		rel.Assets[a.Name] = Asset{URL: a.URL, Size: a.Size}
	}
	return rel, nil
}

// ReleasesPage is opened for the portable exe, and is the only page the app opens.
const ReleasesPage = "https://github.com/" + Repo + "/releases/latest"

// IsInstalled reports whether exe runs from the per-user install folder the installer uses
// (ADR 0001), rather than as the portable exe.
func IsInstalled(exe, localAppData string) bool {
	if exe == "" || localAppData == "" {
		return false
	}
	want := filepath.Join(localAppData, "Programs", "So Many Games")
	return strings.EqualFold(filepath.Clean(filepath.Dir(exe)), filepath.Clean(want))
}

// RunningInstalled is IsInstalled for this process.
func RunningInstalled() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return IsInstalled(exe, os.Getenv("LOCALAPPDATA"))
}

// StartDetached starts the installer; it keeps running after the app quits.
func StartDetached(path string, args ...string) error {
	cmd := exec.Command(path, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
