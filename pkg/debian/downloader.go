package debian

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Downloader struct {
	UserAgent       string
	Timeout         time.Duration
	RetryAttempts   int
	VerifyChecksums bool
}

func NewDownloader() *Downloader {
	return &Downloader{
		UserAgent:       "deb-for-all/1.0",
		Timeout:         30 * time.Second,
		RetryAttempts:   3,
		VerifyChecksums: true,
	}
}

func (d *Downloader) DownloadWithProgress(pkg *Package, destPath string, progressCallback func(downloaded, total int64)) error {
	var err error
	var req *http.Request

	if pkg.DownloadURL == "" {
		return fmt.Errorf("aucune URL de téléchargement spécifiée pour le paquet %s", pkg.Name)
	}

	dir := filepath.Dir(destPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire parent: %v", err)
	}

	client := &http.Client{
		Timeout: d.Timeout,
	}
	var resp *http.Response
	var lastErr error

	for attempt := 1; attempt <= d.RetryAttempts; attempt++ {
		req, err = http.NewRequest("GET", pkg.DownloadURL, nil)
		if err != nil {
			return fmt.Errorf("erreur lors de la création de la requête: %v", err)
		}

		req.Header.Set("User-Agent", d.UserAgent)

		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("statut HTTP %d", resp.StatusCode)
		}

		if resp != nil {
			resp.Body.Close()
			resp = nil
		}

		if attempt < d.RetryAttempts {
			fmt.Printf("Tentative %d échouée, nouvelle tentative dans 2 secondes...\n", attempt)
			time.Sleep(2 * time.Second)
		}
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erreur lors du téléchargement après %d tentatives: %v", d.RetryAttempts, lastErr)
	}
	defer resp.Body.Close()

	var destFile *os.File

	destFile, err = os.Create(destPath)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier de destination: %v", err)
	}
	defer destFile.Close()

	totalSize := resp.ContentLength
	var downloaded int64
	var n int

	buffer := make([]byte, 32*1024) // Buffer de 32KB
	for {
		n, err = resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := destFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("erreur lors de l'écriture: %v", writeErr)
			}
			downloaded += int64(n)
			if progressCallback != nil {
				progressCallback(downloaded, totalSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("erreur lors de la lecture: %v", err)
		}
	}

	fmt.Printf("Paquet %s téléchargé avec succès vers %s\n", pkg.Name, destPath)
	return nil
}

func (d *Downloader) DownloadWithChecksum(pkg *Package, destPath, checksum, checksumType string) error {
	err := d.DownloadWithProgress(pkg, destPath, nil)
	if err != nil {
		return err
	}

	if d.VerifyChecksums && checksum != "" {
		return d.verifyChecksum(destPath, checksum, checksumType)
	}

	return nil
}

func (d *Downloader) verifyChecksum(filePath, expectedChecksum, checksumType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir le fichier pour vérification: %v", err)
	}
	defer file.Close()

	var hasher hash.Hash
	switch strings.ToLower(checksumType) {
	case "md5":
		hasher = md5.New()
	case "sha256":
		hasher = sha256.New()
	default:
		return fmt.Errorf("type de somme de contrôle non supporté: %s", checksumType)
	}

	_, err = io.Copy(hasher, file)
	if err != nil {
		return fmt.Errorf("erreur lors du calcul de la somme de contrôle: %v", err)
	}

	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("la somme de contrôle ne correspond pas. Attendue: %s, Actuelle: %s", expectedChecksum, actualChecksum)
	}

	fmt.Printf("Somme de contrôle %s vérifiée avec succès\n", checksumType)
	return nil
}

func (d *Downloader) DownloadMultiple(packages []*Package, destDir string, maxConcurrent int) []error {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	type downloadJob struct {
		pkg      *Package
		destPath string
	}

	type downloadResult struct {
		pkg *Package
		err error
	}

	jobs := make(chan downloadJob, len(packages))
	results := make(chan downloadResult, len(packages))

	for w := 0; w < maxConcurrent; w++ {
		go func() {
			for job := range jobs {
				err := d.DownloadWithProgress(job.pkg, job.destPath, nil)
				results <- downloadResult{pkg: job.pkg, err: err}
			}
		}()
	}

	for _, pkg := range packages {
		filename := pkg.Filename
		if filename == "" {
			filename = fmt.Sprintf("%s_%s_%s.deb", pkg.Name, pkg.Version, pkg.Architecture)
		}
		destPath := filepath.Join(destDir, filename)
		jobs <- downloadJob{pkg: pkg, destPath: destPath}
	}
	close(jobs)

	var errors []error
	for i := 0; i < len(packages); i++ {
		result := <-results
		if result.err != nil {
			errors = append(errors, fmt.Errorf("erreur pour le paquet %s: %v", result.pkg.Name, result.err))
		}
	}

	return errors
}

func (d *Downloader) GetFileSize(url string) (int64, error) {
	client := &http.Client{Timeout: d.Timeout}

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", d.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("statut HTTP %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}
