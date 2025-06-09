package debian

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Package struct {
	// Required fields (package identification)
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

	// Optional fields
	Source        string
	Section       string
	Priority      string
	Essential     string
	Depends       []string
	PreDepends    []string
	Recommends    []string
	Suggests      []string
	Enhances      []string
	Breaks        []string
	Conflicts     []string
	Provides      []string
	Replaces      []string
	InstalledSize string
	Homepage      string
	BuiltUsing    string
	PackageType   string

	// Additional Debian package fields
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

	// Custom fields (X- prefixed)
	CustomFields map[string]string
}

type SourcePackage struct {
	Name        string
	Version     string
	Maintainer  string
	Description string
	Directory   string
	Files       []SourceFile
}

type SourceFile struct {
	Name      string
	URL       string
	Size      int64
	MD5Sum    string
	SHA256Sum string
	Type      string // "orig", "debian", "dsc", etc.
}

func NewPackage(name, version, architecture, maintainer, description, downloadURL, filename string, size int64) *Package {
	return &Package{
		Name:         name,
		Package:      name, // Ensure both fields are set
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

func (sp *SourcePackage) AddFile(name, url string, size int64, md5sum, sha256sum, fileType string) {
	file := SourceFile{
		Name:      name,
		URL:       url,
		Size:      size,
		MD5Sum:    md5sum,
		SHA256Sum: sha256sum,
		Type:      fileType,
	}
	sp.Files = append(sp.Files, file)
}

func (sp *SourcePackage) GetOrigTarball() *SourceFile {
	for _, file := range sp.Files {
		if file.Type == "orig" || strings.Contains(file.Name, ".orig.tar") {
			return &file
		}
	}
	return nil
}

func (sp *SourcePackage) GetDebianTarball() *SourceFile {
	for _, file := range sp.Files {
		if file.Type == "debian" || strings.Contains(file.Name, ".debian.tar") {
			return &file
		}
	}
	return nil
}

func (sp *SourcePackage) GetDSCFile() *SourceFile {
	for _, file := range sp.Files {
		if file.Type == "dsc" || strings.HasSuffix(file.Name, ".dsc") {
			return &file
		}
	}
	return nil
}

func (sp *SourcePackage) Download(destDir string) error {
	return sp.downloadFiles(destDir, true, nil)
}

func (sp *SourcePackage) DownloadSilent(destDir string) error {
	return sp.downloadFiles(destDir, false, nil)
}

func (sp *SourcePackage) DownloadWithProgress(destDir string, progressCallback func(filename string, downloaded, total int64)) error {
	return sp.downloadFiles(destDir, true, progressCallback)
}

func (sp *SourcePackage) downloadFiles(destDir string, verbose bool, progressCallback func(string, int64, int64)) error {
	if len(sp.Files) == 0 {
		return fmt.Errorf("aucun fichier à télécharger pour le paquet source %s", sp.Name)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %v", err)
	}

	downloader := NewDownloader()

	for _, file := range sp.Files {
		destPath := filepath.Join(destDir, file.Name)

		if verbose {
			fmt.Printf("Téléchargement de %s...\n", file.Name)
		}

		tempPkg := &Package{
			Name:        sp.Name,
			Package:     sp.Name, // Ensure both fields are set
			Version:     sp.Version,
			DownloadURL: file.URL,
			Filename:    file.Name,
			Size:        file.Size,
		}

		var err error
		if progressCallback != nil {
			err = downloader.DownloadWithProgress(tempPkg, destPath, func(downloaded, total int64) {
				progressCallback(file.Name, downloaded, total)
			})
		} else if verbose {
			err = downloader.DownloadWithProgress(tempPkg, destPath, nil)
		} else {
			err = downloader.DownloadSilent(tempPkg, destPath)
		}

		if err != nil {
			return fmt.Errorf("erreur lors du téléchargement de %s: %v", file.Name, err)
		}

		if file.SHA256Sum != "" {
			if err := downloader.verifyChecksum(destPath, file.SHA256Sum, "sha256"); err != nil {
				return fmt.Errorf("erreur de vérification SHA256 pour %s: %v", file.Name, err)
			}
		} else if file.MD5Sum != "" {
			if err := downloader.verifyChecksum(destPath, file.MD5Sum, "md5"); err != nil {
				return fmt.Errorf("erreur de vérification MD5 pour %s: %v", file.Name, err)
			}
		}
	}

	if verbose {
		fmt.Printf("Paquet source %s téléchargé avec succès vers %s\n", sp.Name, destDir)
	}

	return nil
}

func (sp *SourcePackage) String() string {
	return fmt.Sprintf("%s (%s) - %s [%d fichiers]", sp.Name, sp.Version, sp.Description, len(sp.Files))
}

func (p *Package) GetSourceName() string {
	if p.Source == "" {
		return p.Name
	}
	return p.Source
}

func (p *Package) GetDownloadInfo() (*DownloadInfo, error) {
	if p.DownloadURL == "" {
		return nil, fmt.Errorf("aucune URL de téléchargement spécifiée pour le paquet %s", p.Name)
	}

	resp, err := http.Head(p.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la vérification de l'URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("URL non accessible: statut HTTP %d", resp.StatusCode)
	}

	info := &DownloadInfo{
		URL:           p.DownloadURL,
		ContentLength: resp.ContentLength,
		ContentType:   resp.Header.Get("Content-Type"),
		LastModified:  resp.Header.Get("Last-Modified"),
	}

	return info, nil
}

func ReadControlFile(filePath string) (*Package, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return parseControlData(string(data))
}

func (p *Package) WriteControlFile(filePath string) error {
	content := p.FormatAsControl()
	return os.WriteFile(filePath, []byte(content), 0644)
}

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

		switch strings.ToLower(field) {
		case "package":
			pkg.Package = value
			pkg.Name = value // For compatibility
		case "version":
			pkg.Version = value
		case "architecture":
			pkg.Architecture = value
		case "maintainer":
			pkg.Maintainer = value
		case "description":
			pkg.Description = value
		case "source":
			pkg.Source = value
		case "section":
			pkg.Section = value
		case "priority":
			pkg.Priority = value
		case "essential":
			pkg.Essential = value
		case "depends":
			pkg.Depends = parsePackageList(value)
		case "pre-depends":
			pkg.PreDepends = parsePackageList(value)
		case "recommends":
			pkg.Recommends = parsePackageList(value)
		case "suggests":
			pkg.Suggests = parsePackageList(value)
		case "enhances":
			pkg.Enhances = parsePackageList(value)
		case "breaks":
			pkg.Breaks = parsePackageList(value)
		case "conflicts":
			pkg.Conflicts = parsePackageList(value)
		case "provides":
			pkg.Provides = parsePackageList(value)
		case "replaces":
			pkg.Replaces = parsePackageList(value)
		case "installed-size":
			pkg.InstalledSize = value
		case "homepage":
			pkg.Homepage = value
		case "built-using":
			pkg.BuiltUsing = value
		case "package-type":
			pkg.PackageType = value
		case "multi-arch":
			pkg.MultiArch = value
		case "origin":
			pkg.Origin = value
		case "bugs":
			pkg.Bugs = value
		case "preinst":
			pkg.Preinst = value
		case "postinst":
			pkg.Postinst = value
		case "prerm":
			pkg.Prerm = value
		case "postrm":
			pkg.Postrm = value
		case "tag":
			pkg.Tag = value
		case "task":
			pkg.Task = value
		case "uploaders":
			pkg.Uploaders = value
		case "standards-version":
			pkg.StandardsVersion = value
		case "vcs-git":
			pkg.VcsGit = value
		case "vcs-browser":
			pkg.VcsBrowser = value
		case "testsuite":
			pkg.Testsuite = value
		case "auto-built":
			pkg.AutoBuilt = value
		case "build-essential":
			pkg.BuildEssential = value
		case "important":
			pkg.ImportantDescription = value
		case "description-md5":
			pkg.DescriptionMd5 = value
		case "gstreamer-version":
			pkg.Gstreamer = value
		case "python-version":
			pkg.PythonVersion = value
		default:
			// Handle custom fields (X- prefixed or unknown fields)
			pkg.CustomFields[field] = value
		}
	}

	if pkg.Package == "" || pkg.Version == "" || pkg.Architecture == "" || pkg.Maintainer == "" {
		return nil, errors.New("invalid control file: missing required fields (Package, Version, Architecture, Maintainer)")
	}

	return pkg, nil
}

func parsePackageList(value string) []string {
	if value == "" {
		return nil
	}

	packages := strings.Split(value, ",")
	for i := range packages {
		packages[i] = strings.TrimSpace(packages[i])
	}
	return packages
}

type DownloadInfo struct {
	URL           string
	ContentLength int64
	ContentType   string
	LastModified  string
}
