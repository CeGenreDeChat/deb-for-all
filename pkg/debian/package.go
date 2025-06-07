package debian

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Package struct {
	Name         string
	Version      string
	Architecture string
	Maintainer   string
	Description  string
	DownloadURL  string
	Filename     string
	Size         int64
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
		Version:      version,
		Architecture: architecture,
		Maintainer:   maintainer,
		Description:  description,
		DownloadURL:  downloadURL,
		Filename:     filename,
		Size:         size,
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

		// Vérifier les sommes de contrôle si disponibles
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

// Package now only contains metadata - all download functionality moved to Downloader

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
