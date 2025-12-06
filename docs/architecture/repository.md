# Repository Architecture - `pkg/debian/repository.go`

## Overview

The `repository.go` file handles all interactions with Debian repositories, including fetching and parsing Release files, Packages files, and package metadata. It supports multiple compression formats and checksum verification.

## Constants

Local constants for repository operations:

```go
const (
    packagesBufferSize   = 1024 * 1024  // 1MB buffer for Packages file parsing
    packagesInitialAlloc = 64 * 1024    // Initial allocation for scanner buffer
)

var defaultSections = []string{"main", "contrib", "non-free"}
```

Also uses from `package.go`:
- `CompressionExtensions` for trying different Packages file formats in `fetchPackagesForSectionArch()`

## Data Structures

### Repository

The main struct for repository interactions.

```
Repository
├── Configuration
│   ├── Name, URL, Description
│   ├── Distribution (suite)
│   ├── Sections []string (components)
│   └── Architectures []string
├── State
│   ├── Packages []string (package names)
│   ├── PackageMetadata []Package
│   ├── ReleaseInfo *ReleaseFile
│   └── VerifyRelease bool
```

### ReleaseFile

Represents the parsed Release file from a Debian repository.

```
ReleaseFile
├── Metadata
│   ├── Origin, Label, Suite, Version
│   ├── Codename, Date, Description
│   ├── Architectures []string
│   └── Components []string
└── Checksums
    ├── MD5Sum []FileChecksum
    ├── SHA1 []FileChecksum
    └── SHA256 []FileChecksum
```

### FileChecksum

```
FileChecksum
├── Hash string
├── Size int64
└── Filename string
```

### PackageInfo

Lightweight package information struct for search results.

```
PackageInfo
├── Name, Version, Architecture
├── Section, DownloadURL
└── Size int64
```

## Public API

### Constructor

| Function | Description |
|----------|-------------|
| `NewRepository(name, url, description, distribution, sections, architectures)` | Creates a new Repository instance |

### Package Fetching

| Method | Description |
|--------|-------------|
| `FetchPackages()` | Fetches and parses Packages files, returns package names |
| `SearchPackage(name)` | Searches for packages by name (exact + partial matches) |
| `GetPackageMetadata(name)` | Returns complete Package struct for a package |
| `GetAllPackageMetadata()` | Returns all parsed package metadata |

### Release File Operations

| Method | Description |
|--------|-------------|
| `FetchReleaseFile()` | Downloads and parses the Release file |
| `GetReleaseInfo()` | Returns the parsed ReleaseFile |
| `EnableReleaseVerification()` | Enables checksum verification |
| `DisableReleaseVerification()` | Disables checksum verification |
| `IsReleaseVerificationEnabled()` | Returns verification status |

### Package Download

| Method | Description |
|--------|-------------|
| `DownloadPackage(name, version, arch, destDir)` | Downloads a package by name |
| `DownloadPackageByURL(url, destDir)` | Downloads a package by direct URL |
| `DownloadPackageFromSources(name, version, arch, destDir, sections)` | Downloads trying multiple sections |
| `CheckPackageAvailability(name, version, arch)` | Checks if a package exists |
| `SearchPackageInSources(name, version, arch)` | Returns PackageInfo if found |

### Configuration

| Method | Description |
|--------|-------------|
| `SetDistribution(distribution)` | Sets the active suite |
| `SetSections(sections)` | Sets the active components |
| `SetArchitectures(architectures)` | Sets the active architectures |
| `AddSection(section)` | Adds a component |
| `AddArchitecture(architecture)` | Adds an architecture |

### Checksum Verification

| Method | Description |
|--------|-------------|
| `VerifyPackagesFileChecksum(section, arch, data)` | Verifies Packages file checksum (prefers SHA256 over MD5) |
| `verifyDataChecksum(data, hash, type)` | Internal checksum verification |

## Helper Functions

Internal helper functions for code organization and reduced duplication:

| Function | Description |
|----------|-------------|
| `checkURLExists(url)` | Performs HEAD request to verify URL accessibility |
| `getDecompressor(reader, ext)` | Returns appropriate decompressor based on file extension |
| `buildPackageStruct(name, version, arch, url)` | Constructs a Package struct with computed filename |
| `getPoolPrefix(packageName)` | Returns pool directory prefix (4 chars for lib*, 1 char otherwise) |
| `parsePackageField(pkg, field, value)` | Parses a single package field using field mapping from package.go |
| `fetchPackagesForSectionArch(section, arch)` | Fetches Packages file for a specific section/architecture |
| `buildReleaseURL()` | Constructs the URL for the Release file |
| `parseChecksumLine(line)` | Parses a checksum line from Release file |

## URL Building

```
Repository URL Structure:
├── Release File
│   └── {baseURL}/dists/{distribution}/Release
├── Packages File
│   └── {baseURL}/dists/{distribution}/{section}/binary-{arch}/Packages[.gz|.xz]
└── Package Download
    └── {baseURL}/pool/{section}/{prefix}/{source_name}/{filename}

Note: lib* packages use first 4 characters as prefix (e.g., liba, libb)
      Other packages use first character only
```

## Compression Support

The module automatically tries multiple compression formats via `getDecompressor()`:

| Extension | Library | Handler |
|-----------|---------|---------|
| (none) | Direct reading | Returns reader as-is |
| `.gz` | `compress/gzip` | `gzip.NewReader()` |
| `.xz` | `github.com/ulikunitz/xz` | `xz.NewReader()` |

## Packages File Parsing

The parser handles the RFC 822-like format:

1. Uses a 1MB buffer (`packagesBufferSize`) for reading large files
2. Empty lines separate package blocks
3. Uses `parsePackageField()` with field mapping from `package.go` for consistent parsing
4. Parses 40+ field types including:
   - Core: Package, Version, Architecture, Maintainer
   - Dependencies: Depends, Recommends, Suggests, etc.
   - Metadata: Section, Priority, Homepage, etc.
   - Checksums: MD5sum, SHA1, SHA256
5. Stores in `PackageMetadata` slice
6. Constructs download URLs from Filename field

## Release File Parsing

1. Downloads from `dists/{suite}/Release` via `buildReleaseURL()`
2. Parses header fields (Origin, Suite, Codename, etc.)
3. Parses checksum sections (MD5Sum, SHA1, SHA256) using `parseChecksumLine()`
4. Stores checksums for file verification

## Dependencies

- `bufio`, `strings`: Text parsing
- `compress/gzip`: Gzip decompression
- `github.com/ulikunitz/xz`: XZ decompression
- `crypto/md5`, `crypto/sha256`: Checksum verification
- `net/http`: HTTP requests (GET and HEAD)
- `io`, `strconv`: I/O and parsing
- `hash`: Hash interface for checksum verification

## Integration Points

- **package.go**: Uses Package struct and field mapping functions for metadata parsing
- **downloader.go**: Creates Downloader for package downloads
- **mirror.go**: Uses Repository for metadata fetching

## Error Handling

- Wraps errors with context using `fmt.Errorf()` with `%w` verb for error chaining
- Falls back to next compression format on failure
- Continues processing on individual package parse errors
- Validates checksums when verification is enabled (prefers SHA256 over MD5)
