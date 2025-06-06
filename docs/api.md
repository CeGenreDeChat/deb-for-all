# API Documentation

## Overview

This document provides an overview of the API for the Debian package management library. It describes the various types and functions that are available for managing Debian packages, including downloading capabilities.

## Package Types

### Package

```go
type Package struct {
    Name         string // Nom du paquet
    Version      string // Version du paquet
    Architecture string // Architecture du paquet
    Maintainer   string // Responsable du paquet
    Description  string // Description du paquet
    DownloadURL  string // URL de téléchargement du paquet
    Filename     string // Nom du fichier .deb
    Size         int64  // Taille du paquet en bytes
}
```

### ControlFile

```go
type ControlFile struct {
    Package     string
    Version     string
    Maintainer  string
    Architecture string
    Description string
}
```

### Repository

```go
type Repository struct {
    Name        string // Nom du dépôt
    URL         string // URL du dépôt
    Description string // Description du dépôt
}
```

### Downloader

```go
type Downloader struct {
    UserAgent      string
    Timeout        time.Duration
    RetryAttempts  int
    VerifyChecksums bool
}
```

### DownloadInfo

```go
type DownloadInfo struct {
    URL           string
    ContentLength int64
    ContentType   string
    LastModified  string
}
```

## Functions

### ManagePackages

```go
func ManagePackages(pkg Package) error
```
This function manages the installation, removal, or upgrade of a Debian package.

### ReadControlFile

```go
func ReadControlFile(path string) (ControlFile, error)
```
This function reads a Debian control file from the specified path and returns a ControlFile struct.

### WriteControlFile

```go
func WriteControlFile(path string, control ControlFile) error
```
This function writes a ControlFile struct to the specified path.

### SearchPackage

```go
func SearchPackage(name string) ([]Package, error)
```
This function searches for packages by name and returns a list of matching packages.

### Package Methods

#### Download

```go
func (p *Package) Download(destDir string) error
```
Downloads the Debian package to the specified directory.

#### DownloadToFile

```go
func (p *Package) DownloadToFile(filePath string) error
```
Downloads the package to a specific file path.

#### GetDownloadInfo

```go
func (p *Package) GetDownloadInfo() (*DownloadInfo, error)
```
Retrieves download information without downloading the file.

#### DownloadSilent

```go
func (p *Package) DownloadSilent(destDir string) error
```
Downloads the package silently to the specified directory without any console output. Perfect for integration into Go applications without output pollution.

#### DownloadToFileSilent

```go
func (p *Package) DownloadToFileSilent(filePath string) error
```
Downloads the package silently to a specific file path without any console output.

### Repository Methods

#### DownloadPackage

```go
func (r *Repository) DownloadPackage(packageName, version, architecture, destDir string) error
```
Downloads a package from the repository.

#### DownloadPackageByURL

```go
func (r *Repository) DownloadPackageByURL(packageURL, destDir string) error
```
Downloads a package from a specific URL.

#### CheckPackageAvailability

```go
func (r *Repository) CheckPackageAvailability(packageName, version, architecture string) (bool, error)
```
Checks if a package is available in the repository.

### Downloader Methods

#### NewDownloader

```go
func NewDownloader() *Downloader
```
Creates a new Downloader instance with default settings.

#### DownloadWithProgress

```go
func (d *Downloader) DownloadWithProgress(pkg *Package, destPath string, progressCallback func(downloaded, total int64)) error
```
Downloads a package with progress reporting.

#### DownloadWithChecksum

```go
func (d *Downloader) DownloadWithChecksum(pkg *Package, destPath, checksum, checksumType string) error
```
Downloads a package and verifies its checksum.

#### DownloadMultiple

```go
func (d *Downloader) DownloadMultiple(packages []*Package, destDir string, maxConcurrent int) []error
```
Downloads multiple packages concurrently.

#### GetFileSize

```go
func (d *Downloader) GetFileSize(url string) (int64, error)
```
Gets the file size from a URL without downloading.

#### DownloadSilent

```go
func (d *Downloader) DownloadSilent(pkg *Package, destPath string) error
```
Downloads a package silently without any console output or progress reporting. Ideal for integration into Go code without polluting the output.

## Error Handling

The library defines custom error types to handle specific errors related to package management and downloading. These errors can be used to provide more context when an operation fails.

## Usage Examples

### Basic Package Download

```go
package main

import (
    "fmt"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    pkg := &debian.Package{
        Name:         "example-package",
        Version:      "1.0.0",
        Architecture: "amd64",
        DownloadURL:  "https://example.com/package.deb",
        Filename:     "example-package_1.0.0_amd64.deb",
    }

    err := pkg.Download("./downloads")
    if err != nil {
        fmt.Printf("Download failed: %v\n", err)
    }
}
```

### Advanced Download with Progress

```go
downloader := debian.NewDownloader()
downloader.RetryAttempts = 3

progressCallback := func(downloaded, total int64) {
    percentage := float64(downloaded) / float64(total) * 100
    fmt.Printf("\rProgress: %.1f%%", percentage)
}

err := downloader.DownloadWithProgress(pkg, "./downloads/package.deb", progressCallback)
if err != nil {
    fmt.Printf("Download failed: %v\n", err)
}
```

### Repository Usage

```go
repo := debian.NewRepository(
    "debian-main",
    "http://deb.debian.org/debian",
    "Main Debian Repository",
    "bookworm",                              // Distribution
    []string{"main", "contrib", "non-free"}, // Sections
    []string{"amd64"},                       // Architectures
)

// Check availability
available, err := repo.CheckPackageAvailability("curl", "7.74.0-1.3", "amd64")
if err != nil {
    fmt.Printf("Error checking availability: %v\n", err)
}

// Download from repository
err = repo.DownloadPackage("curl", "7.74.0-1.3", "amd64", "./downloads")
if err != nil {
    fmt.Printf("Download failed: %v\n", err)
}
```

### Multiple Package Download

```go
packages := []*debian.Package{
    {Name: "package1", DownloadURL: "https://example.com/package1.deb"},
    {Name: "package2", DownloadURL: "https://example.com/package2.deb"},
}

downloader := debian.NewDownloader()
errors := downloader.DownloadMultiple(packages, "./downloads", 5)
for _, err := range errors {
    fmt.Printf("Error: %v\n", err)
}
```

### Silent Download for Integration

```go
package main

import (
    "log"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func downloadPackageQuietly(name, url, destDir string) error {
    pkg := &debian.Package{
        Name:        name,
        DownloadURL: url,
    }

    // Silent download without any console output
    return pkg.DownloadSilent(destDir)
}

func main() {
    // Perfect for integration into business logic
    err := downloadPackageQuietly("hello",
        "http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2_amd64.deb",
        "./downloads")
    if err != nil {
        log.Printf("Failed to download package: %v", err)
        return
    }

    // Continue with your business logic...
    log.Println("Package downloaded successfully")
}
```

### Silent Download with Downloader

```go
downloader := debian.NewDownloader()
downloader.RetryAttempts = 3
downloader.VerifyChecksums = false

pkg := &debian.Package{
    Name:        "example",
    DownloadURL: "https://example.com/package.deb",
}

// Silent download with retry logic but no console output
err := downloader.DownloadSilent(pkg, "./downloads/package.deb")
if err != nil {
    // Handle error silently or log it
    log.Printf("Download failed: %v", err)
}
```

## Conclusion

This API provides a comprehensive set of functions and types for managing and downloading Debian packages. The library supports various download scenarios including basic downloads, progress reporting, checksum verification, concurrent downloads, and repository-based downloads. For more detailed usage and examples, please refer to the documentation in the `examples` directory.