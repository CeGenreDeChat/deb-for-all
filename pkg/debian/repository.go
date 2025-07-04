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

type Repository struct {
	Name            string
	URL             string
	Description     string
	Distribution    string
	Sections        []string
	Architectures   []string
	Packages        []string
	PackageMetadata []Package // Complete package metadata parsed from Packages files
	ReleaseInfo     *ReleaseFile
	VerifyRelease   bool
}

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

func (r *Repository) FetchPackages() ([]string, error) {
	if r.VerifyRelease {
		err := r.FetchReleaseFile()
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la récupération du fichier Release: %v", err)
		}
	}

	sections := r.Sections
	architectures := r.Architectures
	extensions := []string{"", ".gz", ".xz"}

	allPackages := make(map[string]bool)
	var lastErr error
	foundAtLeastOne := false

	for _, section := range sections {
		for _, arch := range architectures {
			for _, ext := range extensions {
				packagesURL := r.buildPackagesURLWithDist(r.Distribution, section, arch) + ext

				resp, err := http.Head(packagesURL)
				if err != nil {
					lastErr = err
					continue
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					lastErr = fmt.Errorf("impossible de récupérer le fichier Packages depuis %s (HTTP %d)", packagesURL, resp.StatusCode)
					continue
				}

				var packages []string
				if ext == "" {
					packages, err = r.downloadAndParsePackagesWithVerification(packagesURL, section, arch)
				} else {
					packages, err = r.downloadAndParseCompressedPackagesWithVerification(packagesURL, ext, section, arch)
				}

				if err != nil {
					lastErr = err
					continue
				}

				for _, pkg := range packages {
					allPackages[pkg] = true
				}
				foundAtLeastOne = true

				break
			}
		}
	}
	if !foundAtLeastOne {
		return nil, fmt.Errorf("impossible de récupérer les paquets depuis la distribution %s: %v", r.Distribution, lastErr)
	}
	result := make([]string, 0, len(allPackages))
	for pkg := range allPackages {
		result = append(result, pkg)
	}

	r.Packages = result

	return result, nil
}

func (r *Repository) SearchPackage(packageName string) ([]string, error) {
	if len(r.Packages) == 0 {
		return nil, fmt.Errorf("aucun paquet disponible - appelez d'abord FetchPackages()")
	}

	packageNameLower := strings.ToLower(packageName)

	var exactMatches []string
	var partialMatches []string

	for _, pkg := range r.Packages {
		if pkg == packageName {
			exactMatches = append(exactMatches, pkg)
		} else {
			pkgLower := strings.ToLower(pkg)
			if pkgLower == packageNameLower {
				exactMatches = append(exactMatches, pkg)
			} else if strings.Contains(pkgLower, packageNameLower) {
				partialMatches = append(partialMatches, pkg)
			}
		}
	}

	if len(exactMatches) == 0 && len(partialMatches) == 0 {
		return nil, fmt.Errorf("aucun paquet trouvé pour '%s' dans la distribution %s", packageName, r.Distribution)
	}

	// Concatenate exact matches first, then partial matches
	result := make([]string, 0, len(exactMatches)+len(partialMatches))
	result = append(result, exactMatches...)
	result = append(result, partialMatches...)

	return result, nil
}

func (r *Repository) DownloadPackage(packageName, version, architecture, destDir string) error {
	packageURL := r.buildPackageURL(packageName, version, architecture)

	pkg := &Package{
		Name:         packageName,
		Version:      version,
		Architecture: architecture,
		DownloadURL:  packageURL,
		Filename:     fmt.Sprintf("%s_%s_%s.deb", packageName, version, architecture),
	}

	downloader := NewDownloader()
	return downloader.DownloadToDirSilent(pkg, destDir)
}

func (r *Repository) DownloadPackageByURL(packageURL, destDir string) error {
	parts := strings.Split(packageURL, "/")
	filename := parts[len(parts)-1]
	pkg := &Package{
		Name:        strings.Split(filename, "_")[0],
		DownloadURL: packageURL,
		Filename:    filename,
	}

	downloader := NewDownloader()
	return downloader.DownloadToDirSilent(pkg, destDir)
}

