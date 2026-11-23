// Package release implements Consize's signed catalog around versioned Go binaries.
package release

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/consize-oss/consize-prometheus-plugin/prometheus"
	"github.com/consize-oss/consize/pkg/plugin/marketplace"
)

const Repository = "consize-oss/consize-prometheus-plugin"

var Platforms = []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64"}
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[a-zA-Z0-9]+([.-][a-zA-Z0-9]+)*)?$`)

func ReadPrivate(raw string) (ed25519.PrivateKey, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil || len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("expected a base64 Ed25519 private key")
	}
	canonical := ed25519.NewKeyFromSeed(key[:ed25519.SeedSize])
	if !canonical.Equal(ed25519.PrivateKey(key)) {
		return nil, fmt.Errorf("inconsistent Ed25519 private key")
	}
	return ed25519.PrivateKey(key), nil
}

func BinaryName(name, version, platform string) string {
	return name + "_" + version + "_" + strings.ReplaceAll(platform, "/", "_")
}

// Build refuses to overwrite a distribution directory. Released versions are immutable.
func Build(ctx context.Context, dir, version string, key ed25519.PrivateKey) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("version must be semantic version without v prefix")
	}
	if len(key) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid signing key")
	}
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	for _, platform := range Platforms {
		target := strings.Split(platform, "/")
		for _, name := range []string{"consize-prometheus-plugin", "consize-pluginctl"} {
			path := filepath.Join(dir, BinaryName(name, version, platform))
			cmd := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -X github.com/consize-oss/consize-prometheus-plugin/prometheus.Version="+version, "-o", path, "./cmd/"+name)
			cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off", "GOOS="+target[0], "GOARCH="+target[1])
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("build %s %s: %w\n%s", name, platform, err, output)
			}
		}
	}
	for _, name := range []string{"LICENSE", "README.md", "plugin.json", "config.schema.json"} {
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			return err
		}
	}
	return Sign(dir, version, key, time.Now().UTC())
}

// Sign binds the executable digest, size, platform and manifest into the catalog.
func Sign(dir, version string, key ed25519.PrivateKey, now time.Time) error {
	if !versionPattern.MatchString(version) || len(key) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid release identity")
	}
	manifest := prometheus.NewWithClient(nil, prometheus.Config{}).Manifest()
	manifest.Version = version
	catalog := marketplace.Catalog{Protocol: marketplace.Protocol, ExpiresAt: now.Add(365 * 24 * time.Hour), Releases: []marketplace.Release{}}
	for _, platform := range Platforms {
		name := BinaryName("consize-prometheus-plugin", version, platform)
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		target := strings.Split(platform, "/")
		digest := sha256.Sum256(data)
		catalog.Releases = append(catalog.Releases, marketplace.Release{Manifest: manifest, OS: target[0], Arch: target[1], URL: "https://github.com/" + Repository + "/releases/download/v" + version + "/" + name, SHA256: hex.EncodeToString(digest[:]), Size: int64(len(data))})
	}
	payload, err := json.Marshal(catalog)
	if err != nil {
		return err
	}
	envelope := marketplace.Envelope{Payload: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(key, payload))}
	if _, err := marketplace.Verify(envelope, key.Public().(ed25519.PublicKey), true); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "catalog.json"), envelope); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "catalog.payload.json"), catalog); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "manifest.json"), manifest); err != nil {
		return err
	}
	public := base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "publisher.pub"), []byte(public), 0644); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var sums []string
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), "SHA256SUMS") {
			return fmt.Errorf("unexpected existing release entry: %s", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		sums = append(sums, hex.EncodeToString(digest[:])+"  "+entry.Name()+"\n")
	}
	sort.Strings(sums)
	checksums := []byte(strings.Join(sums, ""))
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), checksums, 0644); err != nil {
		return err
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(key, checksums)) + "\n"
	return os.WriteFile(filepath.Join(dir, "SHA256SUMS.sig"), []byte(signature), 0644)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// Verify authenticates checksums with an independently trusted public key before
// inspecting the signed catalog or trusting any executable in the directory.
func Verify(dir string, public ed25519.PublicKey) error {
	sums, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS.sig"))
	if err != nil {
		return err
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(public) != ed25519.PublicKeySize || !ed25519.Verify(public, sums, signature) {
		return fmt.Errorf("checksum signature rejected")
	}
	for _, line := range strings.Split(strings.TrimSpace(string(sums)), "\n") {
		fields := strings.Split(line, "  ")
		if len(fields) != 2 || filepath.Base(fields[1]) != fields[1] || fields[1] == "." || strings.ContainsAny(fields[1], "/\\") {
			return fmt.Errorf("invalid checksum entry")
		}
		data, err := os.ReadFile(filepath.Join(dir, fields[1]))
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		if fields[0] != hex.EncodeToString(digest[:]) {
			return fmt.Errorf("checksum rejected: %s", fields[1])
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "catalog.json"))
	if err != nil {
		return err
	}
	var envelope marketplace.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	catalog, err := marketplace.Verify(envelope, public, true)
	if err != nil {
		return err
	}
	if len(catalog.Releases) != len(Platforms) {
		return fmt.Errorf("incomplete catalog")
	}
	for _, artifact := range catalog.Releases {
		name := BinaryName("consize-prometheus-plugin", artifact.Manifest.Version, artifact.OS+"/"+artifact.Arch)
		expectedURL := "https://github.com/" + Repository + "/releases/download/v" + artifact.Manifest.Version + "/" + name
		if artifact.URL != expectedURL {
			return fmt.Errorf("unexpected catalog artifact URL")
		}
		binary, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		digest := sha256.Sum256(binary)
		if int64(len(binary)) != artifact.Size || hex.EncodeToString(digest[:]) != artifact.SHA256 {
			return fmt.Errorf("catalog digest rejected: %s", name)
		}
	}
	return nil
}
