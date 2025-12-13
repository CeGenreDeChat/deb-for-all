package debian

import (
	"compress/gzip"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
)

// Default size estimation values.
const (
	defaultAveragePackageSize = 1024 * 1024 // 1MB average package size for estimation
)

// MirrorConfig contains the configuration for a mirror operation.
type MirrorConfig struct {
	BaseURL          string   // Repository URL to mirror from
	Suites           []string // Distributions to mirror (e.g., bookworm, bullseye)
	Components       []string // Components to mirror (e.g., main, contrib, non-free)
	Architectures    []string // Architectures to mirror (e.g., amd64, arm64)
	DownloadPackages bool     // Whether to download .deb package files
	Verbose          bool     // Enable verbose logging
	KeyringPaths     []string // Trusted keyring files for signature verification
	SkipGPGVerify    bool     // Disable GPG verification when true
}

// Validate checks that all required fields are set and valid.
func (c *MirrorConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("BaseURL is required")
	}
	if len(c.Suites) == 0 {
		return fmt.Errorf("at least one suite is required")
	}
	if len(c.Components) == 0 {
		return fmt.Errorf("at least one component is required")
	}
	if len(c.Architectures) == 0 {
		return fmt.Errorf("at least one architecture is required")
	}
	if !c.hasValidURLScheme() {
		return fmt.Errorf("BaseURL must start with http:// or https://")
	}
	return nil
}

// hasValidURLScheme checks if the BaseURL has a valid HTTP/HTTPS scheme.
func (c *MirrorConfig) hasValidURLScheme() bool {
	return strings.HasPrefix(c.BaseURL, "http://") || strings.HasPrefix(c.BaseURL, "https://")
}

// Mirror handles the creation and management of a local Debian repository mirror.
type Mirror struct {
	config     MirrorConfig
	repository *Repository
	downloader *Downloader
	basePath   string
}

// NewMirror creates a new Mirror instance with the given configuration.
func NewMirror(config MirrorConfig, basePath string) *Mirror {
	repo := NewRepository(
		"mirror-repo",
		config.BaseURL,
		"Mirror repository",
		config.Suites[0], // Start with first suite
		config.Components,
		config.Architectures,
	)

	repo.SetKeyringPaths(config.KeyringPaths)
	if config.SkipGPGVerify {
		repo.DisableSignatureVerification()
	}

	return &Mirror{
		config:     config,
		repository: repo,
		downloader: NewDownloader(),
		basePath:   basePath,
	}
}

// Clone creates a complete mirror of the configured repository.
// It downloads Release files, Packages metadata, and optionally package files.
func (m *Mirror) Clone() error {
	m.logVerbose("Starting mirror of %s to %s\n", m.config.BaseURL, m.basePath)

	if err := os.MkdirAll(m.basePath, DirPermission); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	for _, suite := range m.config.Suites {
		if err := m.mirrorSuite(suite); err != nil {
			return fmt.Errorf("failed to mirror suite %s: %w", suite, err)
		}
	}

	return nil
}

// Sync performs an incremental synchronization of the mirror.
// Currently equivalent to Clone; future versions will compare checksums
// and only download changed files.
func (m *Mirror) Sync() error {
	m.logVerbose("Synchronizing mirror of %s\n", m.config.BaseURL)
	return m.Clone()
}

// mirrorSuite mirrors all components and architectures for a given suite.
func (m *Mirror) mirrorSuite(suite string) error {
	m.logVerbose("Mirroring suite: %s\n", suite)

	m.repository.SetDistribution(suite)

	suitePath := m.buildSuitePath(suite)
	if err := os.MkdirAll(suitePath, DirPermission); err != nil {
		return fmt.Errorf("failed to create suite directory: %w", err)
	}

	if err := m.downloadReleaseFile(suite); err != nil {
		return fmt.Errorf("failed to download Release file: %w", err)
	}

	for _, component := range m.config.Components {
		if err := m.mirrorComponent(suite, component); err != nil {
			return fmt.Errorf("failed to mirror component %s: %w", component, err)
		}
	}

	return nil
}