func (r *Repository) buildPackageURL(packageName, version, architecture string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	filename := fmt.Sprintf("%s_%s_%s.deb", packageName, version, architecture)

	// pool/main/p/packagename/package_version_architecture.deb
	firstLetter := string(packageName[0])

	if len(packageName) > 3 && strings.HasPrefix(packageName, "lib") {
		if len(packageName) >= 4 {
			firstLetter = packageName[:4] // for libXXX
		}
	}

	section := "main"

	return fmt.Sprintf("%s/pool/%s/%s/%s/%s", baseURL, section, firstLetter, packageName, filename)
}

func (r *Repository) buildPackageURLWithSection(packageName, version, architecture, section string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	filename := fmt.Sprintf("%s_%s_%s.deb", packageName, version, architecture)

	firstLetter := string(packageName[0])
	if len(packageName) > 3 && strings.HasPrefix(packageName, "lib") {
		if len(packageName) >= 4 {
			firstLetter = packageName[:4]
		}
	}

	return fmt.Sprintf("%s/pool/%s/%s/%s/%s", baseURL, section, firstLetter, packageName, filename)
}

func (r *Repository) CheckPackageAvailability(packageName, version, architecture string) (bool, error) {
	packageURL := r.buildPackageURL(packageName, version, architecture)

	resp, err := http.Head(packageURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (r *Repository) DownloadPackageFromSources(packageName, version, architecture, destDir string, sections []string) error {
	if len(sections) == 0 {
		sections = []string{"main", "contrib", "non-free"}
	}

	var lastErr error

	for _, section := range sections {
		url := r.buildPackageURLWithSection(packageName, version, architecture, section)

		resp, err := http.Head(url)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			pkg := &Package{
				Name:         packageName,
				Version:      version,
				Architecture: architecture,
				DownloadURL:  url,
				Filename:     fmt.Sprintf("%s_%s_%s.deb", packageName, version, architecture),
			}

			downloader := NewDownloader()
			return downloader.DownloadToDirSilent(pkg, destDir)
		}

		lastErr = fmt.Errorf("paquet non trouvé dans la section %s (HTTP %d)", section, resp.StatusCode)
	}

	return fmt.Errorf("paquet %s_%s_%s non trouvé dans aucune section: %v", packageName, version, architecture, lastErr)
}

func (r *Repository) SearchPackageInSources(packageName, version, architecture string) (*PackageInfo, error) {
	sections := []string{"main", "contrib", "non-free"}

	for _, section := range sections {
		url := r.buildPackageURLWithSection(packageName, version, architecture, section)

		resp, err := http.Head(url)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return &PackageInfo{
				Name:         packageName,
				Version:      version,
				Architecture: architecture,
				Section:      section,
				DownloadURL:  url,
				Size:         resp.ContentLength,
			}, nil
		}
	}

	return nil, fmt.Errorf("paquet %s_%s_%s non trouvé", packageName, version, architecture)
}

func (r *Repository) buildPackagesURLWithDist(distribution, section, architecture string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", baseURL, distribution, section, architecture)
}

func (r *Repository) EnableReleaseVerification() {
	r.VerifyRelease = true
}

func (r *Repository) DisableReleaseVerification() {
	r.VerifyRelease = false
}

func (r *Repository) GetReleaseInfo() *ReleaseFile {
	return r.ReleaseInfo
}

func (r *Repository) IsReleaseVerificationEnabled() bool {
	return r.VerifyRelease
}

func (r *Repository) downloadAndParsePackagesWithVerification(packagesURL, section, architecture string) ([]string, error) {
	resp, err := http.Get(packagesURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération du fichier Packages: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("impossible de récupérer le fichier Packages (HTTP %d)", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du fichier Packages: %v", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		err = r.VerifyPackagesFileChecksum(section, architecture, data)
		if err != nil {
			return nil, fmt.Errorf("échec de la vérification du checksum: %v", err)
		}
	}

	return r.parsePackagesData(data)
}

func (r *Repository) downloadAndParseCompressedPackagesWithVerification(packagesURL, extension, section, architecture string) ([]string, error) {
	resp, err := http.Get(packagesURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération du fichier Packages compressé: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("impossible de récupérer le fichier Packages compressé (HTTP %d)", resp.StatusCode)
	}

	var reader io.Reader

	switch extension {
	case ".gz":
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la décompression gzip: %v", err)
		}
		defer gzReader.Close()
		reader = gzReader

	case ".xz":
		xzReader, err := xz.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la décompression xz: %v", err)
		}
		reader = xzReader

	default:
		return nil, fmt.Errorf("format de compression non supporté: %s", extension)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du fichier Packages décompressé: %v", err)
	}

	if r.VerifyRelease && r.ReleaseInfo != nil {
		filename := fmt.Sprintf("%s/binary-%s/Packages", section, architecture)
		err = r.verifyDecompressedFileChecksum(filename, data)
		if err != nil {
			return nil, fmt.Errorf("échec de la vérification du checksum décompressé: %v", err)
		}
	}

	return r.parsePackagesData(data)
}

