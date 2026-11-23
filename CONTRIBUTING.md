# Contributing

Open an issue or pull request describing the behavior and evidence behind your change. Run `go test -race ./...`, `go vet ./...` and `go run ./cmd/configschema -check`; use the pinned Prometheus test installer for full integration coverage. CI performs these checks on Linux and macOS and verifies release packaging.

Keep stdout exclusively for protocol responses. Preserve explicit ownership validation and unavailable evidence semantics. Add regression coverage for behavioral changes. Do not put credentials in fixtures or error messages. Changes to the pinned core revision or external protocol need real core-loader compatibility tests.

See [release operations](docs/releasing.md) for distribution changes. Contributions are Apache-2.0 licensed under the repository license.
