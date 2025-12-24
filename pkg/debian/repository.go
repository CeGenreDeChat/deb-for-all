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

// Default repository sections for package search.
var defaultSections = []string{"main", "contrib", "non-free"}

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
	Distribution    string
	Sections        []string
	Architectures   []string
	Packages        []string
	PackageMetadata []Package
	SourceMetadata  []SourcePackage
	ReleaseInfo     *ReleaseFile
	VerifyRelease   bool
	VerifySignature bool
	KeyringPaths    []string
}

// PackageSpec represents a package name/version request.
type PackageSpec struct {
	Name    string
	Version string
}

// NewRepository creates a new Repository instance with the specified configuration.
func NewRepository(name, url, description, distribution string, sections, architectures []string) *Repository {
	return &Repository{
		Name:            name,
		URL:             url,
		Description:     description,
		Distribution:    distribution,
		Sections:        sections,
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

	allPackages := make(map[string]bool)
	var lastErr error
	foundAtLeastOne := false

	for _, section := range r.Sections {
		for _, arch := range r.Architectures {
			packages, err := r.fetchPackagesForSectionArch(section, arch)
			if err != nil {
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
		return nil, fmt.Errorf("unable to fetch packages from distribution %s: %w", r.Distribution, lastErr)
	}

	result := make([]string, 0, len(allPackages))
	for pkg := range allPackages {
		result = append(result, pkg)
	}

	r.Packages = result
	return result, nil
}

// FetchAndCachePackages downloads Packages metadata for all configured sections and architectures
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

	for _, section := range r.Sections {
		for _, arch := range r.Architectures {
			if err := r.cachePackagesForSectionArch(cacheDir, section, arch); err != nil {
				lastErr = err
				continue
			}
			foundAtLeastOne = true
		}
	}

	if !foundAtLeastOne {
		return fmt.Errorf("unable to cache packages from distribution %s: %w", r.Distribution, lastErr)
	}

	return nil
}

// FetchSources fetches and parses Sources files from the repository.
// Returns a list of source package names found across all configured sections.
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

	for _, section := range r.Sections {
		sources, err := r.fetchSourcesForSection(section)
		if err != nil {
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
		return nil, fmt.Errorf("unable to fetch source packages from distribution %s: %w", r.Distribution, lastErr)
	}

	r.SourceMetadata = metadata

	result := make([]string, 0, len(allSources))
	for name := range allSources {
		result = append(result, name)
	}

	sort.Strings(result)
	return result, nil
}

func (r *Repository) fetchSourcesForSection(section string) ([]SourcePackage, error) {
	var lastErr error

	for _, ext := range CompressionExtensions {
		sourcesURL := r.buildSourcesURLWithDist(r.Distribution, section) + ext

		if !r.checkURLExists(sourcesURL) {
			lastErr = fmt.Errorf("Sources file not accessible: %s", sourcesURL)
			continue
		}

		var sources []SourcePackage
		var err error

		if ext == "" {
			sources, err = r.downloadAndParseSourcesWithVerification(sourcesURL, section)
		} else {
			sources, err = r.downloadAndParseCompressedSourcesWithVerification(sourcesURL, ext, section)
		}

		if err != nil {
			lastErr = err
			continue
		}

		return sources, nil
	}

	return nil, lastErr
}

func (r *Repository) downloadAndParseSourcesWithVerification(sourcesURL, section string) ([]SourcePackage, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, sourcesURL, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving Sources file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading Sources file: %w", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		if err = r.VerifySourcesFileChecksum(section, data); err != nil {
			return nil, fmt.Errorf("failed to verify checksum: %w", err)
		}
	}

	return r.parseSourcesData(data, section)
}

func (r *Repository) downloadAndParseCompressedSourcesWithVerification(sourcesURL, extension, section string) ([]SourcePackage, error) {
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

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading decompressed Sources file: %w", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		filename := fmt.Sprintf("%s/source/Sources", section)
		if err = r.verifyDecompressedFileChecksum(filename, data); err != nil {
			return nil, fmt.Errorf("failed to verify decompressed checksum: %w", err)
		}
	}

	return r.parseSourcesData(data, section)
}

