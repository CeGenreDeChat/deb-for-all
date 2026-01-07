package debian

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/ulikunitz/xz"
)

// Repository configuration constants.
const (
	packagesBufferSize   = 1024 * 1024 // 1MB buffer for Packages file parsing
	packagesInitialAlloc = 64 * 1024   // Initial allocation for scanner buffer
)

// Default repository components for package search.
// Note: non-free-firmware was introduced in Debian 12 (Bookworm).
var defaultComponents = []string{"main", "contrib", "non-free", "non-free-firmware"}

// ErrGPGNotFound is returned when gpgv executable cannot be found on Windows.
var ErrGPGNotFound = fmt.Errorf("gpgv executable not found: please install Gpg4win from https://www.gpg4win.org/ or add gpgv.exe to your PATH")

// GetDefaultKeyringPaths returns standard system locations for GPG keyrings based on OS.
func GetDefaultKeyringPaths() []string {
	switch runtime.GOOS {
	case "windows":
		var paths []string
		// GnuPG for Windows (Gpg4win) - user keyrings
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			pubring := filepath.Join(appdata, "gnupg", "pubring.kbx")
			if info, err := os.Stat(pubring); err == nil && !info.IsDir() {
				paths = append(paths, pubring)
			}
		}
		// Gpg4win system installation
		if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
			shareDir := filepath.Join(programFiles, "GnuPG", "share", "gnupg")
			if matches, err := filepath.Glob(filepath.Join(shareDir, "*.gpg")); err == nil {
				paths = append(paths, matches...)
			}
		}
		return paths
	case "darwin":
		var paths []string
		// Homebrew Intel
		paths = append(paths, "/usr/local/share/keyrings/debian-archive-keyring.gpg")
		paths = append(paths, "/usr/local/share/keyrings/ubuntu-archive-keyring.gpg")
		// Homebrew Apple Silicon
		paths = append(paths, "/opt/homebrew/share/keyrings/debian-archive-keyring.gpg")
		paths = append(paths, "/opt/homebrew/share/keyrings/ubuntu-archive-keyring.gpg")
		// User GnuPG
		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths, filepath.Join(home, ".gnupg", "pubring.kbx"))
		}
		return paths
	default: // linux, freebsd, etc.
		return []string{
			"/usr/share/keyrings/debian-archive-keyring.gpg",
			"/usr/share/keyrings/ubuntu-archive-keyring.gpg",
		}
	}
}

// GetDefaultKeyringDirs returns directories that may contain multiple keyring files based on OS.
func GetDefaultKeyringDirs() []string {
	switch runtime.GOOS {
	case "windows":
		var dirs []string
		// User GnuPG directory
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			gnupgDir := filepath.Join(appdata, "gnupg")
			if info, err := os.Stat(gnupgDir); err == nil && info.IsDir() {
				dirs = append(dirs, gnupgDir)
			}
		}
		// Local app data GnuPG
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			gnupgDir := filepath.Join(localAppData, "GnuPG")
			if info, err := os.Stat(gnupgDir); err == nil && info.IsDir() {
				dirs = append(dirs, gnupgDir)
			}
		}
		return dirs
	case "darwin":
		var dirs []string
		dirs = append(dirs, "/usr/local/share/keyrings")
		dirs = append(dirs, "/opt/homebrew/share/keyrings")
		if home := os.Getenv("HOME"); home != "" {
			dirs = append(dirs, filepath.Join(home, ".gnupg"))
		}
		return dirs
	default:
		return []string{"/etc/apt/trusted.gpg.d"}
	}
}

// getGPGVCommand returns the gpgv executable path for the current OS.
// On Windows, it searches common Gpg4win installation paths.
// Returns an error if gpgv is not found on Windows.
func getGPGVCommand() (string, error) {
	if runtime.GOOS == "windows" {
		// Try common Windows installation paths for Gpg4win
		commonPaths := []string{}
		if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
			commonPaths = append(commonPaths, filepath.Join(programFiles, "GnuPG", "bin", "gpgv.exe"))
		}
		if programFilesX86 := os.Getenv("ProgramFiles(x86)"); programFilesX86 != "" {
			commonPaths = append(commonPaths, filepath.Join(programFilesX86, "GnuPG", "bin", "gpgv.exe"))
		}
		// Also check PATH
		if path, err := exec.LookPath("gpgv.exe"); err == nil {
			return path, nil
		}
		if path, err := exec.LookPath("gpgv"); err == nil {
			return path, nil
		}
		for _, p := range commonPaths {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p, nil
			}
		}
		return "", ErrGPGNotFound
	}
	// On Unix-like systems, assume gpgv is in PATH
	return "gpgv", nil
}

// PackageInfo contains lightweight package information for search results.
type PackageInfo struct {
	Name         string
	Version      string
	Architecture string
	Section      string
	DownloadURL  string
	Size         int64
}

// ReleaseFile represents the parsed Release file from a Debian repository.
type ReleaseFile struct {
	Origin        string
	Label         string
	Suite         string
	Version       string
	Codename      string
	Date          string
	Description   string
	Architectures []string
	Components    []string
	MD5Sum        []FileChecksum
	SHA1          []FileChecksum
	SHA256        []FileChecksum
}

// FileChecksum represents a single checksum entry from a Release file.
type FileChecksum struct {
	Hash     string
	Size     int64
	Filename string
}

// Repository handles interactions with a Debian repository, including
// fetching Release files, Packages metadata, and downloading packages.
type Repository struct {
	Name            string
	URL             string
	Description     string
	Suite           string
	Components      []string
	Architectures   []string
	Packages        []string
	PackageMetadata []Package
	SourceMetadata  []SourcePackage
	ReleaseInfo     *ReleaseFile
	VerifyRelease   bool
	VerifySignature bool
	KeyringPaths    []string
	WarningHandler  func(string)
}

// PackageSpec represents a package name/version request.
type PackageSpec struct {
	Name    string
	Version string
}

// NewRepository creates a new Repository instance with the specified configuration.
func NewRepository(name, url, description, suite string, components, architectures []string) *Repository {
	return &Repository{
		Name:            name,
		URL:             url,
		Description:     description,
		Suite:           suite,
		Components:      components,
		Architectures:   architectures,
		VerifyRelease:   true,
		VerifySignature: true,
	}
}

func (r *Repository) downloader() *Downloader {
	return NewDownloader()
}