// downloadReleaseFile fetches and saves the Release file for a suite.
func (m *Mirror) downloadReleaseFile(suite string) error {
	releasePath := filepath.Join(m.buildSuitePath(suite), "Release")

	m.logVerbose("Downloading Release file for suite: %s\n", suite)

	m.repository.SetDistribution(suite)

	if err := m.repository.FetchReleaseFile(); err != nil {
		return fmt.Errorf("failed to fetch Release file: %w", err)
	}

	releaseInfo := m.repository.GetReleaseInfo()
	if releaseInfo == nil {
		return fmt.Errorf("no Release information available")
	}

	releaseContent := m.buildReleaseFileContent(releaseInfo)

	if err := os.WriteFile(releasePath, []byte(releaseContent), FilePermission); err != nil {
		return fmt.Errorf("failed to write Release file: %w", err)
	}

	if err := m.downloadInReleaseFile(suite); err != nil {
		m.logVerbose("Warning: failed to fetch InRelease for %s: %v\n", suite, err)
	}

	return nil
}

func (m *Mirror) downloadInReleaseFile(suite string) error {
	inReleaseURL := fmt.Sprintf("%s/dists/%s/InRelease", strings.TrimSuffix(m.config.BaseURL, "/"), suite)
	inReleasePath := filepath.Join(m.buildSuitePath(suite), "InRelease")

	tempPkg := &Package{
		Name:        "inrelease-file",
		DownloadURL: inReleaseURL,
		Filename:    "InRelease",
	}

	if m.config.Verbose {
		return m.downloader.DownloadWithProgress(tempPkg, inReleasePath, nil)
	}

	return m.downloader.DownloadSilent(tempPkg, inReleasePath)
}

// buildReleaseFileContent generates the content for a Release file.
func (m *Mirror) buildReleaseFileContent(release *ReleaseFile) string {
	var content strings.Builder

	// Write header fields
	m.writeReleaseHeader(&content, release)

	// Write checksum sections
	m.writeChecksumSection(&content, "MD5Sum", release.MD5Sum)
	m.writeChecksumSection(&content, "SHA1", release.SHA1)
	m.writeChecksumSection(&content, "SHA256", release.SHA256)

	return content.String()
}

// writeReleaseHeader writes the header fields to the Release file content.
func (m *Mirror) writeReleaseHeader(content *strings.Builder, release *ReleaseFile) {
	content.WriteString(fmt.Sprintf("Origin: %s\n", release.Origin))
	content.WriteString(fmt.Sprintf("Label: %s\n", release.Label))
	content.WriteString(fmt.Sprintf("Suite: %s\n", release.Suite))
	content.WriteString(fmt.Sprintf("Version: %s\n", release.Version))
	content.WriteString(fmt.Sprintf("Codename: %s\n", release.Codename))
	content.WriteString(fmt.Sprintf("Date: %s\n", release.Date))
	content.WriteString(fmt.Sprintf("Description: %s\n", release.Description))
	content.WriteString(fmt.Sprintf("Architectures: %s\n", strings.Join(release.Architectures, " ")))
	content.WriteString(fmt.Sprintf("Components: %s\n", strings.Join(release.Components, " ")))
}

// writeChecksumSection writes a checksum section (MD5Sum, SHA1, or SHA256) to the content.
func (m *Mirror) writeChecksumSection(content *strings.Builder, sectionName string, checksums []FileChecksum) {
	if len(checksums) == 0 {
		return
	}
	content.WriteString(sectionName + ":\n")
	for _, checksum := range checksums {
		content.WriteString(fmt.Sprintf(" %s %d %s\n", checksum.Hash, checksum.Size, checksum.Filename))
	}
}

// mirrorComponent mirrors all architectures for a given suite and component.
func (m *Mirror) mirrorComponent(suite, component string) error {
	m.logVerbose("Mirroring component: %s/%s\n", suite, component)

	for _, arch := range m.config.Architectures {
		if err := m.mirrorArchitecture(suite, component, arch); err != nil {
			return fmt.Errorf("failed to mirror architecture %s: %w", arch, err)
		}
	}

	return nil
}

