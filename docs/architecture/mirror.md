# Mirror Architecture - `pkg/debian/mirror.go`

## Overview

The `mirror.go` file provides high-level orchestration for creating and managing local mirrors of Debian repositories. It combines Repository and Downloader functionality to clone repository metadata and optionally download packages.

## Constants

Local constants for mirror operations:

```go
const (
    defaultAveragePackageSize = 1024 * 1024  // 1MB for size estimation
)
```

Also uses from `package.go`:
- `DirPermission` for creating directories (Clone, mirrorSuite, mirrorArchitecture, etc.)
- `FilePermission` for writing Release files
- `CompressionExtensions` for trying different Packages file formats

## Data Structures

### MirrorConfig

Configuration for mirror operations.

```
MirrorConfig
├── BaseURL string           // Repository URL to mirror from
├── Suites []string          // Distributions to mirror (e.g., bookworm)
├── Components []string      // Components (e.g., main, contrib, non-free)
├── Architectures []string   // Architectures (e.g., amd64, arm64)
├── DownloadPackages bool    // Whether to download .deb files
└── Verbose bool             // Enable verbose logging
```

### Mirror

The main struct for mirror operations.

```
Mirror
├── config MirrorConfig
├── repository *Repository
├── downloader *Downloader
└── basePath string (local mirror root)
```

## Public API

### Constructor

| Function | Description |
|----------|-------------|
| `NewMirror(config, basePath)` | Creates a new Mirror instance |

### Core Operations

| Method | Description |
|--------|-------------|
| `Clone()` | Creates a complete mirror |
| `Sync()` | Performs incremental synchronization (currently equivalent to Clone) |
| `VerifyMirrorIntegrity(suite)` | Verifies mirror checksums |

### Information

| Method | Description |
|--------|-------------|
| `GetMirrorInfo()` | Returns mirror configuration as map |
| `GetMirrorStatus()` | Returns file count, size, status via `calculateMirrorStats()` |
| `EstimateMirrorSize()` | Estimates required disk space using `defaultAveragePackageSize` |
| `GetRepositoryInfo()` | Returns underlying Repository |

### Configuration

| Method | Description |
|--------|-------------|
| `UpdateConfiguration(config)` | Updates mirror configuration with validation |
| `Validate()` | Validates MirrorConfig via `hasValidURLScheme()` |

## Helper Functions

Internal helper functions for code organization and reduced duplication:

| Function | Description |
|----------|-------------|
| `logVerbose(format, args...)` | Conditional logging when verbose mode is enabled |
| `buildSuitePath(suite)` | Constructs path to suite directory |
| `buildArchPath(suite, component, arch)` | Constructs path to architecture directory |
| `buildPackagesBaseURL(suite, component, arch)` | Constructs base URL for Packages files |
| `hasValidURLScheme()` | Validates HTTP/HTTPS URL scheme |
| `tryDownloadPackagesFile(baseURL, dir, ext)` | Attempts download with specific extension |
| `getPackageMetadataOrFallback(name, arch)` | Gets metadata or creates fallback Package |
| `calculateMirrorStats()` | Walks directory to count files and total size |
| `verifyComponentArch(suite, component, arch)` | Verifies integrity of specific component/arch |
| `writeReleaseHeader(content, release)` | Writes header fields to Release file |
| `writeChecksumSection(content, name, checksums)` | Writes a checksum section to Release file |

## Mirror Directory Structure

The module creates a standard Debian repository structure:

```
{basePath}/
├── dists/
│   └── {suite}/
│       ├── Release
│       ├── {component}/
│       │   └── binary-{arch}/
│       │       ├── Packages
│       │       ├── Packages.gz
│       │       └── Packages.xz
│       └── ...
└── pool/
    └── {component}/
        └── {prefix}/           (first letter, or first 4 for lib*)
            └── {source_name}/
                └── {package}_{version}_{arch}.deb
```

## Clone Operation Flow

