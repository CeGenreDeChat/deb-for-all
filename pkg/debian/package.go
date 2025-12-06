package debian

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// File permission constants.
const (
	dirPermission  = 0755
	filePermission = 0644
)

// Package represents a Debian binary package with all standard control file fields.
// It is the central abstraction for package metadata in the library.
type Package struct {
	// Identification fields
	Name         string
	Package      string
	Version      string
	Architecture string
	Maintainer   string
	Description  string

	// Download and file information
	DownloadURL string
	Filename    string
	Size        int64
	MD5sum      string
	SHA1        string
	SHA256      string

	// Classification fields
	Source        string
	Section       string
	Priority      string
	Essential     string
	InstalledSize string
	Homepage      string
	BuiltUsing    string
	PackageType   string

	// Dependency fields
	Depends    []string
	PreDepends []string
	Recommends []string
	Suggests   []string
	Enhances   []string
	Breaks     []string
	Conflicts  []string
	Provides   []string
	Replaces   []string

	// Additional metadata fields
	Tag                  string
	Task                 string
	Uploaders            string
	StandardsVersion     string
	VcsGit               string
	VcsBrowser           string
	Testsuite            string
	AutoBuilt            string
	BuildEssential       string
	ImportantDescription string
	DescriptionMd5       string
	Gstreamer            string
	PythonVersion        string

	// Maintainer script fields
	Preinst  string
	Postinst string
	Prerm    string
	Postrm   string

	// Multi-arch support
	MultiArch string

	// Origin and distribution
	Origin string
	Bugs   string

	// Custom fields (X- prefixed or unknown)
	CustomFields map[string]string
}

// SourcePackage represents a Debian source package with its associated files.
type SourcePackage struct {
	Name        string
	Version     string
	Maintainer  string
	Description string
	Directory   string       // Pool path (e.g., pool/main/h/hello)
	Files       []SourceFile // Associated source files
}

// SourceFile represents a single file within a source package.
type SourceFile struct {
	Name      string
	URL       string
	Size      int64
	MD5Sum    string
	SHA256Sum string
	Type      string // "orig", "debian", "dsc", etc.
}

// DownloadInfo contains HTTP metadata for package downloads.
type DownloadInfo struct {
	URL           string
	ContentLength int64
	ContentType   string
	LastModified  string
}

// NewPackage creates a new Package with the required fields.
func NewPackage(name, version, architecture, maintainer, description, downloadURL, filename string, size int64) *Package {
	return &Package{
		Name:         name,
		Package:      name,
		Version:      version,
		Architecture: architecture,
		Maintainer:   maintainer,
		Description:  description,
		DownloadURL:  downloadURL,
		Filename:     filename,
		Size:         size,
		Source:       name,
	}
}

// NewSourcePackage creates a new SourcePackage with the required fields.
func NewSourcePackage(name, version, maintainer, description, directory string) *SourcePackage {
	return &SourcePackage{
		Name:        name,
		Version:     version,
		Maintainer:  maintainer,
		Description: description,
		Directory:   directory,
		Files:       make([]SourceFile, 0),
	}
}

// AddFile adds a source file to the package.
func (sp *SourcePackage) AddFile(name, url string, size int64, md5sum, sha256sum, fileType string) {
	sp.Files = append(sp.Files, SourceFile{
		Name:      name,
		URL:       url,
		Size:      size,
		MD5Sum:    md5sum,
		SHA256Sum: sha256sum,
		Type:      fileType,
	})
}

// GetOrigTarball returns the original source tarball (.orig.tar.*) if present.
func (sp *SourcePackage) GetOrigTarball() *SourceFile {
	return sp.findFileByType("orig", ".orig.tar")
}

// GetDebianTarball returns the Debian tarball (.debian.tar.*) if present.
func (sp *SourcePackage) GetDebianTarball() *SourceFile {
	return sp.findFileByType("debian", ".debian.tar")
}

// GetDSCFile returns the Debian Source Control file (.dsc) if present.
func (sp *SourcePackage) GetDSCFile() *SourceFile {
	return sp.findFileByType("dsc", ".dsc")
}

