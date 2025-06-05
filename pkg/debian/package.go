package debian

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

func (p *Package) String() string {
	return p.Name + " (" + p.Version + ") - " + p.Description
}

// Download télécharge le paquet Debian vers le répertoire spécifié avec affichage de confirmation.
func (p *Package) Download(destDir string) error {
	return p.downloadToDir(destDir, true)
}

// DownloadSilent télécharge le paquet Debian vers le répertoire spécifié sans affichage console.
func (p *Package) DownloadSilent(destDir string) error {
	return p.downloadToDir(destDir, false)
}

// downloadToDir est la fonction interne commune pour le téléchargement vers un répertoire.
func (p *Package) downloadToDir(destDir string, verbose bool) error {
	if p.DownloadURL == "" {
		return fmt.Errorf("aucune URL de téléchargement spécifiée pour le paquet %s", p.Name)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %v", err)
	}

	filename := p.Filename
	if filename == "" {
		filename = fmt.Sprintf("%s_%s_%s.deb", p.Name, p.Version, p.Architecture)
	}

	destPath := filepath.Join(destDir, filename)

	resp, err := http.Get(p.DownloadURL)
	if err != nil {
		return fmt.Errorf("erreur lors du téléchargement: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("échec du téléchargement: statut HTTP %d", resp.StatusCode)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier de destination: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		return fmt.Errorf("erreur lors de la copie du fichier: %v", err)
	}

	if verbose {
		fmt.Printf("Paquet %s téléchargé avec succès vers %s\n", p.Name, destPath)
	}
	return nil
}

// DownloadToFile télécharge le paquet Debian vers le fichier spécifié avec affichage de confirmation.
func (p *Package) DownloadToFile(filePath string) error {
	return p.downloadToFile(filePath, true)
}

// DownloadToFileSilent télécharge le paquet Debian vers le fichier spécifié sans affichage console.
func (p *Package) DownloadToFileSilent(filePath string) error {
	return p.downloadToFile(filePath, false)
}

// downloadToFile est la fonction interne commune pour le téléchargement vers un fichier spécifique.
func (p *Package) downloadToFile(filePath string, verbose bool) error {
	if p.DownloadURL == "" {
		return fmt.Errorf("aucune URL de téléchargement spécifiée pour le paquet %s", p.Name)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire parent: %v", err)
	}

	resp, err := http.Get(p.DownloadURL)
	if err != nil {
		return fmt.Errorf("erreur lors du téléchargement: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("échec du téléchargement: statut HTTP %d", resp.StatusCode)
	}

	destFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier de destination: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		return fmt.Errorf("erreur lors de la copie du fichier: %v", err)
	}

	if verbose {
		fmt.Printf("Paquet %s téléchargé avec succès vers %s\n", p.Name, filePath)
	}
	return nil
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