```
Clone()
│
├─► Create base directory (mirrorDirPermission)
│
└─► For each suite:
    │
    ├─► mirrorSuite(suite)
    │   │
    │   ├─► Create suite directory via buildSuitePath()
    │   │
    │   ├─► downloadReleaseFile(suite)
    │   │   ├─► Fetch Release from repository
    │   │   ├─► buildReleaseFileContent() using helpers:
    │   │   │   ├─► writeReleaseHeader()
    │   │   │   └─► writeChecksumSection() for MD5, SHA1, SHA256
    │   │   └─► Write to dists/{suite}/Release (mirrorFilePermission)
    │   │
    │   └─► For each component:
    │       │
    │       └─► mirrorComponent(suite, component)
    │           │
    │           └─► For each architecture:
    │               │
    │               └─► mirrorArchitecture(suite, component, arch)
    │                   │
    │                   ├─► Create directory via buildArchPath()
    │                   │
    │                   ├─► downloadPackagesFile(suite, component, arch)
    │                   │   ├─► buildPackagesBaseURL()
    │                   │   ├─► Try each extension in packagesExtensions
    │                   │   └─► tryDownloadPackagesFile() for each
    │                   │
    │                   ├─► loadPackageMetadata(suite, component)
    │                   │
    │                   └─► If DownloadPackages:
    │                       └─► downloadPackagesForArch(suite, component, arch)
    │                           │
    │                           └─► For each package:
    │                               └─► downloadPackageByName(...)
```

## Package Download Flow

```
downloadPackageByName(packageName, component, arch)
│
├─► getPackageMetadataOrFallback(packageName, arch)
│   └─► Try repository metadata, fallback to constructed Package
│
├─► Determine source package name via pkg.GetSourceName()
│
├─► Build pool path using getPoolPrefix() from repository.go
│   ├─► Regular: pool/{component}/{first_letter}/{source}/
│   └─► lib*: pool/{component}/{first_4_chars}/{source}/
│
├─► Create package directory (mirrorDirPermission)
│
├─► Construct download URL if not in metadata
│
└─► Download using Downloader
```

## Release File Generation

The module generates a Release file from fetched metadata using helper methods:

```
buildReleaseFileContent(release)
│
├─► writeReleaseHeader(content, release)
│   └─► Origin, Label, Suite, Version, Codename, Date,
│       Description, Architectures, Components
│
└─► writeChecksumSection() for each section:
    ├─► MD5Sum
    ├─► SHA1
    └─► SHA256
```

## Validation

`MirrorConfig.Validate()` checks using `hasValidURLScheme()`:

| Check | Error |
|-------|-------|
| Empty BaseURL | "BaseURL is required" |
| No suites | "at least one suite is required" |
| No components | "at least one component is required" |
| No architectures | "at least one architecture is required" |
| Invalid URL scheme | "BaseURL must start with http:// or https://" |

## Mirror Status

`GetMirrorStatus()` uses `calculateMirrorStats()` and returns:

```go
map[string]any{
    "exists":      bool,
    "initialized": bool,
    "base_path":   string,
    "file_count":  int,
    "total_size":  int64,
}
```

## Size Estimation

`EstimateMirrorSize()` calculates:

- Returns 0 if not downloading packages
- Fetches package list for each suite
- Uses `defaultAveragePackageSize` constant (1MB)
- Returns total estimated bytes

## Dependencies

- **package.go**: Package struct for metadata
- **repository.go**: Repository for fetching metadata, `getPoolPrefix()` for pool paths
- **downloader.go**: Downloader for file downloads
- `os`, `path/filepath`: File operations
- `strings`: String manipulation
- `fmt`: Formatting and error handling

## Integration Points

- Uses Repository internally for all metadata operations
- Uses Downloader internally for all file downloads
- Uses `getPoolPrefix()` from repository.go for consistent pool path calculation
- Exposes Repository via `GetRepositoryInfo()`

## Error Handling

- Wraps errors with `fmt.Errorf("context: %w", err)` for proper error chaining
- Continues on individual package download failures (logs warning via `logVerbose()`)
- Logs progress in verbose mode using `logVerbose()` helper
- Returns first fatal error encountered
