package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBuildFiles(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"build/config.yml":                   "version: '3'\ninfo:\n  productName: \"X\"\n  version: \"1.4.0\" # The application version\n#   version: \"0.0.1\"\n",
		"build/windows/nsis/wails_tools.nsh": "!ifndef INFO_PRODUCTVERSION\n    !define INFO_PRODUCTVERSION \"1.4.0\"\n!endif\n",
		"build/windows/info.json":            `{"fixed": {"file_version": "1.4.0", "product_version": "1.4.0"}, "info": {"0409": {"FileVersion": "1.4.0", "ProductVersion": "1.4.0"}}}`,
	}
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBumpSetsEveryVersion(t *testing.T) {
	root := t.TempDir()
	writeBuildFiles(t, root)
	if err := bump(root, "v1.5.0"); err != nil {
		t.Fatal(err)
	}
	for name := range bumpRules {
		b, _ := os.ReadFile(filepath.Join(root, name))
		if strings.Contains(string(b), `"1.4.0"`) {
			t.Errorf("%s still has 1.4.0:\n%s", name, b)
		}
	}
	cfg, _ := os.ReadFile(filepath.Join(root, "build/config.yml"))
	if !strings.Contains(string(cfg), "version: '3'") || !strings.Contains(string(cfg), `#   version: "0.0.1"`) {
		t.Errorf("bump touched the wrong lines:\n%s", cfg)
	}
}

func TestBumpRejectsBadVersion(t *testing.T) {
	root := t.TempDir()
	writeBuildFiles(t, root)
	if err := bump(root, "1.5"); err == nil {
		t.Fatal("want error for 1.5")
	}
}

func TestSignWritesVerifiableSums(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "a-installer.exe")
	os.WriteFile(exe, []byte("installer"), 0o644)
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	if err := signFiles(base64.StdEncoding.EncodeToString(priv.Seed()), []string{exe}); err != nil {
		t.Fatal(err)
	}
	sums, _ := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	sigText, _ := os.ReadFile(filepath.Join(dir, "SHA256SUMS.sig"))
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigText)))
	if err != nil || !ed25519.Verify(pub, sums, sig) {
		t.Fatalf("signature does not verify: %v", err)
	}
	if !strings.HasSuffix(string(sums), "  a-installer.exe\n") {
		t.Errorf("sums = %q", sums)
	}
}

func TestSignNeedsKey(t *testing.T) {
	if err := signFiles("", []string{"x"}); err == nil {
		t.Fatal("want error without a key")
	}
}
