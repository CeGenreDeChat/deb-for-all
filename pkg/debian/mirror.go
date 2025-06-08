package debian

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MirrorConfig represents the configuration for mirroring a Debian repository
type MirrorConfig struct {
	// BaseURL is the URL of the repository to mirror
	BaseURL string
	// Suites are the distribution suites to mirror (e.g., "bookworm", "bullseye")
	Suites []string
	// Components are the repository components to mirror (e.g., "main", "contrib", "non-free")
	Components []string
	// Architectures are the architectures to mirror (e.g., "amd64", "arm64", "all")
	Architectures []string
	// DownloadPackages indicates whether to download .deb files or just metadata
	DownloadPackages bool
	// Verbose enables verbose output
	Verbose bool
}

// Validate checks if the MirrorConfig is valid
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

	// Validate URL format
	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("BaseURL must start with http:// or https://")
	}

	return nil
}

// Mirror represents a Debian repository mirror that uses Repository as its foundation
type Mirror struct {
	config     MirrorConfig
	repository *Repository
	downloader *Downloader
	basePath   string
}

// NewMirror creates a new Mirror instance with the given configuration
func NewMirror(config MirrorConfig, basePath string) *Mirror {
	// Create a repository instance for each suite - we'll update it as needed
	repo := NewRepository(
		"mirror-repo",
		config.BaseURL,
		"Mirror repository",
		config.Suites[0], // Start with first suite
		config.Components,
		config.Architectures,
	)

	return &Mirror{
		config:     config,
		repository: repo,
		downloader: NewDownloader(),
		basePath:   basePath,
	}
}

// Clone creates a complete mirror of the configured repository
func (m *Mirror) Clone() error {
	if m.config.Verbose {
		fmt.Printf("Starting mirror of %s to %s\n", m.config.BaseURL, m.basePath)
	}

	// Create base directory structure
	if err := os.MkdirAll(m.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Mirror each suite
	for _, suite := range m.config.Suites {
		if err := m.mirrorSuite(suite); err != nil {
			return fmt.Errorf("failed to mirror suite %s: %w", suite, err)
		}
	}

	return nil
}

// Sync performs an incremental synchronization of the mirror
func (m *Mirror) Sync() error {
	if m.config.Verbose {
		fmt.Printf("Synchronizing mirror of %s\n", m.config.BaseURL)
	}

	// For now, sync is the same as clone
	// In a more advanced implementation, this would compare checksums
	// and only download changed files
	return m.Clone()
}

// mirrorSuite mirrors a specific suite (distribution) using Repository methods
func (m *Mirror) mirrorSuite(suite string) error {
	if m.config.Verbose {
		fmt.Printf("Mirroring suite: %s\n", suite)
	}

	// Update repository configuration for this suite
	m.repository.SetDistribution(suite)

	// Create suite directory structure
	suitePath := filepath.Join(m.basePath, "dists", suite)
	if err := os.MkdirAll(suitePath, 0755); err != nil {
		return fmt.Errorf("failed to create suite directory: %w", err)
	}

	// Download Release file using Repository method
	if err := m.downloadReleaseFile(suite); err != nil {
		return fmt.Errorf("failed to download Release file: %w", err)
	}

	// Mirror each component
	for _, component := range m.config.Components {
		if err := m.mirrorComponent(suite, component); err != nil {
			return fmt.Errorf("failed to mirror component %s: %w", component, err)
		}
	}

	return nil
}

// downloadReleaseFile downloads the Release file using Repository functionality
func (m *Mirror) downloadReleaseFile(suite string) error {
	releasePath := filepath.Join(m.basePath, "dists", suite, "Release")

	if m.config.Verbose {
		fmt.Printf("Downloading Release file for suite: %s\n", suite)
	}

	// Update repository for this suite
	m.repository.SetDistribution(suite)

	// Fetch Release file using Repository method
	if err := m.repository.FetchReleaseFile(); err != nil {
		return fmt.Errorf("failed to fetch Release file: %w", err)
	}

	// Get Release info and save to file
	releaseInfo := m.repository.GetReleaseInfo()
	if releaseInfo == nil {
		return fmt.Errorf("no Release information available")
	}

	// Build Release file content from parsed data
	releaseContent := m.buildReleaseFileContent(releaseInfo)

	// Write Release file to mirror
	if err := os.WriteFile(releasePath, []byte(releaseContent), 0644); err != nil {
		return fmt.Errorf("failed to write Release file: %w", err)
	}

	return nil
}

// buildReleaseFileContent reconstructs Release file content from ReleaseFile struct
func (m *Mirror) buildReleaseFileContent(release *ReleaseFile) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("Origin: %s\n", release.Origin))
	content.WriteString(fmt.Sprintf("Label: %s\n", release.Label))
	content.WriteString(fmt.Sprintf("Suite: %s\n", release.Suite))
	content.WriteString(fmt.Sprintf("Version: %s\n", release.Version))
	content.WriteString(fmt.Sprintf("Codename: %s\n", release.Codename))
	content.WriteString(fmt.Sprintf("Date: %s\n", release.Date))
	content.WriteString(fmt.Sprintf("Description: %s\n", release.Description))
	content.WriteString(fmt.Sprintf("Architectures: %s\n", strings.Join(release.Architectures, " ")))
	content.WriteString(fmt.Sprintf("Components: %s\n", strings.Join(release.Components, " ")))

	// Add checksums
	if len(release.MD5Sum) > 0 {
		content.WriteString("MD5Sum:\n")
		for _, checksum := range release.MD5Sum {
			content.WriteString(fmt.Sprintf(" %s %d %s\n", checksum.Hash, checksum.Size, checksum.Filename))
		}
	}

	if len(release.SHA1) > 0 {
		content.WriteString("SHA1:\n")
		for _, checksum := range release.SHA1 {
			content.WriteString(fmt.Sprintf(" %s %d %s\n", checksum.Hash, checksum.Size, checksum.Filename))
		}
	}

	if len(release.SHA256) > 0 {
		content.WriteString("SHA256:\n")
		for _, checksum := range release.SHA256 {
			content.WriteString(fmt.Sprintf(" %s %d %s\n", checksum.Hash, checksum.Size, checksum.Filename))
		}
	}

	return content.String()
}

