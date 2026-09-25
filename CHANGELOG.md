# Changelog

All notable changes to KAssetPack will be documented in this file.

KAssetPack follows [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

### Added

### Changed

### Fixed

---

## [v0.11.3] - 2026-09-25

### Added

- Added the new `NewBuilder()` constructor as the recommended way to create an `AssetBuilder`.
- Added a compile-time check to ensure `AssetBuilder` implements `AssetBuilderInterface`.

### Fixed

- Fixed `AssetBuilderInterface.AppendAsset` to support the optional alias and match the `AssetBuilder` implementation.

### Deprecated

- Deprecated `NewAssetBuilder()` in favor of `NewBuilder()`. It remains available for backward compatibility.

### Changed

- Updated the CLI, tests, and documentation to use `NewBuilder()`.

---

## [0.11.2] - 2026-09-23

### Fixed

- Fix release asset paths to use the `build/` directory.
- Prevent releases from being published when expected binaries are missing.
- Add release artifact listing to simplify CI diagnostics.

### Changed

- Configure the release workflow to explicitly upload:
  - `kassetpack_windows_amd64.exe`
  - `kassetpack_linux_amd64`
  - `kassetpack_darwin_amd64`
  - `kassetpack_darwin_arm64`

### Backward Compatibility

This release does not introduce any changes to the KAssetPack public API.

No breaking changes are introduced in v0.11.2.


## [0.11.1] - 2026-09-23

### Fixed

- Fix GitHub Release asset publishing with immutable releases.
- Prevent unintended files such as `kassetpack_test.go` from being included as release assets.
- Update `softprops/action-gh-release` to v3 for Node.js 24 support.


## [0.11.0] - 2026-09-23

### Added

- Add standalone KAssetPack CLI.
- Add `build` command for creating asset packs from the command line.
- Add `unpack` command for extracting existing asset packs.
- Add `AddFolder` method for recursively adding files from a directory.
- Add extension filtering when scanning directories.
- Add logical path rebasing when adding folders.
- Add support for custom output directories in the CLI.
- Add support for configuring the maximum `.kdt` file size from the CLI.
- Add XOR obfuscation key support to CLI commands.
- Add global `-help` flag.
- Add global `-version` flag.
- Add Makefile for common development, testing, build, and release tasks.
- Add multi-platform CLI release builds for:
  - Windows amd64
  - Linux amd64
  - macOS amd64
  - macOS arm64
- Add GitHub issue templates.
- Add GitHub pull request template.

### Changed

- Improve README with CLI documentation and usage examples.
- Improve `AddFolder` API documentation.
- Improve logical path and alias documentation.
- Run CI on the `develop` branch.
- Improve GitHub Actions test workflow.
- Improve GitHub release workflow.

### Backward Compatibility

This release is backward compatible with the previous public API.

No breaking changes are introduced in v0.11.0.

---

## [0.10.0] - 2026-09-17

### Added

- Add asset pack builder.
- Add runtime `AssetBank`.
- Add `.kdx` binary index format.
- Add `.kdt` asset data format.
- Add support for splitting asset data across multiple `.kdt` files.
- Add configurable maximum data file size.
- Add SHA-256 content deduplication.
- Add XOR-based asset obfuscation.
- Add XOR obfuscation for the pack index.
- Add streaming asset writing to avoid loading complete assets into memory.
- Add `AppendAsset` for adding files to an asset pack.
- Add `Save` for writing the asset pack index.
- Add `Load` for loading an existing asset pack.
- Add `Read` for reading an asset completely into memory.
- Add `Open` for streaming asset data through `io.ReadCloser`.
- Add `Close` for releasing resources used by the asset bank.
- Add support for logical asset paths and aliases.

### Architecture

KAssetPack is designed as a lightweight asset packaging layer.

The library intentionally focuses on:

- Asset packaging
- Asset indexing
- Streaming access
- Content deduplication
- Multi-part data files
- Lightweight asset obfuscation

KAssetPack does not aim to provide:

- Asset management
- Asset lifecycle management
- Image or audio conversion
- Runtime asset transformation
- Compression pipelines
- Virtual file systems
- Engine-specific integrations

### Security

XOR is used for lightweight obfuscation only.

It is **not encryption** and should not be considered a security mechanism.