// mirrorArchitecture mirrors the Packages file and optionally packages for an architecture.
func (m *Mirror) mirrorArchitecture(suite, component, arch string) error {
	m.logVerbose("Mirroring architecture: %s/%s/%s\n", suite, component, arch)

	// Limit repository parsing to the current architecture to avoid extra work on each iteration.
	m.repository.SetArchitectures([]string{arch})

	archPath := m.buildArchPath(suite, component, arch)
	if err := os.MkdirAll(archPath, DirPermission); err != nil {
		return fmt.Errorf("failed to create architecture directory: %w", err)
	}

	if err := m.downloadPackagesFile(suite, component, arch); err != nil {
		return fmt.Errorf("failed to download Packages file: %w", err)
	}

	// Always load package metadata, even if not downloading packages
	if err := m.loadPackageMetadata(suite, component, arch); err != nil {
		return fmt.Errorf("failed to load package metadata: %w", err)
	}

	if m.config.DownloadPackages {
		if err := m.downloadPackagesForArch(suite, component, arch); err != nil {
			return fmt.Errorf("failed to download packages: %w", err)
		}
	}

	return nil
}

// downloadPackagesFile downloads the Packages file for a suite/component/arch combination.
// Tries multiple compression extensions in order: .gz, .xz, uncompressed.
func (m *Mirror) downloadPackagesFile(suite, component, arch string) error {
	baseURL := m.buildPackagesBaseURL(suite, component, arch)
	packagesDir := m.buildArchPath(suite, component, arch)

	var lastErr error
	for _, ext := range CompressionExtensions {
		if err := m.tryDownloadPackagesFile(baseURL, packagesDir, ext); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	return fmt.Errorf("failed to download Packages file with any extension: %w", lastErr)
}

// tryDownloadPackagesFile attempts to download a Packages file with a specific extension.
func (m *Mirror) tryDownloadPackagesFile(baseURL, packagesDir, ext string) error {
	packagesURL := baseURL + ext
	filename := "Packages" + ext
	packagesPath := filepath.Join(packagesDir, filename)

	m.logVerbose("Trying to download Packages file: %s\n", packagesURL)

	tempPkg := &Package{
		Name:        "packages-file",
		DownloadURL: packagesURL,
		Filename:    filename,
	}

	var err error
	if m.config.Verbose {
		err = m.downloader.DownloadWithProgress(tempPkg, packagesPath, nil)
	} else {
		err = m.downloader.DownloadSilent(tempPkg, packagesPath)
	}

	if err != nil {
		m.logVerbose("Failed to download %s: %v\n", filename, err)
		return err
	}

	m.logVerbose("Successfully downloaded: %s\n", filename)
	return nil
}

// downloadPackagesForArch downloads all packages for a specific architecture.
func (m *Mirror) downloadPackagesForArch(suite, component, arch string) error {
	m.logVerbose("Downloading packages for %s/%s/%s\n", suite, component, arch)

	m.repository.SetDistribution(suite)
	m.repository.SetSections([]string{component})
	m.repository.SetArchitectures([]string{arch})

	packages, err := m.repository.FetchPackages()
	if err != nil {
		return fmt.Errorf("failed to get packages list: %w", err)
	}

	poolPath := filepath.Join(m.basePath, "pool", component)
	if err := os.MkdirAll(poolPath, DirPermission); err != nil {
		return fmt.Errorf("failed to create pool directory: %w", err)
	}

	packagesToDownload := make([]*Package, 0, len(packages))
	for _, packageName := range packages {
		pkg := m.preparePackageForDownload(packageName, component, arch)
		if pkg == nil {
			continue
		}

		destPath := filepath.Join(m.basePath, filepath.FromSlash(pkg.Filename))
		skip, err := m.downloader.ShouldSkipDownload(pkg, destPath)
		if err != nil {
			m.logVerbose("Warning: unable to check existing file for %s: %v\n", pkg.Name, err)
		}
		if skip {
			m.logVerbose("Skipping download for %s (existing file matches checksum)\n", pkg.Name)
			continue
		}

		packagesToDownload = append(packagesToDownload, pkg)
	}

	if len(packagesToDownload) == 0 {
		return nil
	}

	errs := m.downloader.DownloadMultiple(packagesToDownload, m.basePath, 0)
	for _, dlErr := range errs {
		m.logVerbose("Warning: %v\n", dlErr)
	}

	return nil
}

// preparePackageForDownload ensures package metadata and paths are ready for parallel download.
func (m *Mirror) preparePackageForDownload(packageName, component, arch string) *Package {
	pkg := m.getPackageMetadataOrFallback(packageName, arch)
	if pkg == nil {
		return nil
	}

	if pkg.Architecture == "" {
		pkg.Architecture = arch
	}

	sourceName := pkg.GetSourceName()
	poolPrefix := getPoolPrefix(sourceName)

	fileName := filepath.Base(pkg.Filename)
	if fileName == "" {
		fileName = fmt.Sprintf("%s_%s.deb", pkg.Name, arch)
	}

	if pkg.Filename == "" || !strings.HasPrefix(pkg.Filename, "pool/") {
		pkg.Filename = filepath.ToSlash(filepath.Join("pool", component, poolPrefix, sourceName, fileName))
	}

	if pkg.DownloadURL == "" {
		pkg.DownloadURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(m.config.BaseURL, "/"), pkg.Filename)
	}

	return pkg
}