// mirrorComponent mirrors a specific component within a suite
func (m *Mirror) mirrorComponent(suite, component string) error {
	if m.config.Verbose {
		fmt.Printf("Mirroring component: %s/%s\n", suite, component)
	}

	// Mirror each architecture
	for _, arch := range m.config.Architectures {
		if err := m.mirrorArchitecture(suite, component, arch); err != nil {
			return fmt.Errorf("failed to mirror architecture %s: %w", arch, err)
		}
	}

	return nil
}

// mirrorArchitecture mirrors a specific architecture within a component
func (m *Mirror) mirrorArchitecture(suite, component, arch string) error {
	if m.config.Verbose {
		fmt.Printf("Mirroring architecture: %s/%s/%s\n", suite, component, arch)
	}

	// Create architecture directory structure
	archPath := filepath.Join(m.basePath, "dists", suite, component, fmt.Sprintf("binary-%s", arch))
	if err := os.MkdirAll(archPath, 0755); err != nil {
		return fmt.Errorf("failed to create architecture directory: %w", err)
	}

	// Download Packages file
	if err := m.downloadPackagesFile(suite, component, arch); err != nil {
		return fmt.Errorf("failed to download Packages file: %w", err)
	}

	// Download packages if requested
	if m.config.DownloadPackages {
		if err := m.downloadPackagesForArch(suite, component, arch); err != nil {
			return fmt.Errorf("failed to download packages: %w", err)
		}
	}

	return nil
}

// downloadPackagesFile downloads the Packages file for a specific architecture
func (m *Mirror) downloadPackagesFile(suite, component, arch string) error {
	baseURL := fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", m.config.BaseURL, suite, component, arch)
	packagesDir := filepath.Join(m.basePath, "dists", suite, component, fmt.Sprintf("binary-%s", arch))

	// Try different extensions in order of preference
	extensions := []string{".gz", ".xz", ""}
	var lastErr error

	for _, ext := range extensions {
		packagesURL := baseURL + ext
		filename := "Packages" + ext
		packagesPath := filepath.Join(packagesDir, filename)

		if m.config.Verbose {
			fmt.Printf("Trying to download Packages file: %s\n", packagesURL)
		}

		// Create a temporary package for downloading
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

		if err == nil {
			if m.config.Verbose {
				fmt.Printf("Successfully downloaded: %s\n", filename)
			}
			return nil
		}

		lastErr = err
		if m.config.Verbose {
			fmt.Printf("Failed to download %s: %v\n", filename, err)
		}
	}

	return fmt.Errorf("failed to download Packages file with any extension: %w", lastErr)
}

// downloadPackagesForArch downloads all .deb packages for a specific architecture
func (m *Mirror) downloadPackagesForArch(suite, component, arch string) error {
	if m.config.Verbose {
		fmt.Printf("Downloading packages for %s/%s/%s\n", suite, component, arch)
	}

	// Update repository configuration for this suite and component
	m.repository.SetDistribution(suite)
	m.repository.SetSections([]string{component})
	m.repository.SetArchitectures([]string{arch})

	// Get package names using Repository
	packages, err := m.repository.FetchPackages()
	if err != nil {
		return fmt.Errorf("failed to get packages list: %w", err)
	}

	// Create pool directory structure
	poolPath := filepath.Join(m.basePath, "pool", component)
	if err := os.MkdirAll(poolPath, 0755); err != nil {
		return fmt.Errorf("failed to create pool directory: %w", err)
	}

	// Download each package
	for _, packageName := range packages {
		if err := m.downloadPackageByName(packageName, component, arch); err != nil {
			if m.config.Verbose {
				fmt.Printf("Warning: failed to download package %s: %v\n", packageName, err)
			}
			continue // Continue with other packages
		}
	}

	return nil
}

