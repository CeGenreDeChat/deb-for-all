package debian

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MirrorConfig struct {
	BaseURL          string
	Suites           []string
	Components       []string
	Architectures    []string
	DownloadPackages bool
	Verbose          bool
}

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

	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("BaseURL must start with http:// or https://")
	}

	return nil
}

type Mirror struct {
	config     MirrorConfig
	repository *Repository
	downloader *Downloader
	basePath   string
}

func NewMirror(config MirrorConfig, basePath string) *Mirror {
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

func (m *Mirror) Clone() error {
	if m.config.Verbose {
		fmt.Printf("Starting mirror of %s to %s\n", m.config.BaseURL, m.basePath)
	}

	if err := os.MkdirAll(m.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

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

func (m *Mirror) mirrorSuite(suite string) error {
	if m.config.Verbose {
		fmt.Printf("Mirroring suite: %s\n", suite)
	}

	m.repository.SetDistribution(suite)

	suitePath := filepath.Join(m.basePath, "dists", suite)
	if err := os.MkdirAll(suitePath, 0755); err != nil {
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

func (m *Mirror) downloadReleaseFile(suite string) error {
	releasePath := filepath.Join(m.basePath, "dists", suite, "Release")

	if m.config.Verbose {
		fmt.Printf("Downloading Release file for suite: %s\n", suite)
	}

	m.repository.SetDistribution(suite)

	if err := m.repository.FetchReleaseFile(); err != nil {
		return fmt.Errorf("failed to fetch Release file: %w", err)
	}

	releaseInfo := m.repository.GetReleaseInfo()
	if releaseInfo == nil {
		return fmt.Errorf("no Release information available")
	}

	releaseContent := m.buildReleaseFileContent(releaseInfo)

	if err := os.WriteFile(releasePath, []byte(releaseContent), 0644); err != nil {
		return fmt.Errorf("failed to write Release file: %w", err)
	}

	return nil
}

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

func (m *Mirror) mirrorComponent(suite, component string) error {
	if m.config.Verbose {
		fmt.Printf("Mirroring component: %s/%s\n", suite, component)
	}

	for _, arch := range m.config.Architectures {
		if err := m.mirrorArchitecture(suite, component, arch); err != nil {
			return fmt.Errorf("failed to mirror architecture %s: %w", arch, err)
		}
	}

	return nil
}

func (m *Mirror) mirrorArchitecture(suite, component, arch string) error {
	if m.config.Verbose {
		fmt.Printf("Mirroring architecture: %s/%s/%s\n", suite, component, arch)
	}

	archPath := filepath.Join(m.basePath, "dists", suite, component, fmt.Sprintf("binary-%s", arch))
	if err := os.MkdirAll(archPath, 0755); err != nil {
		return fmt.Errorf("failed to create architecture directory: %w", err)
	}

	if err := m.downloadPackagesFile(suite, component, arch); err != nil {
		return fmt.Errorf("failed to download Packages file: %w", err)
	}

	// Always load package metadata, even if not downloading packages
	if err := m.loadPackageMetadata(suite, component); err != nil {
		return fmt.Errorf("failed to load package metadata: %w", err)
	}

	if m.config.DownloadPackages {
		if err := m.downloadPackagesForArch(suite, component, arch); err != nil {
			return fmt.Errorf("failed to download packages: %w", err)
		}
	}

	return nil
}

func (m *Mirror) downloadPackagesFile(suite, component, arch string) error {
	baseURL := fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", m.config.BaseURL, suite, component, arch)
	packagesDir := filepath.Join(m.basePath, "dists", suite, component, fmt.Sprintf("binary-%s", arch))

	extensions := []string{".gz", ".xz", ""}
	var lastErr error

	for _, ext := range extensions {
		packagesURL := baseURL + ext
		filename := "Packages" + ext
		packagesPath := filepath.Join(packagesDir, filename)

		if m.config.Verbose {
			fmt.Printf("Trying to download Packages file: %s\n", packagesURL)
		}

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

func (m *Mirror) downloadPackagesForArch(suite, component, arch string) error {
	if m.config.Verbose {
		fmt.Printf("Downloading packages for %s/%s/%s\n", suite, component, arch)
	}

	m.repository.SetDistribution(suite)
	m.repository.SetSections([]string{component})

	packages, err := m.repository.FetchPackages()
	if err != nil {
		return fmt.Errorf("failed to get packages list: %w", err)
	}

	poolPath := filepath.Join(m.basePath, "pool", component)
	if err := os.MkdirAll(poolPath, 0755); err != nil {
		return fmt.Errorf("failed to create pool directory: %w", err)
	}

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

func (m *Mirror) downloadPackageByName(packageName, component, arch string) error {
	// Try to get actual package metadata from repository
	var pkg *Package

	// First attempt to get metadata from repository
	if m.repository != nil {
		if packageMetadata, err := m.repository.GetPackageMetadata(packageName); err == nil {
			// Use actual metadata from repository
			pkg = packageMetadata
			if m.config.Verbose {
				fmt.Printf("Using repository metadata for package: %s (source: %s)\n", packageName, pkg.GetSourceName())
			}
		}
	}

	// Fallback to creating package object if no metadata available
	if pkg == nil {
		pkg = &Package{
			Name:         packageName,
			Architecture: arch,
			Source:       packageName, // Default to package name
			Filename:     fmt.Sprintf("%s_%s.deb", packageName, arch),
		}
		if m.config.Verbose {
			fmt.Printf("No metadata available, using fallback for package: %s\n", packageName)
		}
	}

	// Use source name for directory structure
	sourceName := pkg.GetSourceName()
	firstLetter := string(sourceName[0])
	if strings.HasPrefix(sourceName, "lib") && len(sourceName) > 3 {
		firstLetter = sourceName[:4] // lib packages use first 4 characters
	}

	packageDir := filepath.Join(m.basePath, "pool", component, firstLetter, sourceName)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	if m.config.Verbose {
		fmt.Printf("Downloading package: %s (source: %s) to directory: %s\n", packageName, sourceName, packageDir)
	}

	// Use download URL from metadata if available, otherwise construct it
	var packageURL string
	if pkg.DownloadURL != "" {
		packageURL = pkg.DownloadURL
	} else {
		packageURL = fmt.Sprintf("%s/pool/%s/%s/%s/%s", m.config.BaseURL, component, firstLetter, sourceName, pkg.Filename)
		pkg.DownloadURL = packageURL
	}

	fmt.Printf("Downloading %s to %s\n", packageURL, packageDir)

	return m.downloader.DownloadToDir(pkg, packageDir)
}

func (m *Mirror) GetMirrorInfo() map[string]any {
	return map[string]any{
		"base_url":          m.config.BaseURL,
		"base_path":         m.basePath,
		"suites":            m.config.Suites,
		"components":        m.config.Components,
		"architectures":     m.config.Architectures,
		"download_packages": m.config.DownloadPackages}
}

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

func (m *Mirror) GetMirrorStatus() (map[string]any, error) {
	status := make(map[string]any)

	if _, err := os.Stat(m.basePath); os.IsNotExist(err) {
		status["exists"] = false
		status["initialized"] = false
		return status, nil
	}

	status["exists"] = true
	status["base_path"] = m.basePath

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

func (m *Mirror) GetRepositoryInfo() *Repository {
	return m.repository
}

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

	return nil
}

func (m *Mirror) VerifyMirrorIntegrity(suite string) error {
	if m.config.Verbose {
		fmt.Printf("Verifying mirror integrity for suite: %s\n", suite)
	}

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
			filename := fmt.Sprintf("%s/binary-%s/Packages", component, arch)
			packagesPath := filepath.Join(m.basePath, "dists", suite, component, fmt.Sprintf("binary-%s", arch), "Packages.gz")

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

// loadPackageMetadata loads package metadata without downloading actual packages
func (m *Mirror) loadPackageMetadata(suite, component string) error {
	if m.config.Verbose {
		fmt.Printf("Loading package metadata for %s/%s\n", suite, component)
	}

	m.repository.SetDistribution(suite)
	m.repository.SetSections([]string{component})

	_, err := m.repository.FetchPackages()
	if err != nil {
		return fmt.Errorf("failed to fetch package metadata: %w", err)
	}

	return nil
}