// FetchPackages fetches and parses Packages files from the repository.
// Returns a list of package names found across all configured sections and architectures.
func (r *Repository) FetchPackages() ([]string, error) {
	if r.VerifyRelease {
		if err := r.FetchReleaseFile(); err != nil {
			return nil, fmt.Errorf("error retrieving Release file: %w", err)
		}
	}

	// Reset metadata to avoid accumulation across multiple calls
	r.PackageMetadata = r.PackageMetadata[:0]

	allPackages := make(map[string]bool)
	var lastErr error
	foundAtLeastOne := false

	for _, component := range r.Components {
		for _, arch := range r.Architectures {
			packages, err := r.fetchPackagesForComponentArch(component, arch)
			if err != nil {
				if r.WarningHandler != nil {
					r.WarningHandler(fmt.Sprintf("Warning: unable to fetch packages for component '%s', architecture '%s': %v", component, arch, err))
				}
				lastErr = err
				continue
			}

			for _, pkg := range packages {
				allPackages[pkg] = true
			}
			foundAtLeastOne = true
		}
	}

	if !foundAtLeastOne {
		return nil, fmt.Errorf("unable to fetch packages from suite %s: %w", r.Suite, lastErr)
	}

	result := make([]string, 0, len(allPackages))
	for pkg := range allPackages {
		result = append(result, pkg)
	}

	r.Packages = result
	return result, nil
}

// FetchAndCachePackages downloads Packages metadata for all configured components and architectures
// and writes the decompressed files to the provided cache directory.
func (r *Repository) FetchAndCachePackages(cacheDir string) error {
	if cacheDir == "" {
		return fmt.Errorf("cache directory is required")
	}

	if r.VerifyRelease {
		if err := r.FetchReleaseFile(); err != nil {
			return fmt.Errorf("error retrieving Release file: %w", err)
		}
	}

	if err := os.MkdirAll(cacheDir, DirPermission); err != nil {
		return fmt.Errorf("unable to create cache directory: %w", err)
	}

	var lastErr error
	foundAtLeastOne := false

	for _, component := range r.Components {
		for _, arch := range r.Architectures {
			if err := r.cachePackagesForComponentArch(cacheDir, component, arch); err != nil {
				lastErr = err
				continue
			}
			foundAtLeastOne = true
		}
	}

	if !foundAtLeastOne {
		return fmt.Errorf("unable to cache packages from suite %s: %w", r.Suite, lastErr)
	}

	return nil
}

// FetchSources fetches and parses Sources files from the repository.
// Returns a list of source package names found across all configured components.
func (r *Repository) FetchSources() ([]string, error) {
	if r.VerifyRelease {
		if err := r.FetchReleaseFile(); err != nil {
			return nil, fmt.Errorf("error retrieving Release file: %w", err)
		}
	}

	allSources := make(map[string]bool)
	metadata := make([]SourcePackage, 0)

	var lastErr error
	foundAtLeastOne := false

	for _, component := range r.Components {
		sources, err := r.fetchSourcesForComponent(component)
		if err != nil {
			if r.WarningHandler != nil {
				r.WarningHandler(fmt.Sprintf("Warning: unable to fetch sources for component '%s': %v", component, err))
			}
			lastErr = err
			continue
		}

		for _, sp := range sources {
			metadata = append(metadata, sp)
			allSources[sp.Name] = true
		}

		foundAtLeastOne = true
	}

	if !foundAtLeastOne {
		return nil, fmt.Errorf("unable to fetch source packages from suite %s: %w", r.Suite, lastErr)
	}

	r.SourceMetadata = metadata

	result := make([]string, 0, len(allSources))
	for name := range allSources {
		result = append(result, name)
	}

	sort.Strings(result)
	return result, nil
}

func (r *Repository) fetchSourcesForComponent(component string) ([]SourcePackage, error) {
	var lastErr error

	for _, ext := range CompressionExtensions {
		sourcesURL := r.buildSourcesURL(r.Suite, component) + ext

		if !r.checkURLExists(sourcesURL) {
			lastErr = fmt.Errorf("Sources file not accessible: %s", sourcesURL)
			continue
		}

		var sources []SourcePackage
		var err error

		if ext == "" {
			sources, err = r.downloadAndParseSourcesWithVerification(sourcesURL, component)
		} else {
			sources, err = r.downloadAndParseCompressedSourcesWithVerification(sourcesURL, ext, component)
		}

		if err != nil {
			lastErr = err
			continue
		}

		return sources, nil
	}

	return nil, lastErr
}

// parseSourcesFromReader parses source metadata directly from an io.Reader.
func (r *Repository) parseSourcesFromReader(reader io.Reader, component string) ([]SourcePackage, error) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, packagesInitialAlloc)
	scanner.Buffer(buf, packagesBufferSize)

	var sources []SourcePackage
	var current *SourcePackage
	currentField := ""
	files := make(map[string]*SourceFile)

	finalize := func() {
		if current == nil {
			return
		}
		r.finalizeSourcePackage(current, files, component)
		sources = append(sources, *current)
		current = nil
		files = make(map[string]*SourceFile)
		currentField = ""
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			finalize()
			continue
		}

		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if current == nil {
				continue
			}

			switch currentField {
			case "files":
				r.parseSourceFileEntry(trimmedLine, files, "md5")
			case "checksums-sha256":
				r.parseSourceFileEntry(trimmedLine, files, "sha256")
			case "description":
				if current.Description == "" {
					current.Description = trimmedLine
				} else {
					current.Description = current.Description + " " + trimmedLine
				}
			}
			continue
		}

		colonIndex := strings.Index(trimmedLine, ":")
		if colonIndex == -1 {
			continue
		}

		field := strings.ToLower(strings.TrimSpace(trimmedLine[:colonIndex]))
		value := strings.TrimSpace(trimmedLine[colonIndex+1:])
		currentField = field

		if field == "package" {
			finalize()
			current = &SourcePackage{Name: value}
			files = make(map[string]*SourceFile)
			continue
		}

		if current == nil {
			continue
		}

		switch field {
		case "version":
			current.Version = value
		case "maintainer":
			current.Maintainer = value
		case "directory":
			current.Directory = strings.TrimSpace(value)
		case "description":
			current.Description = value
		case "files":
			if value != "" {
				r.parseSourceFileEntry(value, files, "md5")
			}
		case "checksums-sha256":
			if value != "" {
				r.parseSourceFileEntry(value, files, "sha256")
			}
		}
	}

	if current != nil {
		finalize()
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Sources file: %w", err)
	}

	return sources, nil
}

// Deprecated: use parseSourcesFromReader instead.
func (r *Repository) parseSourcesData(data []byte, component string) ([]SourcePackage, error) {
	return r.parseSourcesFromReader(bytes.NewReader(data), component)
}

func (r *Repository) downloadAndParseSourcesWithVerification(sourcesURL, component string) ([]SourcePackage, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, sourcesURL, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving Sources file: %w", err)
	}
	defer resp.Body.Close()

	if !r.VerifyRelease || r.ReleaseInfo == nil {
		return r.parseSourcesFromReader(resp.Body, component)
	}

	// For validation, buffering is acceptable for small sources files or required for full checksum
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading Sources file: %w", err)
	}

	if err = r.VerifySourcesFileChecksum(component, data); err != nil {
		return nil, fmt.Errorf("failed to verify checksum: %w", err)
	}

	return r.parseSourcesFromReader(bytes.NewReader(data), component)
}

