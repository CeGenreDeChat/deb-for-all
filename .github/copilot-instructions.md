# deb-for-all AI Coding Agent Instructions

## Project Architecture

**deb-for-all** is a Go library and CLI tool for Debian package management and repository mirroring. The codebase follows a clean architecture pattern:

- **`pkg/debian/`**: Core library with four main components:
  - `package.go`: Debian package parsing and metadata handling 
  - `repository.go`: Repository interaction, Release/Packages file parsing
  - `downloader.go`: HTTP downloading with retry, progress tracking, checksum verification
  - `mirror.go`: High-level mirroring orchestration combining the above components
- **`cmd/deb-for-all/`**: CLI binary that wraps the library functionality
- **`examples/`**: Working examples demonstrating library usage patterns
- **`internal/`**: Private utilities (config, errors) not part of public API

## Key Patterns & Conventions

### Core Types and Relationships
```go
// Central abstraction: Package struct with extensive metadata fields
type Package struct {
    Name, Version, Architecture string
    DownloadURL, Filename string
    Size int64
    MD5sum, SHA256 string
    // ... 50+ Debian control fields
}

// Repository handles metadata fetching and parsing
type Repository struct {
    URL, Distribution string
    Sections, Architectures []string
    PackageMetadata []Package
}
```

### Error Handling Pattern
- Use `fmt.Errorf()` with error wrapping: `fmt.Errorf("failed to X: %w", err)`
- Custom errors in `internal/errors/` with structured codes and messages
- Retry logic with exponential backoff in downloaders
- Validation methods on config structs: `func (c *Config) Validate() error`

### File Parsing & Compression
Repository files support multiple compression formats automatically:
```go
extensions := []string{"", ".gz", ".xz"}  // Try uncompressed, gzip, xz
```
Uses `github.com/ulikunitz/xz` for XZ decompression (only external dependency).

### CLI Command Structure
```bash
# All commands follow this pattern:
deb-for-all -command <action> [specific-flags]

# Mirror operations (core functionality):
deb-for-all -command mirror -dest ./path -verbose
deb-for-all -command mirror -download-packages -suites bookworm,bullseye

# Package operations:
deb-for-all -command download -package <name> -version <ver>
```

## Development Workflow

### Build & Test Commands
```bash
make build          # Cross-platform build (handles Windows .exe)
make test           # Run Go tests
make mirror-example # Run examples/mirror/main.go
```

### Adding New Features
1. **Library changes**: Add to appropriate `pkg/debian/*.go` file
2. **CLI integration**: Update `cmd/deb-for-all/main.go` flag parsing and run() function
3. **Example usage**: Add to `examples/` directory
4. **Robot tests**: Add test cases to `test/suites/*.robot` using existing patterns

### Testing Strategy
- **No unit tests** by design (stated in project goal)
- **Robot Framework** integration tests in `test/suites/`
- **Examples** serve as living documentation and manual testing

## Debian-Specific Knowledge

### Repository Structure
Standard Debian repository layout:
```
dists/<suite>/
  ├── Release                    # Main metadata file
  ├── <component>/binary-<arch>/ # e.g., main/binary-amd64/
  │   └── Packages[.gz|.xz]     # Package listings
  └── <component>/source/
      └── Sources[.gz|.xz]      # Source package listings
```

### Package Metadata Parsing
Control file format is RFC 822-like with specific Debian extensions:
```go
// Parse dependency lists: "pkg1 (>= 1.0), pkg2 | pkg3"
// Handle multi-line fields with proper continuation
// Support X- prefixed custom fields via CustomFields map
```

## Integration Points

### External Dependencies
- **Minimal**: Only `github.com/ulikunitz/xz` for XZ compression
- **HTTP**: Standard library `net/http` with custom retry logic
- **Crypto**: Uses `crypto/md5` and `crypto/sha256` for verification

### Cross-Platform Considerations
- **Windows support**: Makefile handles `.exe` extension automatically  
- **Path handling**: Use `filepath.Join()` consistently
- **Shell commands**: PowerShell compatible in documentation

## Common Modification Patterns

When adding new Debian package fields, update `Package` struct in `package.go` and parsing logic in `repository.go`. When adding CLI commands, follow the flag-based pattern in `main.go`. For new mirror features, extend `MirrorConfig` and implement in `mirror.go` methods.

## Compliance Requirements

### Commit Standards
- Follow conventional commits: `feat:`, `fix:`, `docs:`, etc.
- Include scope when appropriate: `feat(mirror):`, `fix(parser):`
- Use French for all commit messages
- Breaking changes require `!` in type or `BREAKING CHANGE:` footer

### Versioning
- Follow semantic versioning (SemVer)
- Major version 0.x.x for initial development (current state)
- Increment patch for bug fixes, minor for features, major for breaking changes

### Code Quality
- **No unit tests** by project design
- Examples in `examples/` directory must remain functional
- Follow Go conventions: `gofmt`, exported functions documented
- French user messages preserved in CLI output

---

*This project creates a library for managing Debian packages within Go projects and provides a binary for the same functionality.*