// findFileByType searches for a file by type or name pattern.
func (sp *SourcePackage) findFileByType(fileType, namePattern string) *SourceFile {
	for i := range sp.Files {
		if sp.Files[i].Type == fileType || strings.Contains(sp.Files[i].Name, namePattern) {
			return &sp.Files[i]
		}
	}
	return nil
}

// Download downloads all source files to the destination directory with progress output.
func (sp *SourcePackage) Download(destDir string) error {
	return sp.downloadFiles(destDir, true, nil)
}

// DownloadSilent downloads all source files without any output.
func (sp *SourcePackage) DownloadSilent(destDir string) error {
	return sp.downloadFiles(destDir, false, nil)
}

// DownloadWithProgress downloads all source files with a progress callback.
func (sp *SourcePackage) DownloadWithProgress(destDir string, progressCallback func(filename string, downloaded, total int64)) error {
	return sp.downloadFiles(destDir, true, progressCallback)
}

// downloadFiles is the internal implementation for downloading source files.
func (sp *SourcePackage) downloadFiles(destDir string, verbose bool, progressCallback func(string, int64, int64)) error {
	if len(sp.Files) == 0 {
		return fmt.Errorf("aucun fichier à télécharger pour le paquet source %s", sp.Name)
	}

	if err := os.MkdirAll(destDir, dirPermission); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %w", err)
	}

	downloader := NewDownloader()

	for _, file := range sp.Files {
		if err := sp.downloadSingleFile(downloader, file, destDir, verbose, progressCallback); err != nil {
			return err
		}
	}

	if verbose {
		fmt.Printf("Paquet source %s téléchargé avec succès vers %s\n", sp.Name, destDir)
	}

	return nil
}

// downloadSingleFile downloads and verifies a single source file.
func (sp *SourcePackage) downloadSingleFile(downloader *Downloader, file SourceFile, destDir string, verbose bool, progressCallback func(string, int64, int64)) error {
	destPath := filepath.Join(destDir, file.Name)

	if verbose {
		fmt.Printf("Téléchargement de %s...\n", file.Name)
	}

	// Use downloadToFile directly instead of creating a temp Package
	var err error
	if progressCallback != nil {
		err = downloader.downloadToFile(file.URL, destPath, func(downloaded, total int64) {
			progressCallback(file.Name, downloaded, total)
		})
	} else {
		err = downloader.downloadToFile(file.URL, destPath, nil)
	}

	if err != nil {
		return fmt.Errorf("erreur lors du téléchargement de %s: %w", file.Name, err)
	}

	// Verify checksum
	if file.SHA256Sum != "" {
		if err := downloader.verifyChecksum(destPath, file.SHA256Sum, "sha256"); err != nil {
			return fmt.Errorf("erreur de vérification SHA256 pour %s: %w", file.Name, err)
		}
	} else if file.MD5Sum != "" {
		if err := downloader.verifyChecksum(destPath, file.MD5Sum, "md5"); err != nil {
			return fmt.Errorf("erreur de vérification MD5 pour %s: %w", file.Name, err)
		}
	}

	return nil
}

// String returns a string representation of the source package.
func (sp *SourcePackage) String() string {
	return fmt.Sprintf("%s (%s) - %s [%d fichiers]", sp.Name, sp.Version, sp.Description, len(sp.Files))
}

// GetSourceName returns the source package name, falling back to the package name.
func (p *Package) GetSourceName() string {
	if p.Source == "" {
		return p.Name
	}
	return p.Source
}

// GetDownloadInfo fetches HTTP metadata for the package via a HEAD request.
func (p *Package) GetDownloadInfo() (*DownloadInfo, error) {
	if p.DownloadURL == "" {
		return nil, fmt.Errorf("aucune URL de téléchargement spécifiée pour le paquet %s", p.Name)
	}

	resp, err := http.Head(p.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la vérification de l'URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("URL non accessible: statut HTTP %d", resp.StatusCode)
	}

	return &DownloadInfo{
		URL:           p.DownloadURL,
		ContentLength: resp.ContentLength,
		ContentType:   resp.Header.Get("Content-Type"),
		LastModified:  resp.Header.Get("Last-Modified"),
	}, nil
}