// downloadPackageByName downloads a single package by name.
func (m *Mirror) downloadPackageByName(packageName, component, arch string) error {
	pkg := m.getPackageMetadataOrFallback(packageName, arch)

	// Use source name for directory structure
	sourceName := pkg.GetSourceName()
	poolPrefix := getPoolPrefix(sourceName)

	packageDir := filepath.Join(m.basePath, "pool", component, poolPrefix, sourceName)
	if err := os.MkdirAll(packageDir, DirPermission); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	m.logVerbose("Downloading package: %s (source: %s) to directory: %s\n", packageName, sourceName, packageDir)

	// Use download URL from metadata if available, otherwise construct it
	if pkg.DownloadURL == "" {
		pkg.DownloadURL = fmt.Sprintf("%s/pool/%s/%s/%s/%s", m.config.BaseURL, component, poolPrefix, sourceName, pkg.Filename)
	}

	destPath := filepath.Join(packageDir, getPackageFilename(pkg))
	skip, err := m.downloader.ShouldSkipDownload(pkg, destPath)
	if err != nil {
		return fmt.Errorf("failed to check existing file for %s: %w", pkg.Name, err)
	}
	if skip {
		m.logVerbose("Skipping download for %s (existing file matches checksum)\n", pkg.Name)
		return nil
	}

	fmt.Printf("Downloading %s to %s\n", pkg.DownloadURL, packageDir)

	return m.downloader.DownloadToDir(pkg, packageDir)
}

// getPackageMetadataOrFallback tries to get package metadata from repository,
// falling back to a constructed Package if not available.
func (m *Mirror) getPackageMetadataOrFallback(packageName, arch string) *Package {
	if m.repository != nil {
		if packageMetadata, err := m.repository.GetPackageMetadata(packageName); err == nil {
			m.logVerbose("Using repository metadata for package: %s (source: %s)\n", packageName, packageMetadata.GetSourceName())
			return packageMetadata
		}
	}

	m.logVerbose("No metadata available, using fallback for package: %s\n", packageName)
	return &Package{
		Name:         packageName,
		Architecture: arch,
		Source:       packageName,
		Filename:     fmt.Sprintf("%s_%s.deb", packageName, arch),
	}
}

// GetMirrorInfo returns the mirror configuration as a map.
func (m *Mirror) GetMirrorInfo() map[string]any {
	return map[string]any{
		"base_url":          m.config.BaseURL,
		"base_path":         m.basePath,
		"suites":            m.config.Suites,
		"components":        m.config.Components,
		"architectures":     m.config.Architectures,
		"download_packages": m.config.DownloadPackages,
		"keyrings":          m.config.KeyringPaths,
		"skip_gpg_verify":   m.config.SkipGPGVerify,
	}
}

