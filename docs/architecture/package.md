# Package Architecture - `pkg/debian/package.go`

## Overview

The `package.go` file defines the core data structures for representing Debian packages (binary and source) and provides utilities for parsing and formatting Debian control files. It also exports shared constants and variables used across the entire `pkg/debian` package. deb-for-all uses these structures for download and mirroring flows only; it does not execute installation or maintainer scripts.

## Exported Constants and Variables

These are exported (capitalized) and used by `downloader.go`, `repository.go`, and `mirror.go`:

```go
const (
    DirPermission  = 0755 // Default directory permission
    FilePermission = 0644 // Default file permission
)

var CompressionExtensions = []string{"", ".gz", ".xz"}  // Supported compression formats
```

## Data Structures

### Package

The `Package` struct is the central abstraction representing a Debian binary package with 50+ fields covering all standard Debian control file fields.

```
Package
├── Identification
│   ├── Name, Package, Version, Architecture
│   └── Maintainer, Description
├── Download Information
│   ├── DownloadURL, Filename, Size
│   └── MD5sum, SHA1, SHA256
├── Classification
│   ├── Source, Section, Priority, Essential
│   ├── InstalledSize, Homepage
│   └── BuiltUsing, PackageType
├── Dependencies
│   ├── Depends, PreDepends, Recommends
│   ├── Suggests, Enhances, Breaks
│   └── Conflicts, Provides, Replaces
├── Metadata
│   ├── Tag, Task, Uploaders, StandardsVersion
│   ├── VcsGit, VcsBrowser, Testsuite
│   ├── AutoBuilt, BuildEssential
│   ├── ImportantDescription, DescriptionMd5
│   └── Gstreamer, PythonVersion
├── Maintainer Scripts
│   └── Preinst, Postinst, Prerm, Postrm
├── Multi-Arch
│   └── MultiArch
├── Origin
│   └── Origin, Bugs
└── Custom Fields
    └── CustomFields map[string]string (X- prefixed or unknown)
```

### SourcePackage

Represents a Debian source package with associated source files.

```
SourcePackage
├── Name, Version, Maintainer, Description
├── Directory (pool path, e.g., pool/main/h/hello)
└── Files []SourceFile
```

### SourceFile

Represents a single file within a source package.

```
SourceFile
├── Name, URL, Size
├── MD5Sum, SHA256Sum
└── Type (orig, debian, dsc)
```

### DownloadInfo

Contains HTTP metadata for package downloads.

```
DownloadInfo
├── URL
├── ContentLength
├── ContentType
└── LastModified
```

## Public API

### Constructors

| Function | Description |
|----------|-------------|
| `NewPackage(...)` | Creates a new Package with required fields |
| `NewSourcePackage(...)` | Creates a new SourcePackage |

### Package Methods

| Method | Description |
|--------|-------------|
| `GetSourceName()` | Returns source package name (fallback to package name) |
| `GetDownloadInfo()` | Fetches HTTP metadata via HEAD request |
| `FormatAsControl()` | Formats package as Debian control file string |
| `WriteControlFile(path)` | Writes control data to file |

### SourcePackage Methods

| Method | Description |
|--------|-------------|
| `AddFile(...)` | Adds a source file to the package |
| `GetOrigTarball()` | Returns the .orig.tar.* file |
| `GetDebianTarball()` | Returns the .debian.tar.* file |
| `GetDSCFile()` | Returns the .dsc file |
| `Download(destDir)` | Downloads all source files with progress |
| `DownloadSilent(destDir)` | Downloads all source files silently |
| `DownloadWithProgress(destDir, callback)` | Downloads with progress callback |
| `String()` | Returns string representation |

### Internal Methods

| Method | Description |
|--------|-------------|
| `findFileByType(type, pattern)` | Searches for file by type or name pattern |
| `downloadFiles(destDir, verbose, callback)` | Internal download implementation |
| `downloadSingleFile(downloader, file, ...)` | Downloads and verifies a single file |

### Utility Functions

| Function | Description |
|----------|-------------|
| `ReadControlFile(path)` | Parses a Debian control file |
| `parseControlData(content)` | Internal parser for control format |
| `parsePackageList(value)` | Parses dependency lists |

## Control File Parsing

The module uses map-based field mapping for efficient parsing:

### controlFieldMapping

Maps control file field names (lowercase) to Package field setters:

```go
var controlFieldMapping = map[string]func(*Package, string){
    "package":           func(p *Package, v string) { p.Package = v; p.Name = v },
    "version":           func(p *Package, v string) { p.Version = v },
    "architecture":      func(p *Package, v string) { p.Architecture = v },
    // ... 30+ field mappings
}
```

### dependencyFieldMapping

Maps dependency field names to Package slice setters:

```go
var dependencyFieldMapping = map[string]func(*Package, []string){
    "depends":     func(p *Package, v []string) { p.Depends = v },
    "pre-depends": func(p *Package, v []string) { p.PreDepends = v },
    // ... 9 dependency mappings
}
```

## Parsing Flow

```
parseControlData(content)
│
├─► Split content into lines
├─► Create Package with empty CustomFields map
│
├─► For each line:
│   ├─► Skip empty lines
│   ├─► Find colon separator
│   ├─► Extract field name and value
│   ├─► Lookup in controlFieldMapping
│   │   └─► If found: call setter
│   ├─► Lookup in dependencyFieldMapping
│   │   └─► If found: parsePackageList() then call setter
│   └─► Otherwise: store in CustomFields
│
├─► Validate required fields
└─► Return Package or error
```

## Control File Format

The module handles the RFC 822-like Debian control file format:

- Field names are case-insensitive
- Multi-line values with continuation (space prefix)
- Dependency lists: `pkg1 (>= 1.0), pkg2 | pkg3`
- Custom fields with `X-` prefix stored in CustomFields

## Source File Download Flow

```
downloadFiles(destDir, verbose, callback)
│
├─► Validate files exist
├─► Create destination directory (dirPermission)
├─► Create NewDownloader()
│
├─► For each file:
│   └─► downloadSingleFile(downloader, file, ...)
│       ├─► downloadToFile() directly (no temp Package)
│       └─► Verify checksum (SHA256 preferred, MD5 fallback)
│
└─► Print success message (if verbose)
```

## Dependencies

- `net/http`: HEAD requests for download info
- `os`, `path/filepath`: File operations
- `strings`: String manipulation
- `errors`, `fmt`: Error handling

## Integration Points

- **repository.go**: Parses Package data from Packages files
- **downloader.go**: Uses Package for download operations, provides downloadToFile
- **mirror.go**: Uses Package metadata for mirroring

## Error Handling

All errors use `%w` wrapping for proper error chains:

- Returns descriptive errors with context
- Validates required fields (Package, Version, Architecture, Maintainer)
- Handles missing or malformed control data gracefully
