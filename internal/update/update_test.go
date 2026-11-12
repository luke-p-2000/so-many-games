package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGitHub serves a release API reply and its assets.
type fakeGitHub struct {
	tag       string
	installer []byte
	sums      []byte
	sig       []byte
	status    int
}

func signed(t *testing.T, priv ed25519.PrivateKey, installer []byte) (sums, sig []byte) {
	t.Helper()
	h := sha256.Sum256(installer)
	sums = []byte(fmt.Sprintf("%s  %s\n%s  somanygames.exe\n", hex.EncodeToString(h[:]), InstallerName, strings.Repeat("0", 64)))
	sig = []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sums)) + "\n")
	return
}

func (f *fakeGitHub) server(t *testing.T) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "SoManyGames/1.4.0" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/repos/" + Repo + "/releases/latest":
			if f.status != 0 {
				w.WriteHeader(f.status)
				return
			}
			fmt.Fprintf(w, `{"tag_name":%q,"body":"notes","html_url":"https://github.com/x","assets":[
				{"name":%q,"browser_download_url":"%s/dl/inst","size":%d},
				{"name":"SHA256SUMS","browser_download_url":"%s/dl/sums","size":1},
				{"name":"SHA256SUMS.sig","browser_download_url":"%s/dl/sig","size":1}]}`,
				f.tag, InstallerName, srv.URL, len(f.installer), srv.URL, srv.URL)
		case "/dl/inst":
			w.Write(f.installer)
		case "/dl/sums":
			w.Write(f.sums)
		case "/dl/sig":
			w.Write(f.sig)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newUpdater(t *testing.T, srv *httptest.Server, pub ed25519.PublicKey) *Updater {
	return &Updater{
		Current:   "1.4.0",
		Source:    &GitHub{HTTP: srv.Client(), UserAgent: "SoManyGames/1.4.0", BaseURL: srv.URL, Repo: Repo},
		HTTP:      srv.Client(),
		UserAgent: "SoManyGames/1.4.0",
		Dir:       filepath.Join(t.TempDir(), "update"),
		PublicKey: pub,
		Installed: true,
	}
}

func TestCheckFindsNewerRelease(t *testing.T) {
	f := &fakeGitHub{tag: "v1.5.0", installer: []byte("new")}
	u := newUpdater(t, f.server(t), nil)
	st, _, err := u.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !st.Available || st.Latest != "1.5.0" || st.Notes != "notes" || st.Size != 3 || !st.Installed {
		t.Fatalf("status = %+v", st)
	}
}

func TestCheckSameOrOlderIsNotAvailable(t *testing.T) {
	for _, tag := range []string{"v1.4.0", "v1.3.9", "1.4.0"} {
		f := &fakeGitHub{tag: tag}
		st, _, err := newUpdater(t, f.server(t), nil).Check(context.Background())
		if err != nil || st.Available {
			t.Errorf("%s: available=%v err=%v", tag, st.Available, err)
		}
	}
}

func TestCheckComparesNumerically(t *testing.T) {
	f := &fakeGitHub{tag: "v1.10.0"}
	u := newUpdater(t, f.server(t), nil)
	u.Current = "1.9.3"
	if st, _, _ := u.Check(context.Background()); !st.Available {
		t.Fatal("1.10.0 should be newer than 1.9.3")
	}
}

func TestCheckDevBuild(t *testing.T) {
	f := &fakeGitHub{tag: "v1.5.0"}
	u := newUpdater(t, f.server(t), nil)
	u.Current = "dev"
	if _, _, err := u.Check(context.Background()); !errors.Is(err, ErrDevBuild) {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckRateLimited(t *testing.T) {
	f := &fakeGitHub{status: http.StatusForbidden}
	if _, _, err := newUpdater(t, f.server(t), nil).Check(context.Background()); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v", err)
	}
}

func TestDownloadVerifiesAndInstalls(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	f := &fakeGitHub{tag: "v1.5.0", installer: []byte("the new installer")}
	f.sums, f.sig = signed(t, priv, f.installer)
	u := newUpdater(t, f.server(t), pub)
	var started []string
	u.Start = func(path string, args ...string) error { started = append([]string{path}, args...); return nil }

	_, rel, err := u.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var last Progress
	path, err := u.Download(context.Background(), rel, func(p Progress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "the new installer" {
		t.Fatalf("installer = %q", b)
	}
	if last.Done != int64(len(f.installer)) {
		t.Errorf("last progress = %+v", last)
	}
	if err := u.Install(path); err != nil {
		t.Fatal(err)
	}
	if strings.Join(started, " ") != path+" /S /RELAUNCH" {
		t.Errorf("started %v", started)
	}
}

func TestDownloadRejectsTamperedInstaller(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	f := &fakeGitHub{tag: "v1.5.0", installer: []byte("genuine")}
	f.sums, f.sig = signed(t, priv, f.installer)
	f.installer = []byte("tampered")
	u := newUpdater(t, f.server(t), pub)
	_, rel, _ := u.Check(context.Background())
	if _, err := u.Download(context.Background(), rel, nil); !errors.Is(err, ErrVerify) {
		t.Fatalf("err = %v", err)
	}
	if entries, _ := os.ReadDir(u.Dir); len(entries) != 0 {
		t.Errorf("left files behind: %v", entries)
	}
}

func TestDownloadRejectsWrongKey(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	f := &fakeGitHub{tag: "v1.5.0", installer: []byte("x")}
	f.sums, f.sig = signed(t, priv, f.installer)
	u := newUpdater(t, f.server(t), otherPub)
	_, rel, _ := u.Check(context.Background())
	if _, err := u.Download(context.Background(), rel, nil); !errors.Is(err, ErrVerify) {
		t.Fatalf("err = %v", err)
	}
}

func TestDownloadNeedsSignedAssets(t *testing.T) {
	u := &Updater{}
	rel := Release{Version: "v1.5.0", Assets: map[string]Asset{InstallerName: {URL: "x"}}}
	if _, err := u.Download(context.Background(), rel, nil); err == nil {
		t.Fatal("want error for a release without SHA256SUMS")
	}
}

func TestVerifySumsRejectsEditedSums(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	sums, sig := signed(t, priv, []byte("x"))
	edited := append([]byte{}, sums...)
	edited[0] ^= 1
	if _, err := VerifySums(pub, edited, sig, InstallerName); !errors.Is(err, ErrVerify) {
		t.Fatalf("err = %v", err)
	}
	if _, err := VerifySums(pub, sums, sig, "missing.exe"); !errors.Is(err, ErrVerify) {
		t.Fatalf("missing file: err = %v", err)
	}
}

func TestIsInstalled(t *testing.T) {
	local := `C:\Users\u\AppData\Local`
	cases := map[string]bool{
		`C:\Users\u\AppData\Local\Programs\So Many Games\somanygames.exe`: true,
		`c:\users\u\appdata\local\programs\so many games\somanygames.exe`: true,
		`C:\Users\u\Downloads\somanygames.exe`:                            false,
		``:                                                                false,
	}
	for exe, want := range cases {
		if got := IsInstalled(exe, local); got != want {
			t.Errorf("IsInstalled(%q) = %v", exe, got)
		}
	}
}

func TestPublicKeyDecodes(t *testing.T) {
	if len(DecodedKey()) != ed25519.PublicKeySize {
		t.Fatal("PublicKey is not a base64 ed25519 public key")
	}
}