func (r *Repository) downloadAndParseCompressedSourcesWithVerification(sourcesURL, extension, component string) ([]SourcePackage, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, sourcesURL, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving compressed Sources file: %w", err)
	}
	defer resp.Body.Close()

	reader, cleanup, err := r.createDecompressor(resp.Body, extension)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		// Use TeeReader for streaming checksum verification
		hasher := sha256.New()
		teeReader := io.TeeReader(reader, hasher)

		sources, err := r.parseSourcesFromReader(teeReader, component)
		if err != nil {
			return nil, err
		}

		actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
		filename := fmt.Sprintf("%s/source/Sources", component)

		found := false
		for _, checksum := range r.ReleaseInfo.SHA256 {
			if checksum.Filename == filename {
				found = true
				if actualHash != strings.ToLower(checksum.Hash) {
					return nil, fmt.Errorf("invalid sha256 checksum for %s. Expected: %s, Actual: %s", filename, checksum.Hash, actualHash)
				}
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("no SHA256 checksum found for %s (streaming verification requires SHA256)", filename)
		}

		return sources, nil
	}

	return r.parseSourcesFromReader(reader, component)
}

func (r *Repository) parseSourceFileEntry(line string, files map[string]*SourceFile, checksumType string) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return
	}

	hash := strings.ToLower(parts[0])
	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}
	name := parts[2]

	file, ok := files[name]
	if !ok {
		file = &SourceFile{Name: name, Type: detectSourceFileType(name)}
		files[name] = file
	} else if file.Type == "" {
		file.Type = detectSourceFileType(name)
	}

	if file.Size == 0 {
		file.Size = size
	}

	switch checksumType {
	case "md5":
		file.MD5Sum = hash
	case "sha256":
		file.SHA256Sum = hash
	}
}

func (r *Repository) finalizeSourcePackage(pkg *SourcePackage, files map[string]*SourceFile, component string) {
	if pkg == nil {
		return
	}

	if pkg.Directory == "" {
		pkg.Directory = r.buildSourceDirectory(component, pkg.Name)
	}

	baseURL := strings.TrimSuffix(r.URL, "/")
	dir := strings.TrimPrefix(pkg.Directory, "/")

	fileNames := make([]string, 0, len(files))
	for name := range files {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)

	for _, name := range fileNames {
		file := files[name]
		if file == nil {
			continue
		}

		if file.URL == "" {
			joined := path.Join(dir, file.Name)
			file.URL = fmt.Sprintf("%s/%s", baseURL, joined)
		}

		pkg.Files = append(pkg.Files, *file)
	}
}

func (r *Repository) buildSourceDirectory(section, packageName string) string {
	prefix := getPoolPrefix(packageName)
	return fmt.Sprintf("pool/%s/%s/%s", section, prefix, packageName)
}

func detectSourceFileType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".dsc"):
		return "dsc"
	case strings.Contains(filename, ".orig.tar"):
		return "orig"
	case strings.Contains(filename, ".debian.tar"):
		return "debian"
	default:
		return "file"
	}
}

func (r *Repository) cachePackagesForComponentArch(cacheDir, component, architecture string) error {
	var lastErr error

	for _, ext := range CompressionExtensions {
		packagesURL := r.buildPackagesURL(r.Suite, component, architecture) + ext

		if !r.checkURLExists(packagesURL) {
			lastErr = fmt.Errorf("Packages file not accessible: %s", packagesURL)
			continue
		}

		data, err := r.downloadPackagesData(packagesURL, ext, component, architecture)
		if err != nil {
			lastErr = err
			continue
		}

		targetDir := filepath.Join(cacheDir, r.Suite, component, fmt.Sprintf("binary-%s", architecture))
		if err := os.MkdirAll(targetDir, DirPermission); err != nil {
			return fmt.Errorf("unable to create cache directory: %w", err)
		}

		targetPath := filepath.Join(targetDir, "Packages")
		if err := os.WriteFile(targetPath, data, FilePermission); err != nil {
			return fmt.Errorf("error writing Packages cache: %w", err)
		}

		return nil
	}

	return lastErr
}

func (r *Repository) downloadPackagesData(packagesURL, extension, component, architecture string) ([]byte, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, packagesURL, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving Packages file: %w", err)
	}
	defer resp.Body.Close()

	if extension == "" {
		// For uncompressed files, we still buffer if verification is needed,
		// because we can't easily stream-verify AND save without T-reading to memory buffer or temp file.
		// Since cache functionality implies saving the file, io.ReadAll is appropriate here.
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading Packages file: %w", err)
		}

		if r.VerifyRelease && r.ReleaseInfo != nil {
			if err := r.VerifyPackagesFileChecksum(component, architecture, data); err != nil {
				return nil, fmt.Errorf("failed to verify checksum: %w", err)
			}
		}

		return data, nil
	}

	reader, cleanup, err := r.createDecompressor(resp.Body, extension)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// For caching, we need the decompressed data to write to disk.
	// We read all data since we must write it to file anyway.
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading decompressed Packages file: %w", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		filename := fmt.Sprintf("%s/binary-%s/Packages", component, architecture)
		// Try SHA256 first
		verified := false
		for _, checksum := range r.ReleaseInfo.SHA256 {
			if checksum.Filename == filename {
				if err := r.verifyDataChecksum(data, checksum.Hash, "sha256"); err != nil {
					return nil, err
				}
				verified = true
				break
			}
		}
		// Try MD5 if SHA256 not found
		if !verified {
			for _, checksum := range r.ReleaseInfo.MD5Sum {
				if checksum.Filename == filename {
					if err := r.verifyDataChecksum(data, checksum.Hash, "md5"); err != nil {
						return nil, err
					}
					verified = true
					break
				}
			}
		}

		if !verified {
			// If verification failed because no checksum was found
			return nil, fmt.Errorf("no checksum found for file %s", filename)
		}
	}

	return data, nil
}

// fetchPackagesForComponentArch tries to fetch Packages file for a specific component/arch combination.
func (r *Repository) fetchPackagesForComponentArch(component, arch string) ([]string, error) {
	var lastErr error

	for _, ext := range CompressionExtensions {
		packagesURL := r.buildPackagesURL(r.Suite, component, arch) + ext

		if !r.checkURLExists(packagesURL) {
			lastErr = fmt.Errorf("Packages file not accessible: %s", packagesURL)
			continue
		}

		var packages []string
		var err error

		if ext == "" {
			packages, err = r.downloadAndParsePackagesWithVerification(packagesURL, component, arch)
		} else {
			packages, err = r.downloadAndParseCompressedPackagesWithVerification(packagesURL, ext, component, arch)
		}

		if err != nil {
			lastErr = err
			continue
		}

		return packages, nil
	}

	return nil, lastErr
}