func (r *Repository) parsePackagesData(data []byte) ([]string, error) {
	var packages []string
	var packageMetadata []Package

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // Buffer 1MB

	var currentPackage *Package

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Empty line indicates end of current package block
		if trimmedLine == "" {
			if currentPackage != nil && currentPackage.Name != "" {
				// Ensure source name is set (fallback to package name if not specified)
				if currentPackage.Source == "" {
					currentPackage.Source = currentPackage.Name
				}

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
		parts := strings.SplitN(trimmedLine, ":", 2)
		if len(parts) != 2 {
			continue
		}

		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Start new package block
		if field == "Package" {
			currentPackage = &Package{
				Name:    value, // For compatibility
				Package: value, // Official Debian field name
			}
			continue
		}

		// Skip if no current package
		if currentPackage == nil {
			continue
		}

		// Parse fields
		switch field {
		case "Version":
			currentPackage.Version = value
		case "Architecture":
			currentPackage.Architecture = value
		case "Maintainer":
			currentPackage.Maintainer = value
		case "Description":
			currentPackage.Description = value
		case "Filename":
			currentPackage.Filename = value
			// Construct download URL from repository base URL and filename
			baseURL := strings.TrimSuffix(r.URL, "/")
			currentPackage.DownloadURL = fmt.Sprintf("%s/%s", baseURL, value)
		case "Size":
			if size, err := strconv.ParseInt(value, 10, 64); err == nil {
				currentPackage.Size = size
			}
		case "Source":
			currentPackage.Source = value
		case "MD5sum":
			currentPackage.MD5sum = value
		case "SHA1":
			currentPackage.SHA1 = value
		case "SHA256":
			currentPackage.SHA256 = value
		// Nouveaux champs ajoutés
		case "Section":
			currentPackage.Section = value
		case "Priority":
			currentPackage.Priority = value
		case "Essential":
			currentPackage.Essential = value
		case "Depends":
			currentPackage.Depends = parsePackageList(value)
		case "Pre-Depends":
			currentPackage.PreDepends = parsePackageList(value)
		case "Recommends":
			currentPackage.Recommends = parsePackageList(value)
		case "Suggests":
			currentPackage.Suggests = parsePackageList(value)
		case "Enhances":
			currentPackage.Enhances = parsePackageList(value)
		case "Breaks":
			currentPackage.Breaks = parsePackageList(value)
		case "Conflicts":
			currentPackage.Conflicts = parsePackageList(value)
		case "Provides":
			currentPackage.Provides = parsePackageList(value)
		case "Replaces":
			currentPackage.Replaces = parsePackageList(value)
		case "Installed-Size":
			currentPackage.InstalledSize = value
		case "Homepage":
			currentPackage.Homepage = value
		case "Built-Using":
			currentPackage.BuiltUsing = value
		case "Package-Type":
			currentPackage.PackageType = value
		case "Multi-Arch":
			currentPackage.MultiArch = value
		case "Origin":
			currentPackage.Origin = value
		case "Bugs":
			currentPackage.Bugs = value
		// Additional Debian package fields
		case "Tag":
			currentPackage.Tag = value
		case "Task":
			currentPackage.Task = value
		case "Uploaders":
			currentPackage.Uploaders = value
		case "Standards-Version":
			currentPackage.StandardsVersion = value
		case "Vcs-Git":
			currentPackage.VcsGit = value
		case "Vcs-Browser":
			currentPackage.VcsBrowser = value
		case "Testsuite":
			currentPackage.Testsuite = value
		case "Auto-Built":
			currentPackage.AutoBuilt = value
		case "Build-Essential":
			currentPackage.BuildEssential = value
		case "Important":
			currentPackage.ImportantDescription = value
		case "Description-md5":
			currentPackage.DescriptionMd5 = value
		case "Gstreamer-Version":
			currentPackage.Gstreamer = value
		case "Python-Version":
			currentPackage.PythonVersion = value
		// Maintainer script fields
		case "Preinst":
			currentPackage.Preinst = value
		case "Postinst":
			currentPackage.Postinst = value
		case "Prerm":
			currentPackage.Prerm = value
		case "Postrm":
			currentPackage.Postrm = value
		// Custom fields (X- prefixed)
		default:
			if strings.HasPrefix(field, "X-") {
				if currentPackage.CustomFields == nil {
					currentPackage.CustomFields = make(map[string]string)
				}
				currentPackage.CustomFields[field] = value
			}
		}
	}

	// Handle last package if file doesn't end with empty line
	if currentPackage != nil && currentPackage.Name != "" {
		if currentPackage.Source == "" {
			currentPackage.Source = currentPackage.Name
		}
		packageMetadata = append(packageMetadata, *currentPackage)
		packages = append(packages, currentPackage.Name)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du fichier Packages: %v", err)
	}

	// Store metadata in repository
	r.PackageMetadata = packageMetadata

	return packages, nil
}

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

func (r *Repository) SetDistribution(distribution string) {
	r.Distribution = distribution
}

func (r *Repository) SetSections(sections []string) {
	r.Sections = sections
}

func (r *Repository) SetArchitectures(architectures []string) {
	r.Architectures = architectures
}

func (r *Repository) AddSection(section string) {
	r.Sections = append(r.Sections, section)
}

func (r *Repository) AddArchitecture(architecture string) {
	r.Architectures = append(r.Architectures, architecture)
}

// GetPackageMetadata returns the complete metadata for a specific package
func (r *Repository) GetPackageMetadata(packageName string) (*Package, error) {
	if len(r.PackageMetadata) == 0 {
		return nil, fmt.Errorf("aucune métadonnée de paquet disponible - appelez d'abord FetchPackages()")
	}

	for _, pkg := range r.PackageMetadata {
		if pkg.Name == packageName {
			return &pkg, nil
		}
	}

	return nil, fmt.Errorf("paquet '%s' non trouvé dans les métadonnées", packageName)
}

// GetAllPackageMetadata returns all package metadata
func (r *Repository) GetAllPackageMetadata() []Package {
	return r.PackageMetadata
}

type PackageInfo struct {
	Name         string
	Version      string
	Architecture string
	Section      string
	DownloadURL  string
	Size         int64
}

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

type FileChecksum struct {
	Hash     string
	Size     int64
	Filename string
}

func (r *Repository) FetchReleaseFile() error {
	releaseURL := r.buildReleaseURL()

	resp, err := http.Get(releaseURL)
	if err != nil {
		return fmt.Errorf("erreur lors de la récupération du fichier Release: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("impossible de récupérer le fichier Release (HTTP %d)", resp.StatusCode)
	}

	releaseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erreur lors de la lecture du fichier Release: %v", err)
	}

	releaseInfo, err := r.parseReleaseFile(string(releaseData))
	if err != nil {
		return fmt.Errorf("erreur lors du parsing du fichier Release: %v", err)
	}

	r.ReleaseInfo = releaseInfo
	return nil
}

func (r *Repository) buildReleaseURL() string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/Release", baseURL, r.Distribution)
}

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
		return nil, fmt.Errorf("taille invalide dans la ligne de checksum: %s", sizeStr)
	}

	return &FileChecksum{
		Hash:     hash,
		Size:     size,
		Filename: filename,
	}, nil
}

func (r *Repository) VerifyPackagesFileChecksum(section, architecture string, data []byte) error {
	if r.ReleaseInfo == nil {
		return fmt.Errorf("informations Release non disponibles - appelez d'abord FetchReleaseFile()")
	}

	filename := fmt.Sprintf("%s/binary-%s/Packages", section, architecture)

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
