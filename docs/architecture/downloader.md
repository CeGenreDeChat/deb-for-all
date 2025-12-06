# Downloader Architecture - `pkg/debian/downloader.go`

## Overview

The `downloader.go` file provides HTTP download functionality with retry logic, progress tracking, checksum verification, and concurrent download support. It handles both binary and source package downloads.

## Constants

Local constants for download configuration:

```go
const (
    defaultUserAgent     = "deb-for-all/1.0"
    defaultTimeout       = 30 * time.Second
    defaultRetryAttempts = 3
    defaultConcurrency   = 5
    retryDelay           = 2 * time.Second
    downloadBufferSize   = 32 * 1024 // 32KB buffer
)
```

Also uses from `package.go`:
- `DirPermission` for creating directories in `downloadToFile()`

## Data Structures

### Downloader

The main struct for download operations.

```
Downloader
├── UserAgent string        (default: "deb-for-all/1.0")
├── Timeout time.Duration   (default: 30s)
├── RetryAttempts int       (default: 3)
└── VerifyChecksums bool    (default: true)
```

### Internal Types

```
downloadJob
├── pkg *Package
└── destPath string

downloadResult
├── pkg *Package
└── err error
```

## Public API

### Constructor

| Function | Description |
|----------|-------------|
| `NewDownloader()` | Creates a new Downloader with default settings |

### Binary Package Downloads

| Method | Description |
|--------|-------------|
| `DownloadWithProgress(pkg, destPath, callback)` | Downloads with progress callback |
| `DownloadSilent(pkg, destPath)` | Downloads without output |
| `DownloadWithChecksum(pkg, destPath, checksum, type)` | Downloads and verifies checksum |
| `DownloadToDir(pkg, destDir)` | Downloads to directory with auto filename |
| `DownloadToDirSilent(pkg, destDir)` | Silent download to directory |
| `DownloadMultiple(packages, destDir, maxConcurrent)` | Concurrent multi-package download |

### Source Package Downloads

| Method | Description |
|--------|-------------|
| `DownloadSourcePackage(sourcePkg, destDir)` | Downloads all source files |
| `DownloadSourcePackageSilent(sourcePkg, destDir)` | Silent source download |
| `DownloadSourcePackageWithProgress(sourcePkg, destDir, callback)` | Source download with progress |
| `DownloadSourceFile(sourceFile, destDir)` | Downloads a single source file |
| `DownloadOrigTarball(sourcePkg, destDir)` | Downloads only the orig tarball |

### Utility Methods

| Method | Description |
|--------|-------------|
| `GetFileSize(url)` | Returns Content-Length via HEAD request |

### Internal Methods

| Method | Description |
|--------|-------------|
| `newHTTPClient()` | Creates HTTP client with configured timeout |
| `doRequestWithRetry(url, silent)` | Performs HTTP request with retry logic |
| `downloadToFile(url, destPath, callback)` | Core download implementation |
| `copyWithProgress(src, dst, total, callback)` | Copies with progress reporting |
| `verifyChecksum(filePath, expected, type)` | Verifies file checksum |

### Utility Functions

| Function | Description |
|----------|-------------|
| `getPackageFilename(pkg)` | Returns filename, generating one if not set |

## Download Flow

```
DownloadWithProgress(pkg, destPath, callback)
│
├─► Validate download URL
│
├─► downloadToFile(url, destPath, callback)
│   │
│   ├─► Create parent directories (os.MkdirAll)
│   │
│   ├─► doRequestWithRetry(url, silent)
│   │   │
│   │   ├─► newHTTPClient() with timeout
│   │   │
│   │   └─► Retry Loop (max 3 attempts)
│   │       ├─► Create HTTP request with User-Agent
│   │       ├─► Execute request
│   │       ├─► Check response status (200 OK)
│   │       └─► Sleep retryDelay on failure
│   │
│   ├─► Create destination file
│   │
│   └─► copyWithProgress() or io.Copy()
│       ├─► Read from response body (32KB buffer)
│       ├─► Write to file
│       └─► Call progress callback
│
└─► Print success message
```

## Concurrent Downloads

The `DownloadMultiple` method uses Go's concurrency primitives with `sync.WaitGroup`:

```
DownloadMultiple(packages, destDir, maxConcurrent)
│
├─► Create jobs channel (buffered)
├─► Create results channel (buffered)
│
├─► Start worker goroutines with WaitGroup
│   └─► Each worker: defer wg.Done()
│       └─► Process jobs from channel
│
├─► Queue all packages as jobs
├─► Close jobs channel
│
├─► Goroutine: wg.Wait() then close(results)
│
└─► Collect results from channel and return errors
```

Default concurrency: 5 workers

## Retry Logic

```
doRequestWithRetry(url, silent)
│
├─► Max Attempts: 3 (defaultRetryAttempts)
├─► Delay Between Retries: 2s (retryDelay)
│
├─► Retry Conditions:
│   ├── Network errors
│   └── Non-200 HTTP status
│
├─► On failure (if not silent):
│   └── Print retry message
│
└─► Success Condition: HTTP 200 OK
```

## Checksum Verification

Supports MD5 and SHA256 checksums:

```
verifyChecksum(filePath, expectedChecksum, checksumType)
│
├─► Open file
├─► Create hasher (md5.New() or sha256.New())
├─► io.Copy file content to hasher
├─► Compare computed vs expected
├─► Print success message
└─► Return error if mismatch
```

## Progress Callback

The progress callback signature:

```go
type ProgressCallback func(downloaded, total int64)
```

- `downloaded`: Bytes downloaded so far
- `total`: Total file size (from Content-Length, -1 if unknown)

## Buffer Configuration

| Constant | Value | Purpose |
|----------|-------|---------|
| `downloadBufferSize` | 32 KB | Read/write chunks in copyWithProgress |

## Dependencies

- `net/http`: HTTP client
- `crypto/md5`, `crypto/sha256`: Checksum computation
- `hash`: Hash interface
- `io`: Stream operations
- `os`, `path/filepath`: File operations
- `sync`: WaitGroup for concurrent downloads
- `time`: Timeout and retry delays
- `fmt`, `strings`: Formatting

## Integration Points

- **package.go**: Uses Package and SourcePackage structs
- **repository.go**: Created by Repository for downloads
- **mirror.go**: Uses Downloader for file downloads

## Error Handling

All errors use `%w` wrapping for proper error chains:

| Scenario | Handling |
|----------|----------|
| No download URL | Return error immediately |
| Directory creation failure | Return wrapped error |
| All retry attempts failed | Return error with attempt count |
| File creation failure | Return wrapped error |
| Write error | Return wrapped error |
| Checksum mismatch | Return error with expected vs actual |

## HTTP Configuration

| Setting | Default Value |
|---------|---------------|
| User-Agent | `deb-for-all/1.0` |
| Timeout | 30 seconds |
| Method | GET (download), HEAD (size check) |