func (r *Repository) parseSourcesData(data []byte, section string) ([]SourcePackage, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
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
		r.finalizeSourcePackage(current, files, section)
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

func (r *Repository) finalizeSourcePackage(pkg *SourcePackage, files map[string]*SourceFile, section string) {
	if pkg == nil {
		return
	}

	if pkg.Directory == "" {
		pkg.Directory = r.buildSourceDirectory(section, pkg.Name)
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

func (r *Repository) cachePackagesForSectionArch(cacheDir, section, architecture string) error {
	var lastErr error

	for _, ext := range CompressionExtensions {
		packagesURL := r.buildPackagesURLWithDist(r.Distribution, section, architecture) + ext

		if !r.checkURLExists(packagesURL) {
			lastErr = fmt.Errorf("Packages file not accessible: %s", packagesURL)
			continue
		}

		data, err := r.downloadPackagesData(packagesURL, ext, section, architecture)
		if err != nil {
			lastErr = err
			continue
		}

		targetDir := filepath.Join(cacheDir, r.Distribution, section, fmt.Sprintf("binary-%s", architecture))
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

func (r *Repository) downloadPackagesData(packagesURL, extension, section, architecture string) ([]byte, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, packagesURL, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving Packages file: %w", err)
	}
	defer resp.Body.Close()

	if extension == "" {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading Packages file: %w", err)
		}

		if r.VerifyRelease && r.ReleaseInfo != nil {
			if err := r.VerifyPackagesFileChecksum(section, architecture, data); err != nil {
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

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading decompressed Packages file: %w", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		filename := fmt.Sprintf("%s/binary-%s/Packages", section, architecture)
		if err := r.verifyDecompressedFileChecksum(filename, data); err != nil {
			return nil, fmt.Errorf("failed to verify decompressed checksum: %w", err)
		}
	}

	return data, nil
}

// fetchPackagesForSectionArch tries to fetch Packages file for a specific section/arch combination.
func (r *Repository) fetchPackagesForSectionArch(section, arch string) ([]string, error) {
	var lastErr error

	for _, ext := range CompressionExtensions {
		packagesURL := r.buildPackagesURLWithDist(r.Distribution, section, arch) + ext

		if !r.checkURLExists(packagesURL) {
			lastErr = fmt.Errorf("Packages file not accessible: %s", packagesURL)
			continue
		}

		var packages []string
		var err error

		if ext == "" {
			packages, err = r.downloadAndParsePackagesWithVerification(packagesURL, section, arch)
		} else {
			packages, err = r.downloadAndParseCompressedPackagesWithVerification(packagesURL, ext, section, arch)
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
		return nil, fmt.Errorf("no packages found for '%s' in distribution %s", packageName, r.Distribution)
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

// buildPackageURL constructs the download URL for a package in the default section.
func (r *Repository) buildPackageURL(packageName, version, architecture string) string {
	return r.buildPackageURLWithSection(packageName, version, architecture, "main")
}

// buildPackageURLWithSection constructs the download URL for a package in a specific section.
func (r *Repository) buildPackageURLWithSection(packageName, version, architecture, section string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	filename := fmt.Sprintf("%s_%s_%s.deb", packageName, version, architecture)
	prefix := getPoolPrefix(packageName)
	return fmt.Sprintf("%s/pool/%s/%s/%s/%s", baseURL, section, prefix, packageName, filename)
}

// CheckPackageAvailability checks if a package exists at the expected URL.
func (r *Repository) CheckPackageAvailability(packageName, version, architecture string) (bool, error) {
	return r.checkURLExists(r.buildPackageURL(packageName, version, architecture)), nil
}

// DownloadPackageFromSources tries to download a package from multiple sections.
func (r *Repository) DownloadPackageFromSources(packageName, version, architecture, destDir string, sections []string) error {
	if len(sections) == 0 {
		sections = defaultSections
	}

	var lastErr error
	for _, section := range sections {
		url := r.buildPackageURLWithSection(packageName, version, architecture, section)

		if r.checkURLExists(url) {
			pkg := r.buildPackageStruct(packageName, version, architecture, url)
			return NewDownloader().DownloadToDirSilent(pkg, destDir)
		}

		lastErr = fmt.Errorf("package not found in section %s", section)
	}

	return fmt.Errorf("package %s_%s_%s not found in any section: %w", packageName, version, architecture, lastErr)
}

// SearchPackageInSources searches for a package across all default sections.
func (r *Repository) SearchPackageInSources(packageName, version, architecture string) (*PackageInfo, error) {
	for _, section := range defaultSections {
		url := r.buildPackageURLWithSection(packageName, version, architecture, section)

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
				Section:      section,
				DownloadURL:  url,
				Size:         size,
			}, nil
		}
	}

	return nil, fmt.Errorf("package %s_%s_%s not found", packageName, version, architecture)
}

// buildPackagesURLWithDist constructs the URL for a Packages file.
func (r *Repository) buildPackagesURLWithDist(distribution, section, architecture string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", baseURL, distribution, section, architecture)
}

// buildSourcesURLWithDist constructs the URL for a Sources file.
func (r *Repository) buildSourcesURLWithDist(distribution, section string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/%s/source/Sources", baseURL, distribution, section)
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
func (r *Repository) SetKeyringPaths(paths []string) {
	r.KeyringPaths = paths
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
func (r *Repository) downloadAndParsePackagesWithVerification(packagesURL, section, architecture string) ([]string, error) {
	resp, err := r.downloader().doRequestWithRetry(http.MethodGet, packagesURL, true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving Packages file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading Packages file: %w", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		if err = r.VerifyPackagesFileChecksum(section, architecture, data); err != nil {
			return nil, fmt.Errorf("failed to verify checksum: %w", err)
		}
	}

	return r.parsePackagesData(data)
}

// downloadAndParseCompressedPackagesWithVerification downloads and parses a compressed Packages file.
func (r *Repository) downloadAndParseCompressedPackagesWithVerification(packagesURL, extension, section, architecture string) ([]string, error) {
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

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading decompressed Packages file: %w", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		filename := fmt.Sprintf("%s/binary-%s/Packages", section, architecture)
		if err = r.verifyDecompressedFileChecksum(filename, data); err != nil {
			return nil, fmt.Errorf("failed to verify decompressed checksum: %w", err)
		}
	}

	return r.parsePackagesData(data)
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

// parsePackagesData parses package metadata from Packages file content.
func (r *Repository) parsePackagesData(data []byte) ([]string, error) {
	packagedNames, metadata, err := r.parsePackagesDataInternal(data)
	if err != nil {
		return nil, err
	}

	r.PackageMetadata = metadata
	return packagedNames, nil
}

func (r *Repository) parsePackagesDataInternal(data []byte) ([]string, []Package, error) {
	var packages []string
	var packageMetadata []Package

	scanner := bufio.NewScanner(bytes.NewReader(data))
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

	if r.Distribution == "" {
		return nil, fmt.Errorf("distribution is required to load cache")
	}

	allPackages := make(map[string]bool)
	metadata := make([]Package, 0)
	var lastErr error
	found := false

	for _, section := range r.Sections {
		for _, arch := range r.Architectures {
			cachePath := filepath.Join(cacheDir, r.Distribution, section, fmt.Sprintf("binary-%s", arch), "Packages")

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
			return nil, fmt.Errorf("no cached packages found for %s: %w", r.Distribution, lastErr)
		}
		return nil, fmt.Errorf("no cached packages found for %s", r.Distribution)
	}

	packages := make([]string, 0, len(allPackages))
	for name := range allPackages {
		packages = append(packages, name)
	}

	r.PackageMetadata = metadata
	r.Packages = packages

	return packages, nil
}

// SetDistribution sets the active distribution (suite).
func (r *Repository) SetDistribution(distribution string) {
	r.Distribution = distribution
}

// SetSections sets the active sections (components).
func (r *Repository) SetSections(sections []string) {
	r.Sections = sections
}

// SetArchitectures sets the active architectures.
func (r *Repository) SetArchitectures(architectures []string) {
	r.Architectures = architectures
}

// AddSection adds a section to the repository configuration.
func (r *Repository) AddSection(section string) {
	r.Sections = append(r.Sections, section)
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
	return fmt.Sprintf("%s/dists/%s/Release", baseURL, r.Distribution)
}

// buildInReleaseURL constructs the URL for the InRelease file.
func (r *Repository) buildInReleaseURL() string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/InRelease", baseURL, r.Distribution)
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

	cmd := exec.Command("gpgv", args...)
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
