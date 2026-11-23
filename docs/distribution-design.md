# Plugin distribution design and upstream evidence

Reviewed on 2026-11-23 against the primary repositories and official documentation below. These are source patterns; the Consize wire protocol and signature scheme come from Consize, not from Kong, Datadog or Terraform.

## Kong: executable, configuration factory, metadata

[Kong's Go plugin guide](https://developer.konghq.com/custom-plugins/go/) describes a standalone executable using a configuration struct, a constructor, the shared PDK server and an explicit version. Its executable supports metadata inspection. We use the existing Consize `ServeOnce` server and add `--manifest`/`--version`, with a strict JSON configuration factory. We do not copy Kong's socket protocol or plugin priority model.

The [Kong Go PDK repository](https://github.com/Kong/go-pdk) separates the shared PDK from plugin examples. Its [.goreleaser.yaml](https://github.com/Kong/go-pdk/blob/master/.goreleaser.yaml) skips executable builds because the PDK itself is a library: that file is not evidence that a plugin binary is packaged by the PDK release.

## Datadog: metadata and user-facing installation documentation

The concrete [Prometheus integration manifest](https://github.com/DataDog/integrations-core/blob/master/prometheus/manifest.json) records integration identity, owner, configuration, support and documentation. Its [pyproject.toml](https://github.com/DataDog/integrations-core/blob/master/prometheus/pyproject.toml) separately describes Python packaging. Our `plugin.json`, generated configuration schema, setup examples and support documents serve the metadata/documentation roles without pretending a Python wheel is suitable for Consize.

[Datadog integrations-extras](https://github.com/DataDog/integrations-extras) explicitly says those integrations are not included with the Agent and are not currently packaged. A searchable source listing alone therefore is not treated as a download/install pipeline here. The [official development guide](https://docs.datadoghq.com/extend/integrations/agent_integration/) documents scaffolding, configuration, tests and review.

## Terraform: independent releases, platforms, integrity and authenticity

The [official provider publishing requirements](https://developer.hashicorp.com/terraform/registry/providers/publishing) describe semantic version tags, platform-specific artifacts, metadata, checksums, signatures and immutable released versions. The [scaffolding release workflow](https://github.com/hashicorp/terraform-provider-scaffolding-framework/blob/main/.github/workflows/release.yml) uses tag-triggered GitHub Actions, with signing material stored as a secret. Its [.goreleaser.yml](https://github.com/hashicorp/terraform-provider-scaffolding-framework/blob/main/.goreleaser.yml) builds with CGO disabled and trimpath, injects release version, and creates platform artifacts and signed checksums.

We adopt those distribution properties. Consize's existing installer consumes raw executables and verifies Ed25519-signed catalogs, so we use raw platform binaries and Ed25519 rather than Terraform ZIP/GPG formats. The publishing guide permits manual preparation as well as GoReleaser. A small Go release tool implements that preparation with the standard library, then tests its outputs using the actual core verifier. This avoids introducing a second package format the core cannot install.

## Consize-specific contract verified in source

Pinned [catalog and installer](https://github.com/consize-oss/consize/blob/e196f1e160920f7cdec394665d051251311bad08/pkg/plugin/marketplace/catalog.go) require protocol 1, HTTPS artifact URLs, supported categories, exact platform/version, SHA256, size, expiry and Ed25519 authentication. Installation produces a signed receipt, does not execute code, refuses overwrite, and rejects symlink install roots.

Pinned [subprocess runtime](https://github.com/consize-oss/consize/blob/e196f1e160920f7cdec394665d051251311bad08/pkg/plugin/marketplace/runtime.go) verifies installed bytes again before each invocation, negotiates the manifest and metrics capability, strips ambient credential variables, and imposes a 30-second deadline. Its Unix locking dependency is why Windows is excluded. [Bootstrap](https://github.com/consize-oss/consize/blob/e196f1e160920f7cdec394665d051251311bad08/internal/bootstrap/bootstrap.go) accepts explicit `external_plugins` configuration; enabling the built-in and external metrics plugin together would duplicate the ID.

The core [Plugins UI](https://github.com/consize-oss/consize/blob/e196f1e160920f7cdec394665d051251311bad08/ui/src/features/plugins/PluginsPage.tsx) explicitly says the marketplace is unavailable. This release supplies GitHub discovery, metadata and a usable signed catalog; it does not claim registration in an unimplemented global marketplace or change core UI behavior.

## Verification gates

CI tests the extracted implementation, configuration, actual core loader and signed receipt, signature and artifact tampering, real Prometheus query evaluation, and four-target packaging. Release publication happens only after those checks and full local asset verification. The public release is then downloaded again, its signatures and every checksum verified, and its native plugin installed through the unmodified core installer and handshake loader.

Linux amd64 and macOS arm64 run native tests. Linux arm64 and macOS amd64 are cross-built and individually hashed; no native test claim is made for them. Catalogs expire after one year, and renewals require a new version; installed pinned versions remain usable for recovery under the core contract.
