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
repo.DisableSignatureVerification() // keep or remove based on your trust policy

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
Use the downloader to fetch artifacts; it retries, verifies checksums, and can skip already verified files.
```go
d := debian.NewDownloader()
pkg := &debian.Package{
    Name:         "hello",
    Version:      "2.10-2",
    Architecture: "amd64",
    DownloadURL:  "http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2_amd64.deb",
    Filename:     "hello_2.10-2_amd64.deb",
}

if err := d.DownloadToDir(pkg, "./downloads"); err != nil {
    // handle download failure
}
```

Download multiple in parallel (default 5 workers):
```go
pkgs := []*debian.Package{pkg /*, more */}
if err := d.DownloadMultiple(pkgs, "./downloads", nil); err != nil {
    // handle failure
}
```

## Download source packages
Use `Repository` to locate source entries and `Downloader` to fetch the associated `.dsc`, `.orig`, and `.debian` tarballs.
```go
// Fetch Packages/Sources metadata first
repo := debian.NewRepository(
    "example-repo",
    "http://deb.debian.org/debian",
    "Debian mirror",
    "bookworm",
    []string{"main"},
    []string{"amd64"},
)
repo.DisableSignatureVerification() // optional
if _, err := repo.FetchPackages(); err != nil {
    // handle error
}

// Get source package metadata (directory and files)
sp, err := repo.GetSourcePackage("hello")
if err != nil {
    // handle missing source package
}

d := debian.NewDownloader()
for i := range sp.Files {
    sf := sp.Files[i]
    // SourceFile already carries URL; Directory may be set for pool layout
    if err := d.DownloadToDir(&debian.Package{DownloadURL: sf.URL, Filename: sf.Name}, "./downloads/src"); err != nil {
        // handle download failure for this file
    }
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
    DownloadPackages: true,  // set false for metadata-only
    Verbose:          true,
    SkipGPGVerify:    true,  // set false to enforce signatures
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