// checkURLExists performs a HEAD request to check if a URL is accessible.
func (r *Repository) checkURLExists(url string) bool {
	resp, err := r.downloader().doRequestWithRetry(http.MethodHead, url, true)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// SearchPackage searches for packages by name (exact and partial matches).
// Returns exact matches first, followed by partial matches.
func (r *Repository) SearchPackage(packageName string) ([]string, error) {
	if len(r.Packages) == 0 {
		return nil, fmt.Errorf("no packages available - call FetchPackages() first")
	}

	packageNameLower := strings.ToLower(packageName)
	var exactMatches, partialMatches []string

	for _, pkg := range r.Packages {
		pkgLower := strings.ToLower(pkg)
		switch {
		case pkg == packageName || pkgLower == packageNameLower:
			exactMatches = append(exactMatches, pkg)
		case strings.Contains(pkgLower, packageNameLower):
			partialMatches = append(partialMatches, pkg)
		}
	}

	if len(exactMatches) == 0 && len(partialMatches) == 0 {
		return nil, fmt.Errorf("no packages found for '%s' in suite %s", packageName, r.Suite)
	}

	result := make([]string, 0, len(exactMatches)+len(partialMatches))
	result = append(result, exactMatches...)
	result = append(result, partialMatches...)
	return result, nil
}

// DownloadPackage downloads a package by name, version, and architecture.
func (r *Repository) DownloadPackage(packageName, version, architecture, destDir string) error {
	pkg := r.buildPackageStruct(packageName, version, architecture, r.buildPackageURL(packageName, version, architecture))
	return NewDownloader().DownloadToDirSilent(pkg, destDir)
}

// DownloadPackageByURL downloads a package from a direct URL.
func (r *Repository) DownloadPackageByURL(packageURL, destDir string) error {
	parts := strings.Split(packageURL, "/")
	filename := parts[len(parts)-1]
	pkg := &Package{
		Name:        strings.Split(filename, "_")[0],
		DownloadURL: packageURL,
		Filename:    filename,
	}
	return NewDownloader().DownloadToDirSilent(pkg, destDir)
}

// buildPackageStruct creates a Package struct with the given parameters.
func (r *Repository) buildPackageStruct(name, version, architecture, downloadURL string) *Package {
	return &Package{
		Name:         name,
		Version:      version,
		Architecture: architecture,
		DownloadURL:  downloadURL,
		Filename:     fmt.Sprintf("%s_%s_%s.deb", name, version, architecture),
	}
}

// getPoolPrefix returns the pool directory prefix for a package name.
// For lib* packages, returns the first 4 characters; otherwise, the first character.
func getPoolPrefix(packageName string) string {
	if len(packageName) >= 4 && strings.HasPrefix(packageName, "lib") {
		return packageName[:4]
	}
	return string(packageName[0])
}

// buildPackageURL constructs the download URL for a package in the default component.
func (r *Repository) buildPackageURL(packageName, version, architecture string) string {
	return r.buildPackageURLWithComponent(packageName, version, architecture, "main")
}

// buildPackageURLWithComponent constructs the download URL for a package in a specific component.
func (r *Repository) buildPackageURLWithComponent(packageName, version, architecture, component string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	filename := fmt.Sprintf("%s_%s_%s.deb", packageName, version, architecture)
	prefix := getPoolPrefix(packageName)
	return fmt.Sprintf("%s/pool/%s/%s/%s/%s", baseURL, component, prefix, packageName, filename)
}

// CheckPackageAvailability checks if a package exists at the expected URL.
func (r *Repository) CheckPackageAvailability(packageName, version, architecture string) (bool, error) {
	return r.checkURLExists(r.buildPackageURL(packageName, version, architecture)), nil
}

// DownloadPackageFromSources tries to download a package from multiple components.
func (r *Repository) DownloadPackageFromSources(packageName, version, architecture, destDir string, components []string) error {
	if len(components) == 0 {
		components = defaultComponents
	}

	var lastErr error
	for _, component := range components {
		url := r.buildPackageURLWithComponent(packageName, version, architecture, component)

		if r.checkURLExists(url) {
			pkg := r.buildPackageStruct(packageName, version, architecture, url)
			return NewDownloader().DownloadToDirSilent(pkg, destDir)
		}

		lastErr = fmt.Errorf("package not found in component %s", component)
	}

	return fmt.Errorf("package %s_%s_%s not found in any component: %w", packageName, version, architecture, lastErr)
}

// SearchPackageInComponents searches for a package across all default components.
func (r *Repository) SearchPackageInComponents(packageName, version, architecture string) (*PackageInfo, error) {
	for _, component := range defaultComponents {
		url := r.buildPackageURLWithComponent(packageName, version, architecture, component)

		resp, err := r.downloader().doRequestWithRetry(http.MethodHead, url, true)
		if err != nil {
			continue
		}
		size := resp.ContentLength
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return &PackageInfo{
				Name:         packageName,
				Version:      version,
				Architecture: architecture,
				Section:      component, // "Section" refers to valid location, here it's the component
				DownloadURL:  url,
				Size:         size,
			}, nil
		}
	}

	return nil, fmt.Errorf("package %s_%s_%s not found", packageName, version, architecture)
}

// buildPackagesURL constructs the URL for a Packages file.
func (r *Repository) buildPackagesURL(suite, component, architecture string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", baseURL, suite, component, architecture)
}

// buildSourcesURL constructs the URL for a Sources file.
func (r *Repository) buildSourcesURL(suite, component string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/%s/source/Sources", baseURL, suite, component)
}

// EnableReleaseVerification enables checksum verification for downloaded files.
func (r *Repository) EnableReleaseVerification() {
	r.VerifyRelease = true
}

// DisableReleaseVerification disables checksum verification.
func (r *Repository) DisableReleaseVerification() {
	r.VerifyRelease = false
}

// EnableSignatureVerification enables GPG verification for Release/InRelease files.
func (r *Repository) EnableSignatureVerification() {
	r.VerifySignature = true
}

// DisableSignatureVerification disables GPG verification for Release/InRelease files.
func (r *Repository) DisableSignatureVerification() {
	r.VerifySignature = false
}

// SetKeyringPaths sets the keyring file paths used for signature verification.
// If paths is empty, it uses default system keyrings.
// Paths can be files or directories; directories are expanded to include all .gpg files.
func (r *Repository) SetKeyringPaths(paths []string) {
	r.KeyringPaths = resolveKeyringPaths(paths, nil)
}

// SetKeyringPathsWithDirs sets keyring paths from both explicit paths and directories.
// If both are empty, default system keyrings are used.
func (r *Repository) SetKeyringPathsWithDirs(paths, dirs []string) {
	r.KeyringPaths = resolveKeyringPaths(paths, dirs)
}

// resolveKeyringPaths resolves keyring paths from explicit paths and directories.
// If both are empty, it discovers default system keyrings.
func resolveKeyringPaths(paths, dirs []string) []string {
	var result []string

	// If no explicit paths or dirs provided, use defaults
	if len(paths) == 0 && len(dirs) == 0 {
		result = discoverDefaultKeyrings()
	} else {
		// Add explicit paths that exist
		for _, p := range paths {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				continue
			}
			info, err := os.Stat(trimmed)
			if err != nil {
				continue // File doesn't exist, skip
			}
			if info.IsDir() {
				// Expand directory to .gpg files
				result = append(result, expandKeyringDir(trimmed)...)
			} else {
				result = append(result, trimmed)
			}
		}

		// Expand explicit directories
		for _, d := range dirs {
			trimmed := strings.TrimSpace(d)
			if trimmed == "" {
				continue
			}
			result = append(result, expandKeyringDir(trimmed)...)
		}
	}

	return result
}