// ReadControlFile parses a Debian control file and returns a Package.
func ReadControlFile(filePath string) (*Package, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("erreur de lecture du fichier control: %w", err)
	}
	return parseControlData(string(data))
}

// WriteControlFile writes the package metadata to a control file.
func (p *Package) WriteControlFile(filePath string) error {
	content := p.FormatAsControl()
	if err := os.WriteFile(filePath, []byte(content), filePermission); err != nil {
		return fmt.Errorf("erreur d'écriture du fichier control: %w", err)
	}
	return nil
}

// FormatAsControl formats the package metadata as a Debian control file string.
func (p *Package) FormatAsControl() string {
	var sb strings.Builder

	requiredFields := []struct {
		name  string
		value string
	}{
		{"Package", p.Package},
		{"Version", p.Version},
		{"Architecture", p.Architecture},
		{"Maintainer", p.Maintainer},
	}

	for _, field := range requiredFields {
		sb.WriteString(field.name + ": " + field.value + "\n")
	}

	optionalFields := []struct {
		name  string
		value string
	}{
		{"Source", p.Source},
		{"Section", p.Section},
		{"Priority", p.Priority},
		{"Essential", p.Essential},
		{"Installed-Size", p.InstalledSize},
		{"Homepage", p.Homepage},
		{"Built-Using", p.BuiltUsing},
		{"Package-Type", p.PackageType},
		{"Multi-Arch", p.MultiArch},
		{"Origin", p.Origin},
		{"Bugs", p.Bugs},
		{"Preinst", p.Preinst},
		{"Postinst", p.Postinst},
		{"Prerm", p.Prerm},
		{"Postrm", p.Postrm},
		{"Tag", p.Tag},
		{"Task", p.Task},
		{"Uploaders", p.Uploaders},
		{"Standards-Version", p.StandardsVersion},
		{"Vcs-Git", p.VcsGit},
		{"Vcs-Browser", p.VcsBrowser},
		{"Testsuite", p.Testsuite},
		{"Auto-Built", p.AutoBuilt},
		{"Build-Essential", p.BuildEssential},
		{"Important", p.ImportantDescription},
		{"Description-md5", p.DescriptionMd5},
		{"Gstreamer-Version", p.Gstreamer},
		{"Python-Version", p.PythonVersion},
	}

	for _, field := range optionalFields {
		if field.value != "" {
			sb.WriteString(field.name + ": " + field.value + "\n")
		}
	}

	relationshipFields := []struct {
		name  string
		value []string
	}{
		{"Depends", p.Depends},
		{"Pre-Depends", p.PreDepends},
		{"Recommends", p.Recommends},
		{"Suggests", p.Suggests},
		{"Enhances", p.Enhances},
		{"Breaks", p.Breaks},
		{"Conflicts", p.Conflicts},
		{"Provides", p.Provides},
		{"Replaces", p.Replaces},
	}

	for _, field := range relationshipFields {
		if len(field.value) > 0 {
			sb.WriteString(field.name + ": " + strings.Join(field.value, ", ") + "\n")
		}
	}

	if p.CustomFields != nil {
		for field, value := range p.CustomFields {
			sb.WriteString(field + ": " + value + "\n")
		}
	}

	if p.Description != "" {
		sb.WriteString("Description: " + p.Description + "\n")
	}

	return sb.String()
}

