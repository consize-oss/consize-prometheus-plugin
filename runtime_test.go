package prometheus_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/consize-oss/consize-prometheus-plugin/prometheus"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/plugin/marketplace"
	"github.com/consize-oss/consize/pkg/resource"
)

// Exercise the real core loader, signature verification, handshake and subprocess metrics.
func TestMarketplaceRuntime(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "plugin")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/consize-prometheus-plugin")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/-/ready" {
			w.Write([]byte("ready"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/query" {
			json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"resultType": "scalar", "result": []any{time.Now().Unix(), "1"}}})
			return
		}
		if r.URL.Path != "/api/v1/query_range" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		from, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("start"))
		if err != nil {
			t.Error(err)
		}
		to, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("end"))
		if err != nil {
			t.Error(err)
		}
		start, end := float64(from.UnixMilli())/1000, float64(to.UnixMilli())/1000
		value := "3"
		if strings.Contains(r.URL.Query().Get("query"), "kube_pod_container_info") {
			value = "1"
		}
		if strings.Contains(r.URL.Query().Get("query"), "container_cpu_cfs") {
			value = "0.1"
		}
		values := []any{[]any{start, value}}
		if end > start {
			values = append(values, []any{end, value})
		}
		labels := map[string]string{"cluster": "test-cluster", "namespace": "default", "pod": "checkout-pod", "uid": "pod-uid", "container": "app", "deployment": "checkout", "deployment_uid": "deployment-uid"}
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"resultType": "matrix", "result": []any{map[string]any{"metric": labels, "values": values}}}})
	}))
	defer server.Close()
	config, _ := json.Marshal(map[string]string{"base_url": server.URL, "step": "1m"})
	p, err := prometheus.FromJSON(config)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(marketplace.Catalog{Protocol: 1, ExpiresAt: time.Now().Add(time.Hour), Releases: []marketplace.Release{{Manifest: p.Manifest(), OS: runtime.GOOS, Arch: runtime.GOARCH, URL: "https://example.com/plugin", SHA256: hex.EncodeToString(digest[:]), Size: int64(len(data))}}})
	if err != nil {
		t.Fatal(err)
	}
	envelope := marketplace.Envelope{Payload: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, payload))}
	receipt, _ := json.Marshal(marketplace.Receipt{Envelope: envelope, ID: p.ID(), Version: p.Manifest().Version, OS: runtime.GOOS, Arch: runtime.GOARCH})
	if err := os.WriteFile(filepath.Join(dir, "receipt.json"), receipt, 0600); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(dir, "public.key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(public)), 0600); err != nil {
		t.Fatal(err)
	}
	manager := plugin.NewManager()
	if err := marketplace.Register(manager, marketplace.ExternalConfig{Directory: dir, PublicKeyPath: keyPath, Config: config}); err != nil {
		t.Fatal(err)
	}
	registered, err := manager.MetricsPlugin(p.ID(), resource.TypeKubernetesDeployment)
	if err != nil {
		t.Fatal(err)
	}
	metrics := registered.(plugin.WindowedMetricsPlugin)
	snapshot, err := metrics.ReadMetricsBetween(context.Background(), resource.Resource{ID: "test", Type: resource.TypeKubernetesDeployment, Name: "checkout", Provider: resource.ProviderKubernetes, Account: "test-cluster", Metadata: map[string]any{"namespace": "default", "uid": "deployment-uid", "mapping": map[string]any{"version": "kubernetes-deployment-mapper-v1", "containers": []any{map[string]any{"name": "app", "kind": "container"}}}}}, time.Unix(1700000000, 0), time.Unix(1700000060, 0))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Signals["cpu_cores_p95"].(json.Number).String() != "3" || snapshot.Coverage["cpu_cores"].ActualPoints != 2 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if health := metrics.Health(context.Background()); health.Status != "healthy" {
		t.Fatalf("health: %+v", health)
	}
	if err := os.WriteFile(binary, append(data, '!'), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := marketplace.Load(marketplace.ExternalConfig{Directory: dir, PublicKeyPath: keyPath, Config: config}); err == nil {
		t.Fatal("tampered installed executable was accepted")
	}
}
