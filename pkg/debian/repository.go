package debian

import (
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
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
	ReleaseInfo     *ReleaseFile
	VerifyRelease   bool
}

// NewRepository creates a new Repository instance with the specified configuration.
func NewRepository(name, url, description, distribution string, sections, architectures []string) *Repository {
	return &Repository{
		Name:          name,
		URL:           url,
		Description:   description,
		Distribution:  distribution,
		Sections:      sections,
		Architectures: architectures,
		VerifyRelease: true,
	}
}

// FetchPackages fetches and parses Packages files from the repository.
// Returns a list of package names found across all configured sections and architectures.
func (r *Repository) FetchPackages() ([]string, error) {
	if r.VerifyRelease {
		if err := r.FetchReleaseFile(); err != nil {
			return nil, fmt.Errorf("erreur lors de la récupération du fichier Release: %w", err)
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
		return nil, fmt.Errorf("impossible de récupérer les paquets depuis la distribution %s: %w", r.Distribution, lastErr)
	}

	result := make([]string, 0, len(allPackages))
	for pkg := range allPackages {
		result = append(result, pkg)
	}

	r.Packages = result
	return result, nil
}

// fetchPackagesForSectionArch tries to fetch Packages file for a specific section/arch combination.
func (r *Repository) fetchPackagesForSectionArch(section, arch string) ([]string, error) {
	var lastErr error

	for _, ext := range CompressionExtensions {
		packagesURL := r.buildPackagesURLWithDist(r.Distribution, section, arch) + ext

		if !r.checkURLExists(packagesURL) {
			lastErr = fmt.Errorf("fichier Packages non accessible: %s", packagesURL)
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
	resp, err := http.Head(url)
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
		return nil, fmt.Errorf("aucun paquet disponible - appelez d'abord FetchPackages()")
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
		return nil, fmt.Errorf("aucun paquet trouvé pour '%s' dans la distribution %s", packageName, r.Distribution)
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

		lastErr = fmt.Errorf("paquet non trouvé dans la section %s", section)
	}

	return fmt.Errorf("paquet %s_%s_%s non trouvé dans aucune section: %w", packageName, version, architecture, lastErr)
}

// SearchPackageInSources searches for a package across all default sections.
func (r *Repository) SearchPackageInSources(packageName, version, architecture string) (*PackageInfo, error) {
	for _, section := range defaultSections {
		url := r.buildPackageURLWithSection(packageName, version, architecture, section)

		resp, err := http.Head(url)
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

	return nil, fmt.Errorf("paquet %s_%s_%s non trouvé", packageName, version, architecture)
}

// buildPackagesURLWithDist constructs the URL for a Packages file.
func (r *Repository) buildPackagesURLWithDist(distribution, section, architecture string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", baseURL, distribution, section, architecture)
}

// EnableReleaseVerification enables checksum verification for downloaded files.
func (r *Repository) EnableReleaseVerification() {
	r.VerifyRelease = true
}

// DisableReleaseVerification disables checksum verification.
func (r *Repository) DisableReleaseVerification() {
	r.VerifyRelease = false
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
	resp, err := http.Get(packagesURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération du fichier Packages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("impossible de récupérer le fichier Packages (HTTP %d)", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du fichier Packages: %w", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		if err = r.VerifyPackagesFileChecksum(section, architecture, data); err != nil {
			return nil, fmt.Errorf("échec de la vérification du checksum: %w", err)
		}
	}

	return r.parsePackagesData(data)
}

// downloadAndParseCompressedPackagesWithVerification downloads and parses a compressed Packages file.
func (r *Repository) downloadAndParseCompressedPackagesWithVerification(packagesURL, extension, section, architecture string) ([]string, error) {
	resp, err := http.Get(packagesURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération du fichier Packages compressé: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("impossible de récupérer le fichier Packages compressé (HTTP %d)", resp.StatusCode)
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
		return nil, fmt.Errorf("erreur lors de la lecture du fichier Packages décompressé: %w", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		filename := fmt.Sprintf("%s/binary-%s/Packages", section, architecture)
		if err = r.verifyDecompressedFileChecksum(filename, data); err != nil {
			return nil, fmt.Errorf("échec de la vérification du checksum décompressé: %w", err)
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
			return nil, nil, fmt.Errorf("erreur lors de la décompression gzip: %w", err)
		}
		return gzReader, func() { gzReader.Close() }, nil

	case ".xz":
		xzReader, err := xz.NewReader(body)
		if err != nil {
			return nil, nil, fmt.Errorf("erreur lors de la décompression xz: %w", err)
		}
		return xzReader, nil, nil

	default:
		return nil, nil, fmt.Errorf("format de compression non supporté: %s", extension)
	}
}

// parsePackagesData parses package metadata from Packages file content.
func (r *Repository) parsePackagesData(data []byte) ([]string, error) {
	var packages []string
	var packageMetadata []Package

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
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
		return nil, fmt.Errorf("erreur lors de la lecture du fichier Packages: %w", err)
	}

	r.PackageMetadata = packageMetadata
	return packages, nil
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

	return fmt.Errorf("aucun checksum trouvé pour le fichier %s", filename)
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

// GetPackageMetadata returns the complete metadata for a specific package.
func (r *Repository) GetPackageMetadata(packageName string) (*Package, error) {
	if len(r.PackageMetadata) == 0 {
		return nil, fmt.Errorf("aucune métadonnée de paquet disponible - appelez d'abord FetchPackages()")
	}

	for i := range r.PackageMetadata {
		if r.PackageMetadata[i].Name == packageName {
			return &r.PackageMetadata[i], nil
		}
	}

	return nil, fmt.Errorf("paquet '%s' non trouvé dans les métadonnées", packageName)
}

// GetAllPackageMetadata returns all package metadata.
func (r *Repository) GetAllPackageMetadata() []Package {
	return r.PackageMetadata
}

// FetchReleaseFile downloads and parses the Release file from the repository.
func (r *Repository) FetchReleaseFile() error {
	releaseURL := r.buildReleaseURL()

	resp, err := http.Get(releaseURL)
	if err != nil {
		return fmt.Errorf("erreur lors de la récupération du fichier Release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("impossible de récupérer le fichier Release (HTTP %d)", resp.StatusCode)
	}

	releaseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erreur lors de la lecture du fichier Release: %w", err)
	}

	releaseInfo, err := r.parseReleaseFile(string(releaseData))
	if err != nil {
		return fmt.Errorf("erreur lors du parsing du fichier Release: %w", err)
	}

	r.ReleaseInfo = releaseInfo
	return nil
}

// buildReleaseURL constructs the URL for the Release file.
func (r *Repository) buildReleaseURL() string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/Release", baseURL, r.Distribution)
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

// parseChecksumLine parses a single checksum line from the Release file.
// Format: <hash> <size> <filename>
func (r *Repository) parseChecksumLine(line string) (*FileChecksum, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, fmt.Errorf("ligne de checksum malformée: %s", line)
	}

	hash := fields[0]
	sizeStr := fields[1]
	filename := fields[2]

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("taille invalide dans la ligne de checksum: %w", err)
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
		return fmt.Errorf("informations Release non disponibles - appelez d'abord FetchReleaseFile()")
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

	return fmt.Errorf("aucun checksum trouvé pour le fichier %s", filename)
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
		return fmt.Errorf("type de hash non supporté: %s", hashType)
	}

	hasher.Write(data)
	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))

	if actualHash != strings.ToLower(expectedHash) {
		return fmt.Errorf("checksum %s invalide. Attendu: %s, Actuel: %s", hashType, expectedHash, actualHash)
	}

	return nil
}
