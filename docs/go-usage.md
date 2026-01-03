# Using deb-for-all as a Go library

This guide shows how to embed the `pkg/debian` library in your own Go application to fetch metadata, download packages, and mirror repositories.

## Prerequisites
- Go 1.20+ (module-enabled workspace)
- Network access to your target Debian mirror (default: `http://deb.debian.org/debian`)
- Optional: GPG keyrings if you keep signature verification on

Add the module:
```bash
go get github.com/CeGenreDeChat/deb-for-all@latest
```

## Fetch repository metadata
Create a repository client, optionally disable signature verification if you do not want Release/InRelease checks.
```go
import "github.com/CeGenreDeChat/deb-for-all/pkg/debian"

repo := debian.NewRepository(
    "example-repo",
    "http://deb.debian.org/debian",
    "Debian mirror",
    "bookworm",
    []string{"main"},
    []string{"amd64"},
)

// Optional: Configure GPG keyrings (defaults to OS-specific system paths if empty)
// On Windows, this looks in %APPDATA%\gnupg and %ProgramFiles%\GnuPG
// On Linux, it checks /usr/share/keyrings and /etc/apt/trusted.gpg.d
// repo.SetKeyringPathsWithDirs([]string{"/path/to/keyring.gpg"}, nil)

// Disable verification if needed
// repo.DisableSignatureVerification()

names, err := repo.FetchPackages()
if err != nil {
    // handle error
}
_ = names // package names found across sections/arches

pkgMeta, err := repo.GetPackageMetadata("hello")
if err != nil {
    // handle not found
}
```

## Resolve dependencies
Resolve a set of packages with apt-like dependency closure, excluding optional kinds when needed.
```go
specs := []debian.PackageSpec{{Name: "systemd"}}
exclude := map[string]bool{"recommends": true, "suggests": true}
resolved, err := repo.ResolveDependencies(specs, exclude)
if err != nil {
    // handle resolution error
}
// resolved is a map[string]Package keyed by name
```

## Download packages
Fetch metadata first, then pick the package (with architecture preference) and download using the recorded URL and checksums.
```go
import "path/filepath"

repo := debian.NewRepository(
    "download-repo",
    "http://deb.debian.org/debian",
    "Debian mirror",
    "bookworm",
    []string{"main"},
    []string{"amd64"}, // first architecture is selected
)
repo.DisableSignatureVerification() // keep enabled if you verify Release/InRelease

if _, err := repo.FetchPackages(); err != nil {
    // handle metadata fetch error
}

pkgMeta, err := repo.GetPackageMetadataWithArch("hello", "", []string{"amd64"}) // version optional
if err != nil {
    // handle not found
}

destDir := "./downloads"
destPath := filepath.Join(destDir, filepath.Base(pkgMeta.Filename))

d := debian.NewDownloader()

if skip, err := d.ShouldSkipDownload(pkgMeta, destPath); err != nil {
    // handle stat/checksum error
} else if !skip {
    if err := d.DownloadWithProgress(pkgMeta, destPath, func(filename string, downloaded, total int64) {
        // See 'Progress callback: func(filename string, downloaded, total int64)' section above for details.
        // update progress
    }); err != nil {
        // handle download failure
    }
}
```

## Download source packages
Use `Repository` to locate source entries, then pass the resulting `SourcePackage` (with URLs and hashes) to the downloader. When `version` is empty, the latest available source version is selected from Sources metadata.
```go
repo := debian.NewRepository(
    "source-repo",
    "http://deb.debian.org/debian",
    "Debian mirror",
    "bookworm",
    []string{"main"},
    []string{"source"}, // architectures are ignored for Sources but kept for symmetry
)
repo.DisableSignatureVerification() // keep enabled if you verify Release/InRelease

if _, err := repo.FetchSources(); err != nil {
    // handle metadata fetch error
}

// Auto-latest version (pass a version string to lock a specific release)
sp, err := repo.GetSourcePackageMetadata("hello", "")
if err != nil {
    // handle not found
}

d := debian.NewDownloader()

// Download all source files with progress
if err := d.DownloadSourcePackageWithProgress(sp, "./downloads/src", func(filename string, downloaded, total int64) {
    // See 'Progress callback: func(filename string, downloaded, total int64)' section above for details.
    // update your UI or logs here
}); err != nil {
    // handle failure
}

// Download only the original tarball
if err := d.DownloadOrigTarball(sp, "./downloads/src"); err != nil {
    // handle failure
}
```

## Mirror a repository (metadata + optional .deb files)
Mirror orchestrates Release/Packages fetch and optional package downloads into Debian layout under `dists/` and `pool/`.
```go
cfg := debian.MirrorConfig{
    BaseURL:          "http://deb.debian.org/debian",
    Suites:           []string{"bookworm"},
    Components:       []string{"main"},
    Architectures:    []string{"amd64"},
    DownloadPackages: true,  // default; set false for metadata-only mirrors
    Verbose:          true,
    SkipGPGVerify:    true,  // set false to enforce signatures
    RateDelay:        0,     // delay between .deb downloads; >0 forces sequential mode (useful for legacy repos)
}

mirror := debian.NewMirror(cfg, "./mirror")
if err := mirror.Clone(); err != nil {
    // handle mirror failure
}
```

## Tips
- Checksums: the downloader prefers SHA256; provide `SHA256`/`MD5sum` in `Package` when available to enable skip logic.
- Timeouts/retries: defaults are 30s timeout, 3 attempts, 2s backoff; tune fields on `Downloader` if needed.
- Paths and permissions: directory and file permissions are standardized via `DirPermission`/`FilePermission` from `package.go`.
- GPG verification: keep it enabled for production; disable only when you explicitly trust the source or run tests.
- Localization/UI: the library itself is headless; CLI layers handle i18n. When embedding, surface your own user-facing messages.

## Progress callback: func(filename string, downloaded, total int64)

This library provides download helper methods that accept a progress callback with the signature:

```go
func(filename string, downloaded, total int64)
```

- `filename`: the name (or relative path) of the file currently being downloaded.
- `downloaded`: number of bytes downloaded so far for this file.
- `total`: expected total size in bytes for the file — may be `0` or unknown if the server does not send a Content-Length.

How it's used:

- The downloader calls the callback periodically (for example after each chunk) while writing the file.
- Typical uses: update a progress bar, log progress, compute percentage and ETA.
- If `total <= 0`, show a byte count or an indeterminate progress indicator.

Example:

```go
func(filename string, downloaded, total int64) {
    if total > 0 {
        pct := float64(downloaded) * 100.0 / float64(total)
        fmt.Printf("%s: %.1f%% (%d/%d bytes)\n", filename, pct, downloaded, total)
    } else {
        fmt.Printf("%s: %d bytes downloaded (total unknown)\n", filename, downloaded)
    }
}
```

You can pass this callback to `DownloadWithProgress`, `DownloadSourcePackageWithProgress`, and similar helpers to get fine-grained download updates.
