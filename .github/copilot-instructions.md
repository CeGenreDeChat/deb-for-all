# deb-for-all AI Coding Agent Instructions

## Context-Specific Instructions

This project uses specialized instruction files for different contexts. Apply the relevant instructions based on the task:

| Instruction File | Apply When |
|------------------|------------|
| `.github/instructions/go.instructions.md` | Writing or modifying Go code |
| `.github/instructions/python.instructions.md` | Writing or modifying Python code |
| `.github/instructions/test_suite.instructions.md` | Creating or editing Robot Framework test suites |

## Project Architecture

**deb-for-all** is a Go library and CLI tool for Debian package management and repository mirroring. The codebase follows a clean architecture pattern:

- **`pkg/debian/`**: Core library with four main components:
  - `package.go`: Debian package parsing and metadata handling 
  - `repository.go`: Repository interaction, Release/Packages file parsing
  - `downloader.go`: HTTP downloading with retry, progress tracking, checksum verification
  - `mirror.go`: High-level mirroring orchestration combining the above components
- **`cmd/deb-for-all/`**: CLI binary using Cobra framework with i18n support
  - `main.go`: Entry point, Config struct, i18n initialization, command dispatcher
  - `root.go`: Cobra root command and subcommands definition
  - `help.go`: Help message customization
  - `commands/`: Command implementations (binary.go, source.go, repo.go)
  - `locales/`: Translation files (en.toml, fr.toml)
- **`examples/`**: Working examples demonstrating library usage patterns
- **`internal/`**: Private utilities (config, errors) not part of public API

## Key Patterns & Conventions

### Shared Constants (use these, avoid literals)
- `DirPermission`, `FilePermission` (from `pkg/debian/package.go`) for all filesystem permissions
- `CompressionExtensions` (from `pkg/debian/package.go`) for Packages file formats

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

### CLI Architecture (Cobra + i18n)
The CLI uses [Cobra](https://github.com/spf13/cobra) for command parsing and [go-i18n](https://github.com/nicksnyder/go-i18n) for internationalization:
```go
// Commands are defined in root.go using Cobra
rootCmd.AddCommand(downloadCmd)
rootCmd.AddCommand(downloadSourceCmd)
rootCmd.AddCommand(mirrorCmd)

// Translations loaded from locales/*.toml files
bundle.MustLoadMessageFile("cmd/deb-for-all/locales/en.toml")
bundle.MustLoadMessageFile("cmd/deb-for-all/locales/fr.toml")

// Language selection via DEB_FOR_ALL_LANG environment variable
lang := os.Getenv("DEB_FOR_ALL_LANG") // "en" or "fr"
```

### Error Handling Pattern
- Always wrap errors with `%w`: `fmt.Errorf("failed to X: %w", err)` (avoid `%v`)
- Custom errors in `internal/errors/` with structured codes and messages
- Retry logic with exponential backoff in downloaders
- Validation methods on config structs: `func (c *Config) Validate() error`

### File Parsing & Compression
Repository files support multiple compression formats automatically. Use the shared `CompressionExtensions` slice (from `pkg/debian/package.go`) instead of hard-coding extensions. Uses `github.com/ulikunitz/xz` for XZ decompression.

### CLI Command Structure
```bash
# Subcommand-based pattern using Cobra:
deb-for-all <command> [flags]

# Download binary package:
deb-for-all download -p <package> --version <ver> -d ./dest

# Download source package:
deb-for-all download-source -p <package> --orig-only -d ./dest

# Mirror operations:
deb-for-all mirror -u http://deb.debian.org/debian --suites bookworm -d ./mirror
deb-for-all mirror --download-packages --components main,contrib -v
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
3. **New CLI command**: Add subcommand in `root.go`, implementation in `commands/` directory
4. **Translations**: Update `locales/en.toml` and `locales/fr.toml` with new message keys
5. **Documentation**: Update relevant files (`README.md`, `docs/`) when behavior or flags change
6. **Example usage**: Add to `examples/` directory
7. **Robot tests**: Add test cases to `test/suites/*.robot` using existing patterns

### Testing Strategy
- **No unit tests** by design (stated in project goal)
- **Robot Framework** integration tests in `test/suites/`
- **Examples** serve as living documentation and manual testing
- **Python test runner**: Use `test/start.py` to run Robot Framework tests (requires virtual environment)

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
- Commit messages must be entirely in English (avoid foreign languages)
- Breaking changes require `!` in type or `BREAKING CHANGE:` footer

### Versioning
- Follow semantic versioning (SemVer)
- Major version 0.x.x for initial development (current state)
- Increment patch for bug fixes, minor for features, major for breaking changes

### Code Quality
- **No unit tests** by project design
- Examples in `examples/` directory must remain functional
- Follow Go conventions: `gofmt`, exported functions documented
- English user messages preserved in CLI output

## Agent Commit Summary

- Always write a temporary summary file after code or documentation changes so it can be reused for the commit message.
- Place it in the OS temp directory (for example `os.TempDir()` or `%TEMP%`), name it clearly (e.g., `agent-changes-summary.txt`), and print the full path for later retrieval.
- Content: one-line title plus a short paragraph of key details; keep it concise and readable.
- Purpose: make commit drafting easy without re-summarizing work.
- Windows note: create/update with PowerShell, e.g. `Set-Content -Encoding UTF8 -Path (Join-Path $env:TEMP "agent-changes-summary.txt") -Value "Title`nDetails"` and then `Get-Content` to confirm.

---

*This project creates a library for managing Debian packages within Go projects and provides a binary for the same functionality.*