// discoverDefaultKeyrings finds keyrings in default system locations.
func discoverDefaultKeyrings() []string {
	var keyrings []string

	// Check default keyring files (OS-aware)
	for _, path := range GetDefaultKeyringPaths() {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			keyrings = append(keyrings, path)
		}
	}

	// Expand default keyring directories (OS-aware)
	for _, dir := range GetDefaultKeyringDirs() {
		keyrings = append(keyrings, expandKeyringDir(dir)...)
	}

	return keyrings
}

// expandKeyringDir expands a directory path to all .gpg files within it.
func expandKeyringDir(dir string) []string {
	var result []string
	pattern := filepath.Join(dir, "*.gpg")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return result
	}
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil && !info.IsDir() {
			result = append(result, match)
		}
	}
	return result
}

// ResolveKeyringPathsExternal is the exported version of resolveKeyringPaths
// for use by command packages that need to resolve keyrings before creating MirrorConfig.
func ResolveKeyringPathsExternal(paths, dirs []string) []string {
	return resolveKeyringPaths(paths, dirs)
}

// GetReleaseInfo returns the parsed Release file information.
func (r *Repository) GetReleaseInfo() *ReleaseFile {
	return r.ReleaseInfo
}

// IsReleaseVerificationEnabled returns whether checksum verification is enabled.
func (r *Repository) IsReleaseVerificationEnabled() bool {
	return r.VerifyRelease
}

// downloadAndParsePackagesWithVerification downloads and parses an uncompressed Packages file.
func (r *Repository) downloadAndParsePackagesWithVerification(packagesURL, component, architecture string) ([]string, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, packagesURL, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving Packages file: %w", err)
	}
	defer resp.Body.Close()

	if !r.VerifyRelease || r.ReleaseInfo == nil {
		// If no verification required, stream parse directly from response body
		packagedNames, metadata, err := r.parsePackagesFromReader(resp.Body)
		if err != nil {
			return nil, err
		}
		r.PackageMetadata = append(r.PackageMetadata, metadata...)
		return packagedNames, nil
	}

	// For verification, we need to read the whole content to verify checksum
	// Note: streaming verification with hash calculation is possible but complex
	// because we need to parse valid data only. Here we buffer to verify first.
	// Optimization: If memory is an issue, consider computing hash via TeeReader
	// but this risks parsing corrupted data before verification fails.
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading Packages file: %w", err)
	}

	if err = r.VerifyPackagesFileChecksum(component, architecture, data); err != nil {
		return nil, fmt.Errorf("failed to verify checksum: %w", err)
	}

	return r.parsePackagesData(data)
}

// downloadAndParseCompressedPackagesWithVerification downloads and parses a compressed Packages file.
func (r *Repository) downloadAndParseCompressedPackagesWithVerification(packagesURL, extension, component, architecture string) ([]string, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, packagesURL, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving compressed Packages file: %w", err)
	}
	defer resp.Body.Close()

	reader, cleanup, err := r.createDecompressor(resp.Body, extension)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Stream parsing with simultaneous checksum verification using TeeReader
	if r.VerifyRelease && r.ReleaseInfo != nil {
		// Optimization: Use TeeReader to compute hash while parsing to avoid loading
		// full decompressed file into memory. We use SHA256 by default.
		hasher := sha256.New()
		teeReader := io.TeeReader(reader, hasher)

		packagedNames, metadata, parseErr := r.parsePackagesFromReader(teeReader)
		if parseErr != nil {
			return nil, parseErr
		}

		// Verify checksum AFTER parsing is complete
		actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
		filename := fmt.Sprintf("%s/binary-%s/Packages", component, architecture)

		// Check against Release file SHA256 checksums
		found := false
		for _, checksum := range r.ReleaseInfo.SHA256 {
			if checksum.Filename == filename {
				found = true
				if actualHash != strings.ToLower(checksum.Hash) {
					return nil, fmt.Errorf("invalid sha256 checksum for %s. Expected: %s, Actual: %s", filename, checksum.Hash, actualHash)
				}
				break
			}
		}

		// Fallback to MD5 if SHA256 not found (unlikely for modern repos but possible)
		// Note: Since we hashed with SHA256, we can't verify MD5 here without a second pass or second hasher.
		// If SHA256 is missing from Release file, we fail securely or would need double-hashing.
		// For Debian repositories, SHA256 is standard.
		if !found {
			// If no SHA256 checksum found in Release file, we warn or fail.
			// To support MD5-only repos properly while streaming, we would need to MultiWriter both hashers.
			// Given modern standards, we enforce SHA256 availability for streaming optimization.
			// If you need MD5 support, fallback to buffering method.
			return nil, fmt.Errorf("no SHA256 checksum found for %s (streaming verification requires SHA256)", filename)
		}

		r.PackageMetadata = append(r.PackageMetadata, metadata...)
		return packagedNames, nil
	}

	// No verification needed, just stream parse
	packagedNames, metadata, err := r.parsePackagesFromReader(reader)
	if err != nil {
		return nil, err
	}

	r.PackageMetadata = append(r.PackageMetadata, metadata...)
	return packagedNames, nil
}

// createDecompressor creates a decompression reader based on the file extension.
// Returns the reader, a cleanup function (may be nil), and any error.
func (r *Repository) createDecompressor(body io.Reader, extension string) (io.Reader, func(), error) {
	switch extension {
	case ".gz":
		gzReader, err := gzip.NewReader(body)
		if err != nil {
			return nil, nil, fmt.Errorf("error during gzip decompression: %w", err)
		}
		return gzReader, func() { gzReader.Close() }, nil

	case ".xz":
		xzReader, err := xz.NewReader(body)
		if err != nil {
			return nil, nil, fmt.Errorf("error during xz decompression: %w", err)
		}
		return xzReader, nil, nil

	default:
		return nil, nil, fmt.Errorf("unsupported compression format: %s", extension)
	}
}