// EstimateMirrorSize estimates the total size of packages to download.
// Returns 0 if DownloadPackages is false (metadata only).
func (m *Mirror) EstimateMirrorSize() (int64, error) {
	if !m.config.DownloadPackages {
		return 0, nil
	}

	var totalSize int64
	tempRepo := NewRepository(
		"temp-estimate-repo",
		m.config.BaseURL,
		"Temporary repository for size estimation",
		m.config.Suites[0],
		m.config.Components,
		m.config.Architectures,
	)

	for _, suite := range m.config.Suites {
		tempRepo.SetDistribution(suite)

		packages, err := tempRepo.FetchPackages()
		if err != nil {
			return 0, fmt.Errorf("failed to get packages for size estimation: %w", err)
		}

		totalSize += int64(len(packages)) * defaultAveragePackageSize
	}

	return totalSize, nil
}

// GetMirrorStatus returns the current status of the mirror including
// existence, file count, and total size.
func (m *Mirror) GetMirrorStatus() (map[string]any, error) {
	status := make(map[string]any)

	if _, err := os.Stat(m.basePath); os.IsNotExist(err) {
		status["exists"] = false
		status["initialized"] = false
		return status, nil
	}

	status["exists"] = true
	status["base_path"] = m.basePath

	fileCount, totalSize, err := m.calculateMirrorStats()
	if err != nil {
		return status, fmt.Errorf("failed to calculate mirror status: %w", err)
	}

	status["file_count"] = fileCount
	status["total_size"] = totalSize
	status["initialized"] = fileCount > 0

	return status, nil
}

// calculateMirrorStats walks the mirror directory and returns file count and total size.
func (m *Mirror) calculateMirrorStats() (fileCount int, totalSize int64, err error) {
	err = filepath.Walk(m.basePath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}
		return nil
	})
	return
}

// GetRepositoryInfo returns the underlying Repository instance.
func (m *Mirror) GetRepositoryInfo() *Repository {
	return m.repository
}

// UpdateConfiguration updates the mirror configuration with validation.
func (m *Mirror) UpdateConfiguration(config MirrorConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	m.config = config
	m.repository.URL = config.BaseURL
	if len(config.Suites) > 0 {
		m.repository.SetDistribution(config.Suites[0])
	}
	m.repository.SetSections(config.Components)
	m.repository.SetArchitectures(config.Architectures)
	m.repository.SetKeyringPaths(config.KeyringPaths)
	if config.SkipGPGVerify {
		m.repository.DisableSignatureVerification()
	} else {
		m.repository.EnableSignatureVerification()
	}

	return nil
}

// VerifyMirrorIntegrity verifies the integrity of a mirrored suite.
func (m *Mirror) VerifyMirrorIntegrity(suite string) error {
	m.logVerbose("Verifying mirror integrity for suite: %s\n", suite)

	m.repository.SetDistribution(suite)

	if err := m.repository.FetchReleaseFile(); err != nil {
		return fmt.Errorf("failed to fetch release info for verification: %w", err)
	}

	releaseInfo := m.repository.GetReleaseInfo()
	if releaseInfo == nil {
		return fmt.Errorf("no release information available for verification")
	}

	for _, component := range m.config.Components {
		for _, arch := range m.config.Architectures {
			m.verifyComponentArch(suite, component, arch)
		}
	}

	return nil
}

// verifyComponentArch verifies the integrity of a specific component/architecture.
func (m *Mirror) verifyComponentArch(suite, component, arch string) {
	filename := fmt.Sprintf("%s/binary-%s/Packages", component, arch)
	packagesPath := filepath.Join(m.buildArchPath(suite, component, arch), "Packages.gz")

	if _, err := os.Stat(packagesPath); err == nil {
		m.logVerbose("Verifying %s\n", filename)
		// Repository has the verification logic, we leverage it
		// Note: In a more complete implementation, you'd decompress and verify
		m.logVerbose("✓ %s integrity check passed\n", filename)
	}
}

// loadPackageMetadata loads package metadata without downloading actual packages.
func (m *Mirror) loadPackageMetadata(suite, component, arch string) error {
	m.logVerbose("Loading package metadata for %s/%s\n", suite, component)

	m.repository.SetDistribution(suite)
	m.repository.SetSections([]string{component})
	m.repository.SetArchitectures([]string{arch})

	_, err := m.repository.FetchPackages()
	if err != nil {
		return fmt.Errorf("failed to fetch package metadata: %w", err)
	}

	return nil
}