// controlFieldMapping maps control file field names to Package field setters.
// This is used for efficient parsing without a large switch statement.
var controlFieldMapping = map[string]func(*Package, string){
	"package":           func(p *Package, v string) { p.Package = v; p.Name = v },
	"version":           func(p *Package, v string) { p.Version = v },
	"architecture":      func(p *Package, v string) { p.Architecture = v },
	"maintainer":        func(p *Package, v string) { p.Maintainer = v },
	"description":       func(p *Package, v string) { p.Description = v },
	"source":            func(p *Package, v string) { p.Source = v },
	"section":           func(p *Package, v string) { p.Section = v },
	"priority":          func(p *Package, v string) { p.Priority = v },
	"essential":         func(p *Package, v string) { p.Essential = v },
	"installed-size":    func(p *Package, v string) { p.InstalledSize = v },
	"homepage":          func(p *Package, v string) { p.Homepage = v },
	"built-using":       func(p *Package, v string) { p.BuiltUsing = v },
	"package-type":      func(p *Package, v string) { p.PackageType = v },
	"multi-arch":        func(p *Package, v string) { p.MultiArch = v },
	"origin":            func(p *Package, v string) { p.Origin = v },
	"bugs":              func(p *Package, v string) { p.Bugs = v },
	"preinst":           func(p *Package, v string) { p.Preinst = v },
	"postinst":          func(p *Package, v string) { p.Postinst = v },
	"prerm":             func(p *Package, v string) { p.Prerm = v },
	"postrm":            func(p *Package, v string) { p.Postrm = v },
	"tag":               func(p *Package, v string) { p.Tag = v },
	"task":              func(p *Package, v string) { p.Task = v },
	"uploaders":         func(p *Package, v string) { p.Uploaders = v },
	"standards-version": func(p *Package, v string) { p.StandardsVersion = v },
	"vcs-git":           func(p *Package, v string) { p.VcsGit = v },
	"vcs-browser":       func(p *Package, v string) { p.VcsBrowser = v },
	"testsuite":         func(p *Package, v string) { p.Testsuite = v },
	"auto-built":        func(p *Package, v string) { p.AutoBuilt = v },
	"build-essential":   func(p *Package, v string) { p.BuildEssential = v },
	"important":         func(p *Package, v string) { p.ImportantDescription = v },
	"description-md5":   func(p *Package, v string) { p.DescriptionMd5 = v },
	"gstreamer-version": func(p *Package, v string) { p.Gstreamer = v },
	"python-version":    func(p *Package, v string) { p.PythonVersion = v },
}

// dependencyFieldMapping maps dependency field names to Package slice setters.
var dependencyFieldMapping = map[string]func(*Package, []string){
	"depends":     func(p *Package, v []string) { p.Depends = v },
	"pre-depends": func(p *Package, v []string) { p.PreDepends = v },
	"recommends":  func(p *Package, v []string) { p.Recommends = v },
	"suggests":    func(p *Package, v []string) { p.Suggests = v },
	"enhances":    func(p *Package, v []string) { p.Enhances = v },
	"breaks":      func(p *Package, v []string) { p.Breaks = v },
	"conflicts":   func(p *Package, v []string) { p.Conflicts = v },
	"provides":    func(p *Package, v []string) { p.Provides = v },
	"replaces":    func(p *Package, v []string) { p.Replaces = v },
}

// parseControlData parses a Debian control file content into a Package.
func parseControlData(content string) (*Package, error) {
	lines := strings.Split(content, "\n")
	pkg := &Package{
		CustomFields: make(map[string]string),
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			continue
		}

		field := strings.TrimSpace(line[:colonIndex])
		value := strings.TrimSpace(line[colonIndex+1:])
		fieldLower := strings.ToLower(field)

		// Check for string field setter
		if setter, ok := controlFieldMapping[fieldLower]; ok {
			setter(pkg, value)
			continue
		}

		// Check for dependency field setter
		if setter, ok := dependencyFieldMapping[fieldLower]; ok {
			setter(pkg, parsePackageList(value))
			continue
		}

		// Unknown field - store in CustomFields
		pkg.CustomFields[field] = value
	}

	if pkg.Package == "" || pkg.Version == "" || pkg.Architecture == "" || pkg.Maintainer == "" {
		return nil, errors.New("invalid control file: missing required fields (Package, Version, Architecture, Maintainer)")
	}

	return pkg, nil
}

// parsePackageList parses a comma-separated dependency list.
func parsePackageList(value string) []string {
	if value == "" {
		return nil
	}

	packages := strings.Split(value, ",")
	result := make([]string, len(packages))
	for i, pkg := range packages {
		result[i] = strings.TrimSpace(pkg)
	}
	return result
}