// parsePackagesFromReader parses package metadata directly from an io.Reader.
func (r *Repository) parsePackagesFromReader(reader io.Reader) ([]string, []Package, error) {
	var packages []string
	var packageMetadata []Package

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, packagesInitialAlloc)
	scanner.Buffer(buf, packagesBufferSize)

	var currentPackage *Package

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Empty line indicates end of current package block
		if trimmedLine == "" {
			if currentPackage != nil && currentPackage.Name != "" {
				r.finalizePackage(currentPackage)
				packageMetadata = append(packageMetadata, *currentPackage)
				packages = append(packages, currentPackage.Name)
			}
			currentPackage = nil
			continue
		}

		// Skip continuation lines (starting with space or tab)
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}

		// Parse field: value pairs
		colonIndex := strings.Index(trimmedLine, ":")
		if colonIndex == -1 {
			continue
		}

		field := strings.TrimSpace(trimmedLine[:colonIndex])
		value := strings.TrimSpace(trimmedLine[colonIndex+1:])

		// Start new package block
		if field == "Package" {
			currentPackage = &Package{
				Name:    value,
				Package: value,
			}
			continue
		}

		// Skip if no current package
		if currentPackage == nil {
			continue
		}

		// Parse field using mapping or special handling
		r.parsePackageField(currentPackage, field, value)
	}

	// Handle last package if file doesn't end with empty line
	if currentPackage != nil && currentPackage.Name != "" {
		r.finalizePackage(currentPackage)
		packageMetadata = append(packageMetadata, *currentPackage)
		packages = append(packages, currentPackage.Name)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading Packages file: %w", err)
	}

	return packages, packageMetadata, nil
}

// parsePackagesData parses package metadata from Packages file content.
// Deprecated: use parsePackagesFromReader instead.
func (r *Repository) parsePackagesData(data []byte) ([]string, error) {
	packagedNames, metadata, err := r.parsePackagesFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Accumulate metadata instead of replacing it
	r.PackageMetadata = append(r.PackageMetadata, metadata...)
	return packagedNames, nil
}

func (r *Repository) parsePackagesDataInternal(data []byte) ([]string, []Package, error) {
	return r.parsePackagesFromReader(bytes.NewReader(data))
}

// finalizePackage sets default values for a package before storing.
func (r *Repository) finalizePackage(pkg *Package) {
	if pkg.Source == "" {
		pkg.Source = pkg.Name
	}
}

// parsePackageField parses a single field from a Packages file entry.
func (r *Repository) parsePackageField(pkg *Package, field, value string) {
	fieldLower := strings.ToLower(field)

	// Use field mappings from package.go for standard fields
	if setter, ok := controlFieldMapping[fieldLower]; ok {
		setter(pkg, value)
		return
	}

	// Use dependency field mappings
	if setter, ok := dependencyFieldMapping[fieldLower]; ok {
		setter(pkg, parsePackageList(value))
		return
	}

	// Handle special fields not in the standard mappings
	switch field {
	case "Filename":
		pkg.Filename = value
		baseURL := strings.TrimSuffix(r.URL, "/")
		pkg.DownloadURL = fmt.Sprintf("%s/%s", baseURL, value)
	case "Size":
		if size, err := strconv.ParseInt(value, 10, 64); err == nil {
			pkg.Size = size
		}
	case "MD5sum":
		pkg.MD5sum = value
	case "SHA1":
		pkg.SHA1 = value
	case "SHA256":
		pkg.SHA256 = value
	default:
		// Custom fields (X- prefixed or unknown)
		if pkg.CustomFields == nil {
			pkg.CustomFields = make(map[string]string)
		}
		pkg.CustomFields[field] = value
	}
}

// verifyDecompressedFileChecksum verifies the checksum of decompressed file content.
func (r *Repository) verifyDecompressedFileChecksum(filename string, data []byte) error {
	for _, checksum := range r.ReleaseInfo.SHA256 {
		if checksum.Filename == filename {
			return r.verifyDataChecksum(data, checksum.Hash, "sha256")
		}
	}

	for _, checksum := range r.ReleaseInfo.MD5Sum {
		if checksum.Filename == filename {
			return r.verifyDataChecksum(data, checksum.Hash, "md5")
		}
	}

	return fmt.Errorf("no checksum found for file %s", filename)
}

// LoadCachedPackages loads Packages metadata from an existing cache directory
// without performing any network requests.
func (r *Repository) LoadCachedPackages(cacheDir string) ([]string, error) {
	if cacheDir == "" {
		return nil, fmt.Errorf("cache directory is required")
	}

	if r.Suite == "" {
		return nil, fmt.Errorf("suite is required to load cache")
	}

	allPackages := make(map[string]bool)
	metadata := make([]Package, 0)
	var lastErr error
	found := false

	for _, component := range r.Components {
		for _, arch := range r.Architectures {
			cachePath := filepath.Join(cacheDir, r.Suite, component, fmt.Sprintf("binary-%s", arch), "Packages")

			data, err := os.ReadFile(cachePath)
			if err != nil {
				if lastErr == nil {
					lastErr = err
				}
				continue
			}

			names, pkgMetadata, err := r.parsePackagesDataInternal(data)
			if err != nil {
				lastErr = err
				continue
			}

			for _, name := range names {
				allPackages[name] = true
			}
			metadata = append(metadata, pkgMetadata...)
			found = true
		}
	}

	if !found {
		if lastErr != nil {
			return nil, fmt.Errorf("no cached packages found for %s: %w", r.Suite, lastErr)
		}
		return nil, fmt.Errorf("no cached packages found for %s", r.Suite)
	}

	packages := make([]string, 0, len(allPackages))
	for name := range allPackages {
		packages = append(packages, name)
	}

	r.PackageMetadata = metadata
	r.Packages = packages

	return packages, nil
}

// SetSuite sets the active suite.
func (r *Repository) SetSuite(suite string) {
	r.Suite = suite
}

// SetComponents sets the active components.
func (r *Repository) SetComponents(components []string) {
	r.Components = components
}

// SetArchitectures sets the active architectures.
func (r *Repository) SetArchitectures(architectures []string) {
	r.Architectures = architectures
}

// AddComponent adds a component to the repository configuration.
func (r *Repository) AddComponent(component string) {
	r.Components = append(r.Components, component)
}

// AddArchitecture adds an architecture to the repository configuration.
func (r *Repository) AddArchitecture(architecture string) {
	r.Architectures = append(r.Architectures, architecture)
}

// GetPackageMetadata returns the metadata for a package without architecture preference.
func (r *Repository) GetPackageMetadata(packageName string) (*Package, error) {
	return r.GetPackageMetadataWithArch(packageName, "", nil)
}

// GetPackageMetadataWithArch returns package metadata honoring version (optional) and
// architecture order preference. The first matching architecture in archOrder is selected;
// when archOrder is empty, the repository architectures are used; when both are empty,
// the first match is returned.
func (r *Repository) GetPackageMetadataWithArch(packageName, version string, archOrder []string) (*Package, error) {
	if len(r.PackageMetadata) == 0 {
		return nil, fmt.Errorf("no package metadata available - call FetchPackages() first")
	}

	matches := make([]*Package, 0)
	for i := range r.PackageMetadata {
		p := &r.PackageMetadata[i]
		if p.Name != packageName {
			continue
		}
		if version != "" && p.Version != version {
			continue
		}
		matches = append(matches, p)
	}

	if len(matches) == 0 {
		if version == "" {
			return nil, fmt.Errorf("package '%s' not found in metadata", packageName)
		}
		return nil, fmt.Errorf("version %s not found for package %s", version, packageName)
	}

	order := archOrder
	if len(order) == 0 {
		order = r.Architectures
	}

	if len(order) == 0 {
		return matches[0], nil
	}

	best := matches[0]
	bestRank := len(order) + 1
	for _, p := range matches {
		rank := len(order) + 1
		for idx, arch := range order {
			if p.Architecture == arch {
				rank = idx
				break
			}
		}

		if rank < bestRank {
			best = p
			bestRank = rank
		}
	}

	return best, nil
}