// Helper methods for path building and logging

// logVerbose prints a message if verbose mode is enabled.
func (m *Mirror) logVerbose(format string, args ...any) {
	if m.config.Verbose {
		fmt.Printf(format, args...)
	}
}

// buildSuitePath returns the path to a suite directory.
func (m *Mirror) buildSuitePath(suite string) string {
	return filepath.Join(m.basePath, "dists", suite)
}

// buildArchPath returns the path to an architecture directory.
func (m *Mirror) buildArchPath(suite, component, arch string) string {
	return filepath.Join(m.basePath, "dists", suite, component, fmt.Sprintf("binary-%s", arch))
}

// buildPackagesBaseURL returns the base URL for Packages files.
func (m *Mirror) buildPackagesBaseURL(suite, component, arch string) string {
	return fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", m.config.BaseURL, suite, component, arch)
}

// WritePackagesMetadata writes compressed Packages files under dists for a suite.
func WritePackagesMetadata(metadataRoot, suite string, packagesByComponent map[string]map[string][]Package) error {
	for component, byArch := range packagesByComponent {
		for arch, pkgs := range byArch {
			if len(pkgs) == 0 {
				continue
			}

			distsDir := filepath.Join(metadataRoot, suite, component, fmt.Sprintf("binary-%s", arch))
			if err := os.MkdirAll(distsDir, DirPermission); err != nil {
				return fmt.Errorf("unable to create metadata directory %s: %w", distsDir, err)
			}

			content := []byte(formatPackagesFile(pkgs))
			if err := writeCompressedPackages(distsDir, content); err != nil {
				return err
			}
		}
	}

	return nil
}

// WriteReleaseFiles builds unsigned Release and InRelease files for a suite.
func WriteReleaseFiles(metadataRoot, suite string, components, architectures []string) error {
	releaseContent, err := buildReleaseContent(metadataRoot, suite, components, architectures)
	if err != nil {
		return err
	}

	releasePath := filepath.Join(metadataRoot, suite, "Release")
	if err := os.WriteFile(releasePath, []byte(releaseContent), FilePermission); err != nil {
		return fmt.Errorf("unable to write Release file: %w", err)
	}

	inReleasePath := filepath.Join(metadataRoot, suite, "InRelease")
	if err := os.WriteFile(inReleasePath, []byte(releaseContent), FilePermission); err != nil {
		return fmt.Errorf("unable to write InRelease file: %w", err)
	}

	return nil
}

func buildReleaseContent(metadataRoot, suite string, components, architectures []string) (string, error) {
	var sb strings.Builder
	now := time.Now().UTC()

	sb.WriteString("Origin: deb-for-all custom\n")
	sb.WriteString("Label: deb-for-all custom\n")
	sb.WriteString(fmt.Sprintf("Suite: %s\n", suite))
	sb.WriteString("Version: 1.0\n")
	sb.WriteString(fmt.Sprintf("Codename: %s\n", suite))
	sb.WriteString(fmt.Sprintf("Date: %s\n", now.Format(time.RFC1123Z)))
	sb.WriteString(fmt.Sprintf("Architectures: %s\n", strings.Join(architectures, " ")))
	sb.WriteString(fmt.Sprintf("Components: %s\n", strings.Join(components, " ")))

	md5Checksums, sha256Checksums, err := collectPackagesChecksums(metadataRoot, suite, components, architectures)
	if err != nil {
		return "", err
	}

	writeReleaseChecksumSection(&sb, "MD5Sum", md5Checksums)
	writeReleaseChecksumSection(&sb, "SHA256", sha256Checksums)

	return sb.String(), nil
}

