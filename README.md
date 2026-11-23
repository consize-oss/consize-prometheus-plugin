# Consize Prometheus plugin

[![CI](https://github.com/consize-oss/consize-prometheus-plugin/actions/workflows/ci.yml/badge.svg)](https://github.com/consize-oss/consize-prometheus-plugin/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/consize-oss/consize-prometheus-plugin)](https://github.com/consize-oss/consize-prometheus-plugin/releases)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

Read-only Prometheus metrics and evidence collection for Consize's protocol 1 external-plugin runtime. Independently versioned and distributed by **consize-oss**.

Collects Kubernetes Deployment CPU, memory, throttling, restarts, OOMs, evictions, and current resource requests/limits. Validates workload ownership before aggregation and preserves missing, stale, partial and unavailable evidence.

**Compatibility:** experimental plugin release `0.1.0`, tested against Consize `main` at [`e196f1e16092`](https://github.com/consize-oss/consize/commit/e196f1e160920f7cdec394665d051251311bad08). Consize v0.2.0 does not have this external-plugin runtime. The UI marketplace is not available in the current core; discovery and installation use the signed catalog and CLI below.

## Discover and download

- [Versioned binary releases](https://github.com/consize-oss/consize-prometheus-plugin/releases): Linux and macOS, amd64 and arm64.
- [Signed catalog](https://github.com/consize-oss/consize-prometheus-plugin/releases/latest/download/catalog.json): machine-readable platform, version, download URL, SHA256 and size.
- [Integration listing](plugin.json): identity, owner, source, support, compatibility, catalog and documentation links.
- [Configuration schema](config.schema.json) and [publisher public key](keys/publisher.pub).

Each release includes the standalone plugin, `consize-pluginctl`, manifest, catalog, public key, configuration schema, license, and Ed25519-signed `SHA256SUMS`. The catalog signs the exact binary hashes the core installer verifies. Assets are raw executables because the core installer does not extract archives. Windows is not advertised: the existing core runtime uses Unix file locking.

## Install

Use a compatible Consize core and an explicitly trusted publisher key. The public key's SHA256 fingerprint is recorded in [release operations](docs/releasing.md); establish trust from a reviewed source revision before installing code. Fetching a key and binary from the same untrusted endpoint is not independent verification.

With Go installed, use the version-pinned installer:

```sh
go install github.com/consize-oss/consize-prometheus-plugin/cmd/consize-pluginctl@v0.1.0
curl --fail --location --proto '=https' --proto-redir '=https' \
  https://raw.githubusercontent.com/consize-oss/consize-prometheus-plugin/v0.1.0/keys/publisher.pub \
  --output publisher.pub

consize-pluginctl catalog \
  -catalog https://github.com/consize-oss/consize-prometheus-plugin/releases/download/v0.1.0/catalog.json \
  -public-key publisher.pub

consize-pluginctl install \
  -catalog https://github.com/consize-oss/consize-prometheus-plugin/releases/download/v0.1.0/catalog.json \
  -public-key publisher.pub \
  -id prometheus-metrics -version 0.1.0 \
  -dir /absolute/path/to/consize-plugins
```

`-dir` must be an administrator-owned absolute path without symlink components. On macOS use a physical path (`pwd -P`); `/tmp` is a symlink on macOS. Installation verifies the catalog signature, expiry, exact platform, size and SHA256. It downloads without executing or enabling the plugin.

To use the prebuilt installer instead, download `consize-pluginctl_0.1.0_<os>_<arch>` from the release, verify its entry in the signed checksums using a trusted verifier, and mark it executable. Source users can verify a complete downloaded release with `go run ./cmd/releasectl verify -dir /path/to/downloads -public-key publisher.pub`.

## Enable in Consize

Merge [examples/external-plugin.config.json](examples/external-plugin.config.json) into your compatible core configuration, replacing paths and the endpoint. Disable the built-in Prometheus plugin to avoid registering the same ID twice:

```json
{
  "prometheus": {"enabled": false},
  "external_plugins": [{
    "directory": "/absolute/path/to/consize-plugins/prometheus-metrics-0.1.0-linux-amd64",
    "public_key_path": "/absolute/path/to/publisher.pub",
    "config": {
      "base_url": "https://prometheus.example.com",
      "window": "24h",
      "step": "5m",
      "transport": {
        "auth": {"bearer_token_file": "/run/secrets/prometheus-token"}
      }
    }
  }]
}
```

Use the directory reported by `install`, which includes your actual OS and architecture. Set `metrics_plugin_id` to `prometheus-metrics` in the recommender and verification configuration; set collection `plugin_id` to the same ID when continuous collection is enabled. The host must be able to read the files referenced by configuration.

Validate the installed artifact and handshake without contacting Prometheus:

```sh
consize-pluginctl check \
  -dir /absolute/path/to/consize-plugins/prometheus-metrics-0.1.0-linux-amd64 \
  -public-key publisher.pub
```

## Configuration and telemetry prerequisites

`base_url` is required. `window` defaults to `24h`, `step` to `5m`; both must be positive, and step must be whole seconds and no larger than window. The [generated JSON Schema](config.schema.json) describes available fields. Unknown fields are rejected. The implementation additionally validates endpoints, mapping, bounds and duration semantics.

Optional settings include `percentile` (default 95), `minimum_samples` (default 2), `queries`, `units`, `mapping`, and bounded HTTP `transport`. File-based bearer tokens, custom CA certificates and client certificate/key pairs are supported. Ambient environment credential references are rejected because Consize strips them from subprocesses. Basic authentication and environment-based headers are not exposed by this external plugin. Credentials are never placed in URLs or emitted in diagnostic errors.

Ownership labels are required. Default `direct` mode needs cluster, namespace, pod, pod UID, container, Deployment name and Deployment UID labels on measurements and container inventory. `owner_chain` mode uses Kubernetes owner-reference metrics and still requires trusted Deployment UID enrichment. Stock cAdvisor and kube-state-metrics alone are not claimed sufficient. Resources need validated Kubernetes identity and container inventory. Missing or ambiguous ownership produces unavailable evidence, not inferred attribution. See the pinned core [mapping contract](https://github.com/consize-oss/consize/blob/e196f1e160920f7cdec394665d051251311bad08/docs/reference/prometheus-mapping.md), [query/evidence contract](https://github.com/consize-oss/consize/blob/e196f1e160920f7cdec394665d051251311bad08/docs/reference/prometheus-evidence.md), and [transport reference](https://github.com/consize-oss/consize/blob/e196f1e160920f7cdec394665d051251311bad08/docs/reference/prometheus-client.md).

The core imposes a 30-second total deadline per subprocess request. Individual request timeout/retry settings do not extend that budget; choose a window, step and endpoint that complete collection within it. The checked-in tests exercise real Prometheus as well as fixture responses.

## Development

A fresh clone builds without a sibling core checkout or `go.work`. Dependencies and checksums are pinned in `go.mod` and `go.sum`; CI and release builds use the toolchain in `.go-version`.

```sh
go mod download
go test -race ./...
go vet ./...
go build -o bin/consize-prometheus-plugin ./cmd/consize-prometheus-plugin
bin/consize-prometheus-plugin --manifest
bin/consize-prometheus-plugin < examples/handshake.json
go run ./cmd/configschema -check
```

For the full real-server tests, run `bash scripts/ci/install-prometheus-test.sh /absolute/test/tool/path`, then export the printed `CONSIZE_PROMETHEUS_BIN` value before testing. CI requires these tests on Linux amd64 and macOS arm64 and cross-builds all four supported release targets. Other target binaries are cross-compiled, not claimed natively tested.

The executable serves one JSON request on stdin and one JSON response on stdout, then exits. Protocol handling uses Consize's `marketplace.ServeOnce`; `--manifest`, `--version` and `--help` do not need configuration or network access. Logs belong on stderr. Supported RPC methods are `handshake`, `health`, `metrics` and `metrics_between`.

## Maintenance and support

See [release operations](docs/releasing.md), [upstream implementation research and design choices](docs/distribution-design.md), [changelog](CHANGELOG.md), [contributing](CONTRIBUTING.md), and [security reporting](SECURITY.md). Report bugs through [GitHub Issues](https://github.com/consize-oss/consize-prometheus-plugin/issues).

Apache-2.0. The initial collection implementation and associated tests were extracted from Consize at `e196f1e160920f7cdec394665d051251311bad08`; core integration tests that depend on private `internal` packages remain in the core repository.
