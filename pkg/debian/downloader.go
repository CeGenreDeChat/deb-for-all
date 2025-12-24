package debian

import (
	"crypto/md5"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Download configuration constants.
const (
	defaultUserAgent     = "deb-for-all/1.0"
	defaultTimeout       = 30 * time.Second
	defaultRetryAttempts = 3
	defaultConcurrency   = 5
	retryDelay           = 2 * time.Second
	downloadBufferSize   = 32 * 1024 // 32KB buffer
)

// Downloader handles HTTP downloads with retry logic, progress tracking,
// and checksum verification for Debian packages.
type Downloader struct {
	UserAgent       string
	Timeout         time.Duration
	RetryAttempts   int
	VerifyChecksums bool
}

// NewDownloader creates a new Downloader with default settings.
func NewDownloader() *Downloader {
	return &Downloader{
		UserAgent:       defaultUserAgent,
		Timeout:         defaultTimeout,
		RetryAttempts:   defaultRetryAttempts,
		VerifyChecksums: true,
	}
}

// newHTTPClient creates a new HTTP client with the configured timeout.
func (d *Downloader) newHTTPClient() *http.Client {
	return &http.Client{Timeout: d.Timeout}
}

// doRequestWithRetry performs an HTTP request with retry logic.
// Returns the response and any error encountered.
func (d *Downloader) doRequestWithRetry(method, url string, silent bool) (*http.Response, error) {
	client := d.newHTTPClient()
	var lastErr error

	for attempt := 1; attempt <= d.RetryAttempts; attempt++ {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la création de la requête: %w", err)
		}
		req.Header.Set("User-Agent", d.UserAgent)

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("statut HTTP %d", resp.StatusCode)
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < d.RetryAttempts {
			if !silent {
				fmt.Printf("Tentative %d échouée, nouvelle tentative dans %v...\n", attempt, retryDelay)
			}
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("erreur lors du téléchargement après %d tentatives: %w", d.RetryAttempts, lastErr)
}

// getPackageFilename returns the filename for a package, generating one if not set.
func getPackageFilename(pkg *Package) string {
	if pkg.Filename != "" {
		return pkg.Filename
	}
	return fmt.Sprintf("%s_%s_%s.deb", pkg.Name, pkg.Version, pkg.Architecture)
}

// downloadToFile performs the actual download to a file with optional progress callback.
func (d *Downloader) downloadToFile(url, destPath string, progressCallback func(downloaded, total int64)) error {
	if err := os.MkdirAll(filepath.Dir(destPath), DirPermission); err != nil {
		return fmt.Errorf("impossible de créer le répertoire parent: %w", err)
	}

	resp, err := d.doRequestWithRetry(http.MethodGet, url, progressCallback == nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier de destination: %w", err)
	}
	defer destFile.Close()

	if progressCallback == nil {
		_, err = io.Copy(destFile, resp.Body)
		if err != nil {
			return fmt.Errorf("erreur lors de la copie du fichier: %w", err)
		}
		return nil
	}

	return d.copyWithProgress(resp.Body, destFile, resp.ContentLength, progressCallback)
}

// copyWithProgress copies data from src to dst while reporting progress.
func (d *Downloader) copyWithProgress(src io.Reader, dst io.Writer, totalSize int64, callback func(downloaded, total int64)) error {
	buffer := make([]byte, downloadBufferSize)
	var downloaded int64

	for {
		n, err := src.Read(buffer)
		if n > 0 {
			if _, writeErr := dst.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("erreur lors de l'écriture: %w", writeErr)
			}
			downloaded += int64(n)
			callback(downloaded, totalSize)
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("erreur lors de la lecture: %w", err)
		}
	}
}

// DownloadWithProgress downloads a package to the specified path with progress reporting.
func (d *Downloader) DownloadWithProgress(pkg *Package, destPath string, progressCallback func(downloaded, total int64)) error {
	if pkg.DownloadURL == "" {
		return fmt.Errorf("aucune URL de téléchargement spécifiée pour le paquet %s", pkg.Name)
	}

	if err := d.downloadToFile(pkg.DownloadURL, destPath, progressCallback); err != nil {
		return err
	}

	fmt.Printf("Paquet %s téléchargé avec succès vers %s\n", pkg.Name, destPath)
	return nil
}

// DownloadSilent downloads a package without any output.
func (d *Downloader) DownloadSilent(pkg *Package, destPath string) error {
	if pkg.DownloadURL == "" {
		return fmt.Errorf("aucune URL de téléchargement spécifiée pour le paquet %s", pkg.Name)
	}
	return d.downloadToFile(pkg.DownloadURL, destPath, nil)
}

// DownloadWithChecksum downloads a package and verifies its checksum.
func (d *Downloader) DownloadWithChecksum(pkg *Package, destPath, checksum, checksumType string) error {
	if err := d.DownloadWithProgress(pkg, destPath, nil); err != nil {
		return err
	}

	if d.VerifyChecksums && checksum != "" {
		return d.verifyChecksum(destPath, checksum, checksumType)
	}
	return nil
}

// verifyChecksum verifies a file's checksum against the expected value.
func (d *Downloader) verifyChecksum(filePath, expectedChecksum, checksumType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir le fichier pour vérification: %w", err)
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

	if _, err = io.Copy(hasher, file); err != nil {
		return fmt.Errorf("erreur lors du calcul de la somme de contrôle: %w", err)
	}

	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("la somme de contrôle ne correspond pas. Attendue: %s, Actuelle: %s", expectedChecksum, actualChecksum)
	}

	fmt.Printf("Somme de contrôle %s vérifiée avec succès\n", checksumType)
	return nil
}

