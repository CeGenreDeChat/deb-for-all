package debian

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Package struct {
	// Required fields (package identification)
	Name         string // Nom du paquet (alias pour Package)
	Package      string // Nom du paquet (champ officiel Debian)
	Version      string
	Architecture string
	Maintainer   string
	Description  string

	// Download and file information
	DownloadURL string // URL de téléchargement
	Filename    string // Nom du fichier .deb
	Size        int64  // Taille du paquet en bytes
	MD5sum      string // Somme de contrôle MD5
	SHA1        string // Somme de contrôle SHA1
	SHA256      string // Somme de contrôle SHA256

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
	Tag                  string // Debtags information
	Task                 string // Task information
	Uploaders            string // Additional uploaders
	StandardsVersion     string // Standards version
	VcsGit               string // Version control system git URL
	VcsBrowser           string // Version control browser URL
	Testsuite            string // Test suite information
	AutoBuilt            string // Auto-built information
	BuildEssential       string // Build essential flag
	ImportantDescription string // Important description
	DescriptionMd5       string // Description MD5 hash
	Gstreamer            string // GStreamer information
	PythonVersion        string // Python version information

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

		// Créer un paquet temporaire pour utiliser le downloader existant
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

type DownloadInfo struct {
	URL           string
	ContentLength int64
	ContentType   string
	LastModified  string
}