// GetAllPackageMetadata returns all package metadata.
func (r *Repository) GetAllPackageMetadata() []Package {
	return r.PackageMetadata
}

// GetSourcePackageMetadata returns source package metadata, optionally filtered by version.
// When version is empty, the first matching entry is returned.
func (r *Repository) GetSourcePackageMetadata(packageName, version string) (*SourcePackage, error) {
	if len(r.SourceMetadata) == 0 {
		return nil, fmt.Errorf("no source package metadata available - call FetchSources() first")
	}

	for i := range r.SourceMetadata {
		sp := &r.SourceMetadata[i]
		if sp.Name != packageName {
			continue
		}

		if version == "" || sp.Version == version {
			return sp, nil
		}
	}

	if version == "" {
		return nil, fmt.Errorf("source package '%s' not found in metadata", packageName)
	}

	return nil, fmt.Errorf("version %s not found for source package %s", version, packageName)
}

// GetAllSourceMetadata returns all source package metadata.
func (r *Repository) GetAllSourceMetadata() []SourcePackage {
	return r.SourceMetadata
}

// ResolveDependencies returns all packages required for the given specs, following dependency
// relationships and excluding types listed in exclude map (keys lowercased: depends, pre-depends,
// recommends, suggests, enhances, breaks, conflicts, provides, replaces).
// Default behavior (exclude empty) mirrors apt: Depends + Pre-Depends + Recommends; other
// relationships are included unless explicitly excluded.
func (r *Repository) ResolveDependencies(specs []PackageSpec, exclude map[string]bool) (map[string]Package, error) {
	if len(r.PackageMetadata) == 0 {
		return nil, fmt.Errorf("no package metadata available - call FetchPackages() first")
	}

	index := make(map[string]*Package, len(r.PackageMetadata))
	for i := range r.PackageMetadata {
		p := &r.PackageMetadata[i]
		if _, exists := index[p.Name]; !exists {
			index[p.Name] = p
		}
	}

	result := make(map[string]Package)
	seen := make(map[string]bool)
	queue := make([]PackageSpec, 0, len(specs))
	queue = append(queue, specs...)

	for len(queue) > 0 {
		spec := queue[0]
		queue = queue[1:]

		name := strings.TrimSpace(spec.Name)
		if name == "" || seen[name] {
			continue
		}

		pkg := index[name]
		if pkg == nil {
			return nil, fmt.Errorf("package '%s' not found in metadata", name)
		}
		if spec.Version != "" && pkg.Version != spec.Version {
			return nil, fmt.Errorf("version %s not found for %s (found: %s)", spec.Version, name, pkg.Version)
		}

		result[name] = *pkg
		seen[name] = true

		deps := r.collectDependencies(pkg, exclude)
		for _, depExpr := range deps {
			depName := chooseAvailableAlternative(depExpr, index)
			if depName == "" || seen[depName] {
				continue
			}
			queue = append(queue, PackageSpec{Name: depName})
		}
	}

	return result, nil
}

func (r *Repository) collectDependencies(pkg *Package, exclude map[string]bool) []string {
	var deps []string
	add := func(kind string, items []string) {
		if exclude != nil && exclude[strings.ToLower(kind)] {
			return
		}
		deps = append(deps, items...)
	}

	// Align with apt-style resolution: hard deps only, optionals when not excluded.
	add("depends", pkg.Depends)
	add("pre-depends", pkg.PreDepends)
	add("recommends", pkg.Recommends) // apt installs Recommends by default
	add("suggests", pkg.Suggests)     // optional; can be excluded via flag
	add("enhances", pkg.Enhances)     // optional; can be excluded via flag

	return deps
}

// chooseAvailableAlternative returns the first available package name from an OR expression.
func chooseAvailableAlternative(expr string, index map[string]*Package) string {
	parts := strings.Split(expr, "|")
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if space := strings.IndexAny(candidate, " (<"); space > 0 {
			candidate = strings.TrimSpace(candidate[:space])
		}
		if candidate == "" {
			continue
		}
		if _, ok := index[candidate]; ok {
			return candidate
		}
	}
	return ""
}

// FetchReleaseFile downloads and parses the Release file from the repository.
func (r *Repository) FetchReleaseFile() error {
	var releaseData []byte
	var err error

	if r.VerifySignature {
		releaseData, err = r.fetchSignedRelease()
	} else {
		releaseData, err = r.fetchUnsignedRelease()
	}

	if err != nil {
		return err
	}

	releaseInfo, err := r.parseReleaseFile(string(releaseData))
	if err != nil {
		return fmt.Errorf("error parsing Release file: %w", err)
	}

	r.ReleaseInfo = releaseInfo
	return nil
}

// buildReleaseURL constructs the URL for the Release file.
func (r *Repository) buildReleaseURL() string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/Release", baseURL, r.Suite)
}

// buildInReleaseURL constructs the URL for the InRelease file.
func (r *Repository) buildInReleaseURL() string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/InRelease", baseURL, r.Suite)
}

// fetchUnsignedRelease downloads the Release file without signature verification.
func (r *Repository) fetchUnsignedRelease() ([]byte, error) {
	return r.fetchURL(r.buildReleaseURL())
}

// fetchSignedRelease downloads and verifies InRelease or Release+Release.gpg.
func (r *Repository) fetchSignedRelease() ([]byte, error) {
	// Prefer InRelease (clearsigned)
	inReleaseURL := r.buildInReleaseURL()
	inReleaseData, err := r.fetchURL(inReleaseURL)
	if err == nil {
		if err := r.verifyClearsigned(inReleaseData); err == nil {
			content, extractErr := extractClearsignedContent(inReleaseData)
			if extractErr != nil {
				return nil, extractErr
			}
			return content, nil
		}
	}

	// Fallback to Release + Release.gpg
	releaseURL := r.buildReleaseURL()
	releaseData, err := r.fetchURL(releaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Release file: %w", err)
	}

	signatureURL := releaseURL + ".gpg"
	signatureData, err := r.fetchURL(signatureURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Release.gpg: %w", err)
	}

	if err := r.verifyDetachedSignature(releaseData, signatureData); err != nil {
		return nil, err
	}

	return releaseData, nil
}

