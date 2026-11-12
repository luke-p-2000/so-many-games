// Command release holds the release chores (ADR 0021):
//
//	go run ./cmd/release bump 1.5.0           set the version in all three build files
//	go run ./cmd/release keygen -out FILE     make the update-signing key pair (once)
//	go run ./cmd/release sign FILE...         write SHA256SUMS and SHA256SUMS.sig beside the files
//
// sign reads the private key from the SOMANYGAMES_SIGNING_KEY environment variable, which the
// release workflow fills from a GitHub secret.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "bump":
		if len(os.Args) != 3 {
			usage()
		}
		err = bump(".", os.Args[2])
	case "keygen":
		fs := flag.NewFlagSet("keygen", flag.ExitOnError)
		out := fs.String("out", "", "file to write the private key to (must not exist)")
		fs.Parse(os.Args[2:])
		if *out == "" {
			usage()
		}
		err = keygen(*out)
	case "sign":
		if len(os.Args) < 3 {
			usage()
		}
		err = signFiles(os.Getenv("SOMANYGAMES_SIGNING_KEY"), os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: release bump X.Y.Z | keygen -out FILE | sign FILE...")
	os.Exit(2)
}

var versionRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// bumpRules are the version spots the release workflow checks, as pattern → replacement.
var bumpRules = map[string][]*regexp.Regexp{
	"build/config.yml":                   {regexp.MustCompile(`(?m)^(\s+version:\s*")[^"]+(")`)},
	"build/windows/nsis/wails_tools.nsh": {regexp.MustCompile(`(INFO_PRODUCTVERSION ")[^"]+(")`)},
	"build/windows/info.json": {
		regexp.MustCompile(`("file_version":\s*")[^"]+(")`),
		regexp.MustCompile(`("product_version":\s*")[^"]+(")`),
		regexp.MustCompile(`("FileVersion":\s*")[^"]+(")`),
		regexp.MustCompile(`("ProductVersion":\s*")[^"]+(")`),
	},
}

// bump sets version in every build file under root. It checks every file before writing any.
func bump(root, version string) error {
	version = strings.TrimPrefix(version, "v")
	if !versionRe.MatchString(version) {
		return fmt.Errorf("version %q is not X.Y.Z", version)
	}
	out := map[string][]byte{}
	for file, rules := range bumpRules {
		b, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			return err
		}
		for _, re := range rules {
			if !re.Match(b) {
				return fmt.Errorf("%s: no match for %s", file, re)
			}
			b = re.ReplaceAll(b, []byte("${1}"+version+"${2}"))
		}
		out[file] = b
	}
	for file, b := range out {
		if err := os.WriteFile(filepath.Join(root, file), b, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("Version set to %s. Commit, then tag v%s.\n", version, version)
	return nil
}

func keygen(out string) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(base64.StdEncoding.EncodeToString(priv.Seed()) + "\n"); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Printf("Private key written to %s. Put its contents in the SOMANYGAMES_SIGNING_KEY secret.\n", out)
	fmt.Printf("Public key for internal/update/key.go:\n%s\n", base64.StdEncoding.EncodeToString(pub))
	return nil
}

// signFiles writes SHA256SUMS (sha256sum format) next to the first file, and SHA256SUMS.sig,
// the base64 ed25519 signature of SHA256SUMS.
func signFiles(seedB64 string, files []string) error {
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(seedB64))
	if err != nil || len(seed) != ed25519.SeedSize {
		return errors.New("SOMANYGAMES_SIGNING_KEY is missing or not a base64 ed25519 seed")
	}
	priv := ed25519.NewKeyFromSeed(seed)
	var sums strings.Builder
	for _, file := range files {
		sum, err := hashFile(file)
		if err != nil {
			return err
		}
		fmt.Fprintf(&sums, "%s  %s\n", sum, filepath.Base(file))
	}
	dir := filepath.Dir(files[0])
	body := []byte(sums.String())
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), body, 0o644); err != nil {
		return err
	}
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, body)) + "\n"
	return os.WriteFile(filepath.Join(dir, "SHA256SUMS.sig"), []byte(sig), 0o644)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
