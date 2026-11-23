package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSignedDistributionRejectsTampering(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	for _, target := range []string{"binary", "catalog", "checksums", "signature", "wrong-key", "valid"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			for _, platform := range Platforms {
				if err := os.WriteFile(filepath.Join(dir, BinaryName("consize-prometheus-plugin", "0.1.0", platform)), []byte("test executable "+platform), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if err := Sign(dir, "0.1.0", private, time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			if err := Verify(dir, public); err != nil {
				t.Fatal(err)
			}
			name := ""
			switch target {
			case "binary":
				name = BinaryName("consize-prometheus-plugin", "0.1.0", Platforms[0])
			case "catalog":
				name = "catalog.json"
			case "checksums":
				name = "SHA256SUMS"
			case "signature":
				name = "SHA256SUMS.sig"
			}
			if name != "" {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("tampered"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			key := public
			if target == "wrong-key" {
				key, _, _ = ed25519.GenerateKey(rand.Reader)
			}
			err := Verify(dir, key)
			if target == "valid" && err != nil {
				t.Fatal(err)
			}
			if target != "valid" && err == nil {
				t.Fatal("tampered release accepted")
			}
		})
	}
}

func TestSigningKeyValidation(t *testing.T) {
	_, private, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := ReadPrivate(base64.StdEncoding.EncodeToString(private)); err != nil {
		t.Fatal(err)
	}
	private[len(private)-1] ^= 1
	if _, err := ReadPrivate(base64.StdEncoding.EncodeToString(private)); err == nil {
		t.Fatal("inconsistent key accepted")
	}
	if _, err := ReadPrivate("invalid"); err == nil {
		t.Fatal("invalid key accepted")
	}
}