// parseReleaseFile parses the content of a Release file.
func (r *Repository) parseReleaseFile(content string) (*ReleaseFile, error) {
	release := &ReleaseFile{
		Architectures: make([]string, 0),
		Components:    make([]string, 0),
		MD5Sum:        make([]FileChecksum, 0),
		SHA1:          make([]FileChecksum, 0),
		SHA256:        make([]FileChecksum, 0),
	}

	lines := strings.Split(content, "\n")
	currentSection := ""

	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if line == "MD5Sum:" {
			currentSection = "MD5Sum"
			continue
		} else if line == "SHA1:" {
			currentSection = "SHA1"
			continue
		} else if line == "SHA256:" {
			currentSection = "SHA256"
			continue
		}

		if currentSection != "" && strings.HasPrefix(originalLine, " ") {
			checksum, err := r.parseChecksumLine(originalLine)
			if err != nil {
				continue // ignore malformed checksum lines
			}

			switch currentSection {
			case "MD5Sum":
				release.MD5Sum = append(release.MD5Sum, *checksum)
			case "SHA1":
				release.SHA1 = append(release.SHA1, *checksum)
			case "SHA256":
				release.SHA256 = append(release.SHA256, *checksum)
			}
			continue
		}

		// Dectection of new section
		if !strings.HasPrefix(originalLine, " ") && currentSection != "" {
			currentSection = ""
		}

		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "Origin":
				release.Origin = value
			case "Label":
				release.Label = value
			case "Suite":
				release.Suite = value
			case "Version":
				release.Version = value
			case "Codename":
				release.Codename = value
			case "Date":
				release.Date = value
			case "Description":
				release.Description = value
			case "Architectures":
				release.Architectures = strings.Fields(value)
			case "Components":
				release.Components = strings.Fields(value)
			}
		}
	}

	return release, nil
}

func (r *Repository) fetchURL(url string) ([]byte, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, url, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving %s: %w", url, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", url, err)
	}

	return data, nil
}

func (r *Repository) verifyClearsigned(data []byte) error {
	return r.verifyWithGPG(data, nil, true)
}

func (r *Repository) verifyDetachedSignature(payload, signature []byte) error {
	return r.verifyWithGPG(payload, signature, false)
}

func (r *Repository) verifyWithGPG(payload, signature []byte, clearsigned bool) error {
	// Get gpgv executable (OS-aware, returns error on Windows if not found)
	gpgvPath, err := getGPGVCommand()
	if err != nil {
		return err
	}

	releaseFile, err := os.CreateTemp("", "deb-release-*.txt")
	if err != nil {
		return fmt.Errorf("unable to create temp file for release: %w", err)
	}
	defer os.Remove(releaseFile.Name())

	if err := os.WriteFile(releaseFile.Name(), payload, FilePermission); err != nil {
		return fmt.Errorf("unable to write release data: %w", err)
	}

	var signatureFile string
	if !clearsigned {
		sig, err := os.CreateTemp("", "deb-release-sig-*.gpg")
		if err != nil {
			return fmt.Errorf("unable to create temp signature file: %w", err)
		}
		defer os.Remove(sig.Name())

		if err := os.WriteFile(sig.Name(), signature, FilePermission); err != nil {
			return fmt.Errorf("unable to write signature data: %w", err)
		}

		signatureFile = sig.Name()
	}

	args := []string{"--status-fd", "1"}
	for _, keyring := range r.KeyringPaths {
		trimmed := strings.TrimSpace(keyring)
		if trimmed != "" {
			args = append(args, "--keyring", trimmed)
		}
	}

	if clearsigned {
		args = append(args, releaseFile.Name())
	} else {
		args = append(args, signatureFile, releaseFile.Name())
	}

	cmd := exec.Command(gpgvPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gpg verification failed: %w: %s", err, string(output))
	}

	return nil
}

func extractClearsignedContent(data []byte) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	var content strings.Builder
	started := false

	for _, line := range lines {
		if strings.HasPrefix(line, "-----BEGIN PGP SIGNATURE-----") {
			break
		}

		if !started {
			if line == "" {
				started = true
			}
			continue
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	result := strings.TrimSpace(content.String())
	if result == "" {
		return nil, fmt.Errorf("unable to extract clearsigned content")
	}

	return []byte(result + "\n"), nil
}

// parseChecksumLine parses a single checksum line from the Release file.
// Format: <hash> <size> <filename>
func (r *Repository) parseChecksumLine(line string) (*FileChecksum, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, fmt.Errorf("malformed checksum line: %s", line)
	}

	hash := fields[0]
	sizeStr := fields[1]
	filename := fields[2]

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid size in checksum line: %w", err)
	}

	return &FileChecksum{
		Hash:     hash,
		Size:     size,
		Filename: filename,
	}, nil
}

// VerifyPackagesFileChecksum verifies the checksum of a Packages file against
// the checksums in the Release file. It prefers SHA256 over MD5.
func (r *Repository) VerifyPackagesFileChecksum(section, architecture string, data []byte) error {
	if r.ReleaseInfo == nil {
		return fmt.Errorf("Release information unavailable - call FetchReleaseFile() first")
	}

	filename := fmt.Sprintf("%s/binary-%s/Packages", section, architecture)

	// Prefer SHA256 over MD5
	for _, checksum := range r.ReleaseInfo.SHA256 {
		if checksum.Filename == filename {
			return r.verifyDataChecksum(data, checksum.Hash, "sha256")
		}
	}

	for _, checksum := range r.ReleaseInfo.MD5Sum {
		if checksum.Filename == filename {
			return r.verifyDataChecksum(data, checksum.Hash, "md5")
		}
	}

	return fmt.Errorf("no checksum found for file %s", filename)
}

// VerifySourcesFileChecksum verifies the checksum of a Sources file against
// the checksums in the Release file. It prefers SHA256 over MD5.
func (r *Repository) VerifySourcesFileChecksum(section string, data []byte) error {
	if r.ReleaseInfo == nil {
		return fmt.Errorf("Release information unavailable - call FetchReleaseFile() first")
	}

	filename := fmt.Sprintf("%s/source/Sources", section)

	for _, checksum := range r.ReleaseInfo.SHA256 {
		if checksum.Filename == filename {
			return r.verifyDataChecksum(data, checksum.Hash, "sha256")
		}
	}

	for _, checksum := range r.ReleaseInfo.MD5Sum {
		if checksum.Filename == filename {
			return r.verifyDataChecksum(data, checksum.Hash, "md5")
		}
	}

	return fmt.Errorf("no checksum found for file %s", filename)
}

// verifyDataChecksum computes and verifies a checksum against expected value.
func (r *Repository) verifyDataChecksum(data []byte, expectedHash, hashType string) error {
	var hasher hash.Hash

	switch strings.ToLower(hashType) {
	case "md5":
		hasher = md5.New()
	case "sha256":
		hasher = sha256.New()
	default:
		return fmt.Errorf("unsupported hash type: %s", hashType)
	}

	hasher.Write(data)
	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))

	if actualHash != strings.ToLower(expectedHash) {
		return fmt.Errorf("invalid %s checksum. Expected: %s, Actual: %s", hashType, expectedHash, actualHash)
	}

	return nil
}
