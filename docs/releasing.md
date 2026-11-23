# Release operations

The publisher is `consize-oss/consize-prometheus-plugin`. Releases are independent of core versioning. The initial release is `v0.1.0`; it is experimental and compatible with the exact core revision in `plugin.json` and `go.mod`, not Consize v0.2.0.

## Signing identity

`keys/publisher.pub` is the base64 Ed25519 public key. The private counterpart belongs only in the repository Actions secret `CONSIZE_CATALOG_PRIVATE_KEY`, never in Git, release assets, logs or developer examples. Release packaging verifies that the private key matches the committed public key before signing.

Public key fingerprint: `89ef8cbd7b7878e0efb771a0c598e69febb09c044563a5fddbfcda4996847667`.

Fingerprint algorithm: SHA256 over the decoded 32-byte Ed25519 public key, expressed in hexadecimal. Pin this reviewed key when trusting the publisher. The first download of a key is a trust-establishment step, not proof by itself.

## Publish a new version

1. Update `prometheus.Version` for local builds, changelog, release notes and versioned installation examples. Update the pinned core dependency only with compatibility tests.
2. Regenerate schema using `go run ./cmd/configschema` and pass CI on `main`.
3. Create and push a new semantic version tag such as `v0.1.1` on tested `main`. The Release workflow tests again, builds Linux/macOS amd64/arm64 plugin and installer binaries with CGO disabled and trimpath, injects the tag version, and creates signed distribution metadata.
4. The workflow verifies all assets locally, creates a draft release containing every asset, then publishes it. A post-publication step downloads all assets and verifies them again. It also discovers, installs and handshakes the public native binary through the core CLI/runtime.
5. Confirm the Release run is green and versioned/catalog URLs work without authentication. The stable discovery URL is `/releases/latest/download/catalog.json`; install always selects an exact plugin version.

Never replace a published executable or reuse a version/tag. If a release is wrong, issue a new version. A failure before publication may leave a draft: inspect the run and remove the unpublished draft before a controlled rerun. A post-publication check failure needs investigation, not silent asset replacement.

Catalogs expire 365 days after signing. Publish a fresh version before expiry to keep new installs available; do not edit an old release's signed bytes. Core intentionally permits already-installed pinned versions after catalog expiry for recovery.

## Local release verification

For release-tool development, use an ephemeral test key outside the repository:

```sh
go run ./cmd/releasectl keygen -private-key /private/tmp/test-release.key -public-key /private/tmp/test-release.pub
go run ./cmd/releasectl package -version 0.0.0-test -dir dist-test -private-key /private/tmp/test-release.key -public-key /private/tmp/test-release.pub
go run ./cmd/releasectl verify -dir dist-test -public-key /private/tmp/test-release.pub
```

The output directory must not already exist. The signed catalog contains only plugin artifacts; the signed checksum manifest also covers the installer binaries and distribution documentation. Signature verification uses the original bytes, without JSON reserialization.

## Key rotation

Do not regenerate the key for each release. To rotate after review, commit a new public key, update the repository secret and fingerprint documentation together, and publish a new version with explicit migration instructions. Existing installations keep their original trust key and signed receipts. Preserve historical public keys so installed releases remain verifiable. If a key is compromised, publish a security advisory and replacement trust instructions; the current core has no central key-revocation service.
