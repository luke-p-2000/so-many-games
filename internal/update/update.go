// Package update checks GitHub for a newer release, downloads its installer, verifies it
// against the signed SHA256SUMS and starts it (ADR 0021). It runs only when the owner asks.
package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Asset names on each GitHub release (see .github/workflows/release.yml).
const (
	InstallerName = "somanygames-amd64-installer.exe"
	SumsName      = "SHA256SUMS"
	SigName       = "SHA256SUMS.sig"
)

// Size caps, so a bad reply can't fill the disk.
const (
	maxSmall     = 64 << 10
	maxInstaller = 512 << 20
)

// ErrDevBuild means the app has no release version to compare against.
var ErrDevBuild = errors.New("updates are off in development builds")

// ErrVerify means a download did not match the signed checksums; nothing is installed.
var ErrVerify = errors.New("the update failed its security check and was not installed")

// Release is the newest published release.
type Release struct {
	Version string
	Notes   string
	PageURL string
	Assets  map[string]Asset
}

// Asset is one file on a release.
type Asset struct {
	URL  string
	Size int64
}

// Source finds the newest release; GitHub in the app, a fake in tests.
type Source interface {
	Latest(ctx context.Context) (Release, error)
}

// Status is what Settings shows after a check.
type Status struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	Notes     string `json:"notes"`
	PageURL   string `json:"pageUrl"`
	Size      int64  `json:"size"`
	// Installed is false for the portable exe, which can only open the release page.
	Installed bool `json:"installed"`
}

// Progress reports download progress in bytes.
type Progress struct {
	Done  int64 `json:"done"`
	Total int64 `json:"total"`
}

// Updater holds what a check and an install need.
type Updater struct {
	Current   string
	Source    Source
	HTTP      *http.Client
	UserAgent string
	Dir       string // where the installer is downloaded; emptied before each download
	PublicKey ed25519.PublicKey
	Installed bool
	Start     func(path string, args ...string) error
}

// Check asks for the newest release and compares it with the running version.
func (u *Updater) Check(ctx context.Context) (Status, Release, error) {
	cur, ok := parseVersion(u.Current)
	if !ok {
		return Status{}, Release{}, ErrDevBuild
	}
	rel, err := u.Source.Latest(ctx)
	if err != nil {
		return Status{}, Release{}, err
	}
	latest, ok := parseVersion(rel.Version)
	if !ok {
		return Status{}, Release{}, fmt.Errorf("the latest release has an unreadable version %q", rel.Version)
	}
	st := Status{
		Current:   u.Current,
		Latest:    strings.TrimPrefix(rel.Version, "v"),
		Available: newer(latest, cur),
		Notes:     rel.Notes,
		PageURL:   rel.PageURL,
		Size:      rel.Assets[InstallerName].Size,
		Installed: u.Installed,
	}
	return st, rel, nil
}

// Download fetches and verifies rel's installer and returns its path. The signature on
// SHA256SUMS is checked before the installer is fetched, and the installer's hash before it
// is renamed into place, so an unverified file is never left where Install would run it.
func (u *Updater) Download(ctx context.Context, rel Release, progress func(Progress)) (string, error) {
	inst, ok := rel.Assets[InstallerName]
	sums, ok2 := rel.Assets[SumsName]
	sig, ok3 := rel.Assets[SigName]
	if !ok || !ok2 || !ok3 {
		return "", fmt.Errorf("release %s has no signed installer; download it from the release page", rel.Version)
	}
	sumsBody, err := u.fetchSmall(ctx, sums.URL)
	if err != nil {
		return "", err
	}
	sigBody, err := u.fetchSmall(ctx, sig.URL)
	if err != nil {
		return "", err
	}
	want, err := VerifySums(u.PublicKey, sumsBody, sigBody, InstallerName)
	if err != nil {
		return "", err
	}

	if err := os.RemoveAll(u.Dir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(u.Dir, 0o755); err != nil {
		return "", err
	}
	final := filepath.Join(u.Dir, InstallerName)
	part := final + ".part"
	got, err := u.fetchFile(ctx, inst.URL, part, inst.Size, progress)
	if err != nil {
		os.Remove(part)
		return "", err
	}
	if got != want {
		os.Remove(part)
		return "", ErrVerify
	}
	if err := os.Rename(part, final); err != nil {
		return "", err
	}
	return final, nil
}

// Install starts the verified installer silently; it waits for the app to close, upgrades in
// place and reopens the app. The caller quits straight after.
func (u *Updater) Install(path string) error {
	return u.Start(path, "/S", "/RELAUNCH")
}

// VerifySums checks the ed25519 signature on sums and returns the hex SHA-256 listed for name.
func VerifySums(pub ed25519.PublicKey, sums, sigText []byte, name string) (string, error) {
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigText)))
	if err != nil || len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, sums, sig) {
		return "", ErrVerify
	}
	for _, line := range strings.Split(string(sums), "\n") {
		hash, file, ok := strings.Cut(strings.TrimSpace(line), "  ")
		if ok && strings.TrimPrefix(file, "*") == name && len(hash) == 64 {
			return strings.ToLower(hash), nil
		}
	}
	return "", ErrVerify
}

func (u *Updater) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", u.UserAgent)
	resp, err := u.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach GitHub: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub replied HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

func (u *Updater) fetchSmall(ctx context.Context, url string) ([]byte, error) {
	resp, err := u.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxSmall+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxSmall {
		return nil, ErrVerify
	}
	return b, nil
}

// fetchFile streams url to path, hashing as it goes, and returns the hex SHA-256.
func (u *Updater) fetchFile(ctx context.Context, url, path string, size int64, progress func(Progress)) (string, error) {
	resp, err := u.get(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	total := resp.ContentLength
	if total <= 0 {
		total = size
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	w := &counter{w: io.MultiWriter(f, h), total: total, report: progress}
	n, err := io.Copy(w, io.LimitReader(resp.Body, maxInstaller+1))
	if err != nil {
		return "", err
	}
	if n > maxInstaller {
		return "", ErrVerify
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// counter reports progress about every 1% so the UI isn't flooded with events.
type counter struct {
	w            io.Writer
	done, total  int64
	lastReported int64
	report       func(Progress)
}

func (c *counter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.done += int64(n)
	if c.report != nil && (c.total <= 0 || c.done-c.lastReported >= c.total/100 || c.done == c.total) {
		c.lastReported = c.done
		c.report(Progress{Done: c.done, Total: c.total})
	}
	return n, err
}

type version [3]int

// parseVersion reads "1.4.0" or "v1.4.0". Anything else, including "dev", is not a version.
func parseVersion(s string) (version, bool) {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(s), "v"), ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var v version
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return version{}, false
		}
		v[i] = n
	}
	return v, true
}

func newer(a, b version) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}