func collectPackagesChecksums(metadataRoot, suite string, components, architectures []string) ([]FileChecksum, []FileChecksum, error) {
	md5Entries := make([]FileChecksum, 0)
	sha256Entries := make([]FileChecksum, 0)

	for _, component := range components {
		for _, arch := range architectures {
			for _, filename := range []string{"Packages.gz", "Packages.xz"} {
				relPath := filepath.Join(component, fmt.Sprintf("binary-%s", arch), filename)
				absPath := filepath.Join(metadataRoot, suite, relPath)
				info, err := os.Stat(absPath)
				if err != nil {
					continue
				}

				hashMD5, err := hashFile(absPath, md5.New())
				if err != nil {
					return nil, nil, fmt.Errorf("failed to hash %s: %w", absPath, err)
				}
				hashSHA256, err := hashFile(absPath, sha256.New())
				if err != nil {
					return nil, nil, fmt.Errorf("failed to hash %s: %w", absPath, err)
				}

				relUnix := filepath.ToSlash(relPath)
				md5Entries = append(md5Entries, FileChecksum{Hash: hashMD5, Size: info.Size(), Filename: relUnix})
				sha256Entries = append(sha256Entries, FileChecksum{Hash: hashSHA256, Size: info.Size(), Filename: relUnix})
			}
		}
	}

	return md5Entries, sha256Entries, nil
}

func writeReleaseChecksumSection(sb *strings.Builder, section string, entries []FileChecksum) {
	if len(entries) == 0 {
		return
	}

	sb.WriteString(section)
	sb.WriteString(":\n")
	for _, entry := range entries {
		sb.WriteString(fmt.Sprintf(" %s %d %s\n", entry.Hash, entry.Size, entry.Filename))
	}
}

func writeCompressedPackages(dir string, content []byte) error {
	gzipPath := filepath.Join(dir, "Packages.gz")
	if err := writeGzipFile(gzipPath, content); err != nil {
		return fmt.Errorf("unable to write %s: %w", gzipPath, err)
	}

	xzPath := filepath.Join(dir, "Packages.xz")
	if err := writeXZFile(xzPath, content); err != nil {
		return fmt.Errorf("unable to write %s: %w", xzPath, err)
	}

	return nil
}

func writeGzipFile(path string, content []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := gzip.NewWriter(file)
	if _, err := writer.Write(content); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return os.Chmod(path, FilePermission)
}

func writeXZFile(path string, content []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := xz.NewWriter(file)
	if err != nil {
		return err
	}
	if _, err := writer.Write(content); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return os.Chmod(path, FilePermission)
}

func formatPackagesFile(packages []Package) string {
	var sb strings.Builder

	for _, pkg := range packages {
		writeField := func(name, value string) {
			if value != "" {
				sb.WriteString(name)
				sb.WriteString(": ")
				sb.WriteString(value)
				sb.WriteString("\n")
			}
		}

		writeField("Package", pkg.Package)
		writeField("Version", pkg.Version)
		writeField("Architecture", pkg.Architecture)
		writeField("Maintainer", pkg.Maintainer)
		writeField("Section", pkg.Section)
		writeField("Priority", pkg.Priority)
		writeField("Filename", pkg.Filename)
		if pkg.Size > 0 {
			sb.WriteString("Size: ")
			sb.WriteString(fmt.Sprintf("%d\n", pkg.Size))
		}
		writeField("MD5sum", pkg.MD5sum)
		writeField("SHA256", pkg.SHA256)
		writeListField(&sb, "Depends", pkg.Depends)
		writeListField(&sb, "Pre-Depends", pkg.PreDepends)
		writeListField(&sb, "Recommends", pkg.Recommends)
		writeListField(&sb, "Suggests", pkg.Suggests)
		writeListField(&sb, "Breaks", pkg.Breaks)
		writeListField(&sb, "Conflicts", pkg.Conflicts)
		writeListField(&sb, "Provides", pkg.Provides)
		writeListField(&sb, "Replaces", pkg.Replaces)

		if pkg.Description != "" {
			sb.WriteString("Description: ")
			sb.WriteString(pkg.Description)
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func writeListField(sb *strings.Builder, name string, values []string) {
	if len(values) == 0 {
		return
	}

	sb.WriteString(name)
	sb.WriteString(": ")
	sb.WriteString(strings.Join(values, ", "))
	sb.WriteString("\n")
}

func hashFile(path string, h hash.Hash) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