// ShouldSkipDownload checks if destPath already contains the expected file for the given package.
// It returns true when the file exists and its checksum matches the package metadata.
func (d *Downloader) ShouldSkipDownload(pkg *Package, destPath string) (bool, error) {
	info, err := os.Stat(destPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("unable to stat existing file %s: %w", destPath, err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("existing path %s is not a regular file", destPath)
	}

	expectedChecksum := strings.ToLower(pkg.SHA256)
	checksumType := "sha256"
	if expectedChecksum == "" {
		expectedChecksum = strings.ToLower(pkg.MD5sum)
		checksumType = "md5"
	}

	if expectedChecksum == "" {
		return false, nil
	}

	if err := d.verifyChecksum(destPath, expectedChecksum, checksumType); err != nil {
		return false, nil
	}

	return true, nil
}

// downloadJob represents a download task for concurrent processing.
type downloadJob struct {
	pkg      *Package
	destPath string
}

// downloadResult represents the result of a download task.
type downloadResult struct {
	pkg *Package
	err error
}

// DownloadMultiple downloads multiple packages concurrently.
// maxConcurrent specifies the number of parallel downloads (defaults to 5).
func (d *Downloader) DownloadMultiple(packages []*Package, destDir string, maxConcurrent int) []error {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultConcurrency
	}

	jobs := make(chan downloadJob, len(packages))
	results := make(chan downloadResult, len(packages))

	// Start worker goroutines
	var wg sync.WaitGroup
	for w := 0; w < maxConcurrent; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				err := d.DownloadWithProgress(job.pkg, job.destPath, nil)
				results <- downloadResult{pkg: job.pkg, err: err}
			}
		}()
	}

	// Queue download jobs
	for _, pkg := range packages {
		destPath := filepath.Join(destDir, getPackageFilename(pkg))
		jobs <- downloadJob{pkg: pkg, destPath: destPath}
	}
	close(jobs)

	// Wait for all workers to complete in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var errors []error
	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Errorf("erreur pour le paquet %s: %w", result.pkg.Name, result.err))
		}
	}

	return errors
}

// DownloadSourcePackage downloads all files of a source package.
func (d *Downloader) DownloadSourcePackage(sourcePkg *SourcePackage, destDir string) error {
	return sourcePkg.downloadFiles(destDir, true, nil)
}

// DownloadSourcePackageSilent downloads all files of a source package without output.
func (d *Downloader) DownloadSourcePackageSilent(sourcePkg *SourcePackage, destDir string) error {
	return sourcePkg.downloadFiles(destDir, false, nil)
}

// DownloadSourcePackageWithProgress downloads a source package with progress reporting.
func (d *Downloader) DownloadSourcePackageWithProgress(sourcePkg *SourcePackage, destDir string, progressCallback func(filename string, downloaded, total int64)) error {
	return sourcePkg.downloadFiles(destDir, true, progressCallback)
}

// DownloadSourceFile downloads a single source file with checksum verification.
func (d *Downloader) DownloadSourceFile(sourceFile *SourceFile, destDir string) error {
	if sourceFile.URL == "" {
		return fmt.Errorf("aucune URL spécifiée pour le fichier %s", sourceFile.Name)
	}

	destPath := filepath.Join(destDir, sourceFile.Name)

	if err := d.downloadToFile(sourceFile.URL, destPath, nil); err != nil {
		return err
	}

	fmt.Printf("Fichier %s téléchargé avec succès\n", sourceFile.Name)

	if d.VerifyChecksums {
		if sourceFile.SHA256Sum != "" {
			return d.verifyChecksum(destPath, sourceFile.SHA256Sum, "sha256")
		} else if sourceFile.MD5Sum != "" {
			return d.verifyChecksum(destPath, sourceFile.MD5Sum, "md5")
		}
	}

	return nil
}

// DownloadOrigTarball downloads only the original tarball from a source package.
func (d *Downloader) DownloadOrigTarball(sourcePkg *SourcePackage, destDir string) error {
	origFile := sourcePkg.GetOrigTarball()
	if origFile == nil {
		return fmt.Errorf("aucun fichier tarball original trouvé pour le paquet source %s", sourcePkg.Name)
	}
	return d.DownloadSourceFile(origFile, destDir)
}

// GetFileSize returns the Content-Length of a URL via HEAD request.
func (d *Downloader) GetFileSize(url string) (int64, error) {
	client := d.newHTTPClient()

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, fmt.Errorf("erreur lors de la création de la requête: %w", err)
	}
	req.Header.Set("User-Agent", d.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("erreur lors de la requête HEAD: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("statut HTTP %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

// DownloadToDir downloads a package to a directory with automatic filename generation.
func (d *Downloader) DownloadToDir(pkg *Package, destDir string) error {
	destPath := filepath.Join(destDir, getPackageFilename(pkg))
	return d.DownloadWithProgress(pkg, destPath, nil)
}

// DownloadToDirSilent downloads a package to a directory silently with automatic filename generation.
func (d *Downloader) DownloadToDirSilent(pkg *Package, destDir string) error {
	destPath := filepath.Join(destDir, getPackageFilename(pkg))
	return d.DownloadSilent(pkg, destPath)
}
