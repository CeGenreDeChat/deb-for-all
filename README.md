# README for deb-for-all

# deb-for-all

deb-for-all is a Go library designed for managing Debian packages. This project provides both a library and a command-line binary to facilitate the handling of Debian packages efficiently, including downloading capabilities.

## Features

- Manage Debian packages with ease.
- **Download Debian packages** from repositories or direct URLs
- **Progress tracking** and retry mechanisms for downloads
- **Checksum verification** for downloaded packages
- **Concurrent downloads** for multiple packages
- Read, write, and validate Debian control files.
- Interact with Debian package repositories.
- Utility functions for common tasks.

## Installation

To install the deb-for-all library, you can use the following command:

```bash
go get github.com/CeGenreDeChat/deb-for-all
```

## Usage

To use the library in your Go application, import it as follows:

```go
import "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
```

### Basic Package Download

```go
package main

import (
    "fmt"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    // Create a package with download information
    pkg := &debian.Package{
        Name:         "example-package",
        Version:      "1.0.0",
        Architecture: "amd64",
        DownloadURL:  "https://example.com/package.deb",
        Filename:     "example-package_1.0.0_amd64.deb",
    }

    // Simple download
    err := pkg.Download("./downloads")
    if err != nil {
        fmt.Printf("Download failed: %v\n", err)
    }
}
```

### Advanced Download with Progress

```go
// Create a downloader with custom settings
downloader := debian.NewDownloader()
downloader.RetryAttempts = 3
downloader.VerifyChecksums = true

// Progress callback
progressCallback := func(downloaded, total int64) {
    if total > 0 {
        percentage := float64(downloaded) / float64(total) * 100
        fmt.Printf("\rProgress: %.1f%%", percentage)
    }
}

// Download with progress reporting
err := downloader.DownloadWithProgress(pkg, "./downloads/package.deb", progressCallback)
if err != nil {
    fmt.Printf("Download failed: %v\n", err)
}
```

### Repository Usage

```go
// Create a repository
repo := debian.NewRepository("debian-main", "http://deb.debian.org/debian", "Main Debian Repository")

// Check if a package is available
available, err := repo.CheckPackageAvailability("curl", "7.74.0-1.3", "amd64")
if err != nil {
    fmt.Printf("Error: %v\n", err)
} else {
    fmt.Printf("Package available: %v\n", available)
}

// Download from repository
err = repo.DownloadPackage("curl", "7.74.0-1.3", "amd64", "./downloads")
if err != nil {
    fmt.Printf("Download failed: %v\n", err)
}
```

### Multiple Package Downloads

```go
packages := []*debian.Package{
    {Name: "package1", DownloadURL: "https://example.com/package1.deb"},
    {Name: "package2", DownloadURL: "https://example.com/package2.deb"},
    {Name: "package3", DownloadURL: "https://example.com/package3.deb"},
}

downloader := debian.NewDownloader()
errors := downloader.DownloadMultiple(packages, "./downloads", 5) // Max 5 concurrent downloads

// Handle any errors
for _, err := range errors {
    fmt.Printf("Error: %v\n", err)
}
```

You can find more examples in the `examples/` directory:
- `examples/basic/` - Basic usage example
- `examples/download/` - Real download examples with Debian packages

## Command-Line Tool

The project also includes a command-line tool. To run the tool, navigate to the `cmd/deb-for-all` directory and execute:

```bash
go run main.go
```

## Documentation

API documentation is available in the `docs/api.md` file. This documentation provides detailed information about the functions and types exported by the library.

## Contributing

Contributions are welcome! Please read the [CONTRIBUTING.md](CONTRIBUTING.md) file for guidelines on how to contribute to this project.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.