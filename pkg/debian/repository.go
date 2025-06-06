package debian

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ulikunitz/xz"
)

type Repository struct {
	Name        string
	URL         string
	Description string
}

func NewRepository(name, url, description string) *Repository {
	return &Repository{
		Name:        name,
		URL:         url,
		Description: description,
	}
}

func (r *Repository) FetchPackages() ([]string, error) {
	distributions := []string{"bookworm", "bullseye", "buster"}
	sections := []string{"main", "contrib", "non-free"}
	architecture := "amd64"
	extensions := []string{"", ".gz", ".xz"}

	var lastErr error

	for _, dist := range distributions {
		for _, section := range sections {
			for _, ext := range extensions {
				packagesURL := r.buildPackagesURLWithDist(dist, section, architecture) + ext

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

				if ext == "" {
					return r.downloadAndParsePackages(packagesURL)
				} else {
					return r.downloadAndParseCompressedPackages(packagesURL, ext)
				}
			}
		}
	}

	return nil, fmt.Errorf("impossible de récupérer les paquets depuis toutes les distributions testées: %v", lastErr)
}

func (r *Repository) SearchPackage(packageName string) (string, error) {
	return "", nil
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

	return pkg.Download(destDir)
}

func (r *Repository) DownloadPackageByURL(packageURL, destDir string) error {
	parts := strings.Split(packageURL, "/")
	filename := parts[len(parts)-1]

	pkg := &Package{
		Name:        strings.Split(filename, "_")[0],
		DownloadURL: packageURL,
		Filename:    filename,
	}

	return pkg.Download(destDir)
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

			return pkg.Download(destDir)
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

func (r *Repository) buildPackagesURL(section, architecture string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/stable/%s/binary-%s/Packages", baseURL, section, architecture)
}

func (r *Repository) buildPackagesURLWithDist(distribution, section, architecture string) string {
	baseURL := strings.TrimSuffix(r.URL, "/")
	return fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages", baseURL, distribution, section, architecture)
}

func (r *Repository) downloadAndParsePackages(packagesURL string) ([]string, error) {
	resp, err := http.Get(packagesURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération du fichier Packages: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("impossible de récupérer le fichier Packages (HTTP %d)", resp.StatusCode)
	}

	var packages []string
	scanner := bufio.NewScanner(resp.Body)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // Buffer 1MB

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Package:") {
			packageName := strings.TrimSpace(strings.TrimPrefix(line, "Package:"))
			if packageName != "" {
				packages = append(packages, packageName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du fichier Packages: %v", err)
	}

	return packages, nil
}

func (r *Repository) downloadAndParseCompressedPackages(packagesURL string, extension string) ([]string, error) {
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

	var packages []string
	scanner := bufio.NewScanner(reader)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // Buffer 1MB

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Package:") {
			packageName := strings.TrimSpace(strings.TrimPrefix(line, "Package:"))
			if packageName != "" {
				packages = append(packages, packageName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du fichier Packages décompressé: %v", err)
	}

	return packages, nil
}

type PackageInfo struct {
	Name         string
	Version      string
	Architecture string
	Section      string
	DownloadURL  string
	Size         int64
}
