# Debian Package Toolkit Architecture

This document summarizes the responsibilities and data flow for the core files in pkg/debian. Each section focuses on what the module owns, how it is used by others, and the key behaviors that matter at runtime. Each module now has its own textual schematic for quick scanning.

## pkg/debian/package.go — Data model and shared constants
- Core data contracts: `Package` (binary metadata), `SourcePackage`/`SourceFile` (source metadata and pool paths), `DownloadInfo` (HTTP metadata).
- Shared constants: `DirPermission`, `FilePermission`, and `CompressionExtensions` reused across repository, downloader, and mirror for consistent filesystem and archive handling.
- Field coverage: identification, dependencies, checksums, sections, multi-arch, maintainer scripts, and `CustomFields` to preserve unknown/X-prefixed fields when parsing control/Packages data.
- Responsibilities: provide the structs that parsing code in repository.go populates and that downloader/mirror consume for filenames, checksums, and pool layout decisions.

Schematic (data contracts)
```
Package structs
        |-- Package (binary) -> filenames, checksums, deps, sections
        |-- SourcePackage -> pool directory + []SourceFile
        |-- SourceFile -> name, URL, sizes, hash, type (orig/debian/dsc)
Shared constants
        |-- DirPermission / FilePermission
        |-- CompressionExtensions
Used by
        |-- repository.go (parsing)
        |-- downloader.go (filenames, checksums)
        |-- mirror.go (pool layout)
```

## pkg/debian/repository.go — Metadata fetch and dependency resolution
- Repository lifecycle: `NewRepository` wires suite/component/arch, signature verification, and keyrings; `FetchReleaseFile` downloads Release/InRelease with optional signature checks; `FetchPackages` pulls Packages indices per section/arch.
- GPG Verification: `verifyWithGPG` handles signature validation using `gpgv`. It supports cross-platform execution by detecting the OS (`runtime.GOOS`) to locate keyrings (Linux defaults, Windows Gpg4win/AppData, macOS Homebrew) and the `gpgv` executable.
- Parsing: handles gzip/xz compressed Packages files, Release checksum sections, and RFC822-style package stanzas via shared field mapping to `Package` structs.
- Dependency resolution: `ResolveDependencies` performs apt-like traversal with configurable exclusions (depends, pre-depends, recommends, suggests, enhances) and selects available alternatives within the fetched metadata.
- URL/build helpers: constructs Release/Packages URLs, pools architecture/section context, and exposes getters (`GetPackageMetadata`, `GetAllPackageMetadata`) for downstream consumers like mirror and CLI commands.

Schematic (metadata + deps)
```
Input: base URL, suites, components, architectures
        -> build Release URL / InRelease URL
        -> fetch + optional signature verify
        -> fetch Packages per (section, arch), decompress (gz/xz), parse RFC822
        -> map fields -> Package structs

ResolveDependencies
        queue PackageSpec -> find Package -> collect deps
                | filter by exclude set (depends, pre-depends, recommends, suggests, enhances)
                | pick first available alternative in OR expressions
        enqueue remaining until queue empty

Outputs: Release info, Package metadata list, dependency-closed package set
Consumers: CLI commands, mirror, integration tests
```

## pkg/debian/downloader.go — HTTP, retries, and integrity
- HTTP pipeline: `Downloader` encapsulates UA, timeouts, retry/backoff (3 attempts, 2s delay), and optional progress callbacks; concurrency defaults to 5 for multi-downloads.
- Rate limiting: `RateDelay` field enables sequential downloads with configurable delay between requests; useful for legacy repositories that cannot handle high request rates.
- Integrity: verifies checksums (prefers SHA256, falls back to MD5), and can short-circuit downloads when local files match expected hashes.
- APIs: `DownloadToDir` (single artifact), `DownloadMultiple` (batched with worker pool), plus internal helpers for filename generation and retry logic; uses shared permissions from package.go.
- Consumers: called by CLI commands (binary/source/custom repo), repository/mirror flows, and integration tests that validate real HTTP fetches.

Schematic (download path)
```
DownloadToDir(pkg)
        -> build dest path (use pkg.Filename or synthesized)
        -> ShouldSkipDownload? (hash matches) yes => return
        -> retry loop (3 attempts, 2s delay)
                         http GET with UA + timeout
                         if 200 OK -> stream to file (option progress)
                         else retry
        -> verify checksum (prefer SHA256 else MD5)

DownloadMultiple(pkgs, workers=5)
        -> if RateDelay > 0: force workers=1, sleep between downloads
        -> worker pool executes DownloadToDir in parallel (or sequential with delay)
```

## pkg/debian/mirror.go — Mirror orchestration
- Configuration: `MirrorConfig` validates base URL, suites, components, arches, and toggles for package downloads, verbosity, GPG verification, and rate limiting for legacy repos.
- Orchestration: `Mirror` composes a `Repository` and `Downloader` to fetch metadata, materialize Release/Packages files locally, and optionally download `.deb` files into Debian pool layout (dists/ and pool/ with prefix rules).
- Rate limiting: `RateDelay` propagates to the downloader to throttle requests when mirroring legacy repositories that cannot handle high concurrency.
- Operations: `Clone` builds a full mirror; `Sync` currently reuses Clone as a placeholder for future incremental logic. Helper methods compute suite/component paths, regenerate Release checksum sections, and emit verbose logs when requested.

Schematic (mirror flow)
```
for each suite/component/arch:
        repository.SetDistribution(suite)
        fetch Release (+sig if enabled)
        fetch Packages (gz/xz) and parse into Package structs
        write Release/Packages under dists/
        if DownloadPackages:
                 resolve packages -> download .deb into pool/ with prefix rules
regenerate Release sections with checksums

Layout:
        dists/<suite>/<component>/binary-<arch>/Packages[.gz|.xz]
        pool/<prefix>/<package>/<package>_<version>_<arch>.deb
```
