Initial independently distributed Consize Prometheus plugin.

- Read-only Kubernetes Deployment metrics and provenance, including workload ownership validation, sample coverage and quality classification.
- File-based bearer authentication, TLS and bounded transport configuration.
- Linux/macOS amd64 and arm64 executables, plus `consize-pluginctl`.
- Protocol 1 manifest, Ed25519-signed catalog, signed SHA256 checksums and configuration schema.
- Native Linux amd64/macOS arm64 tests, real Prometheus integration tests, and four-target build verification.

Experimental. Requires the Consize v0.3 development runtime at core revision `e196f1e160920f7cdec394665d051251311bad08`; Consize v0.2.0 is unsupported. Current core has no UI marketplace: use the signed catalog and explicit external-plugin configuration. See the README for telemetry ownership labels and installation.

Trust anchor: `keys/publisher.pub` at the tagged source revision. Versioned assets must be verified against that reviewed key. Installation selects an exact version and does not enable or execute the plugin.