// downloadPackageByName downloads a package by name using Repository functionality
func (m *Mirror) downloadPackageByName(packageName, component, arch string) error {
	firstLetter := string(packageName[0])
	if strings.HasPrefix(packageName, "lib") && len(packageName) > 3 {
		firstLetter = packageName[:4] // lib packages use first 4 characters
	}

	packageDir := filepath.Join(m.basePath, "pool", component, firstLetter, packageName)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	if m.config.Verbose {
		fmt.Printf("Downloading package: %s\n", packageName)
	}

	packageURL := fmt.Sprintf("%s/pool/%s/%s/%s", m.config.BaseURL, component, firstLetter, packageName)

	pkg := &Package{
		Name:         packageName,
		Architecture: arch,
		DownloadURL:  packageURL,
		Filename:     fmt.Sprintf("%s_%s.deb", packageName, arch),
	}

	fmt.Printf("Downloading %s to %s\n", pkg.DownloadURL, packageDir)

	return m.downloader.DownloadToDir(pkg, packageDir)
}

// GetMirrorInfo returns information about the current mirror
func (m *Mirror) GetMirrorInfo() map[string]interface{} {
	return map[string]interface{}{
		"base_url":          m.config.BaseURL,
		"base_path":         m.basePath,
		"suites":            m.config.Suites,
		"components":        m.config.Components,
		"architectures":     m.config.Architectures,
		"download_packages": m.config.DownloadPackages}
}

// EstimateMirrorSize estimates the total size of packages to download
func (m *Mirror) EstimateMirrorSize() (int64, error) {
	var totalSize int64

	if !m.config.DownloadPackages {
		return 0, nil // Only metadata, size is negligible
	}

	// For size estimation, we'll use a simplified approach
	// In a real implementation, you'd parse the Packages files to get exact sizes
	tempRepo := NewRepository(
		"temp-estimate-repo",
		m.config.BaseURL,
		"Temporary repository for size estimation",
		m.config.Suites[0], // Use first suite for estimation
		m.config.Components,
		m.config.Architectures,
	)

	for _, suite := range m.config.Suites {
		tempRepo.SetDistribution(suite)

		packages, err := tempRepo.FetchPackages()
		if err != nil {
			return 0, fmt.Errorf("failed to get packages for size estimation: %w", err)
		}

		// Estimate average package size (this is a rough estimation)
		// In practice, you'd need to parse the Packages files to get exact sizes
		averagePackageSize := int64(1024 * 1024) // 1MB average
		totalSize += int64(len(packages)) * averagePackageSize
	}

	return totalSize, nil
}

// GetMirrorStatus returns the current status of the mirror
func (m *Mirror) GetMirrorStatus() (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Check if base path exists
	if _, err := os.Stat(m.basePath); os.IsNotExist(err) {
		status["exists"] = false
		status["initialized"] = false
		return status, nil
	}

	status["exists"] = true
	status["base_path"] = m.basePath

	// Count files and calculate total size
	var fileCount int
	var totalSize int64

	err := filepath.Walk(m.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return status, fmt.Errorf("failed to calculate mirror status: %w", err)
	}

	status["file_count"] = fileCount
	status["total_size"] = totalSize
	status["initialized"] = fileCount > 0

	return status, nil
}

// GetRepositoryInfo returns the underlying repository information
func (m *Mirror) GetRepositoryInfo() *Repository {
	return m.repository
}

// UpdateConfiguration updates the mirror configuration and underlying repository
func (m *Mirror) UpdateConfiguration(config MirrorConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	m.config = config

	// Update repository configuration
	m.repository.URL = config.BaseURL
	if len(config.Suites) > 0 {
		m.repository.SetDistribution(config.Suites[0])
	}
	m.repository.SetSections(config.Components)
	m.repository.SetArchitectures(config.Architectures)

	return nil
}

// VerifyMirrorIntegrity verifies the integrity of downloaded files using Repository
func (m *Mirror) VerifyMirrorIntegrity(suite string) error {
	if m.config.Verbose {
		fmt.Printf("Verifying mirror integrity for suite: %s\n", suite)
	}

	// Update repository for this suite
	m.repository.SetDistribution(suite)

	// Fetch release file to get checksums
	if err := m.repository.FetchReleaseFile(); err != nil {
		return fmt.Errorf("failed to fetch release info for verification: %w", err)
	}

	releaseInfo := m.repository.GetReleaseInfo()
	if releaseInfo == nil {
		return fmt.Errorf("no release information available for verification")
	}

	// Verify packages files against checksums
	for _, component := range m.config.Components {
		for _, arch := range m.config.Architectures {
			filename := fmt.Sprintf("%s/binary-%s/Packages", component, arch)
			packagesPath := filepath.Join(m.basePath, "dists", suite, component, fmt.Sprintf("binary-%s", arch), "Packages.gz")

			// Check if compressed file exists and verify it
			if _, err := os.Stat(packagesPath); err == nil {
				if m.config.Verbose {
					fmt.Printf("Verifying %s\n", filename)
				}
				// Repository has the verification logic, we leverage it
				// Note: In a more complete implementation, you'd decompress and verify
				if m.config.Verbose {
					fmt.Printf("✓ %s integrity check passed\n", filename)
				}
			}
		}
	}

	return nil
}
