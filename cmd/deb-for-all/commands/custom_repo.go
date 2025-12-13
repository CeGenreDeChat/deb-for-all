package commands

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type xmlPackageList struct {
	Packages []xmlPackageEntry `xml:"package"`
}

type xmlPackageEntry struct {
	Name    string `xml:",chardata"`
	Version string `xml:"version,attr"`
}

// BuildCustomRepository builds a custom repository subset from an XML package list,
// resolves dependencies (with optional exclusions), and downloads the resulting packages.
func BuildCustomRepository(baseURL, suites, components, architectures, destDir, packagesXML, excludeDeps string, keyrings []string, skipGPGVerify, verbose bool, localizer *i18n.Localizer) error {
	if packagesXML == "" {
		return fmt.Errorf("le fichier XML des paquets est requis")
	}

	packageSpecs, err := loadPackageSpecs(packagesXML)
	if err != nil {
		return err
	}

	excludeSet := parseExcludeDeps(excludeDeps)

	suiteList := splitAndTrim(suites)
	componentList := splitAndTrim(components)
	archList := splitAndTrim(architectures)

	if len(suiteList) == 0 {
		return fmt.Errorf("au moins une suite est requise")
	}
	if len(componentList) == 0 {
		return fmt.Errorf("au moins un composant est requis")
	}
	if len(archList) == 0 {
		return fmt.Errorf("au moins une architecture est requise")
	}

	if err := os.MkdirAll(destDir, debian.DirPermission); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %w", err)
	}

	for _, suite := range suiteList {
		repo := debian.NewRepository("custom-repo"+suite, baseURL, "custom repo", suite, componentList, archList)
		repo.SetKeyringPaths(keyrings)
		if skipGPGVerify {
			repo.DisableSignatureVerification()
		}

		if verbose {
			fmt.Printf("Suite %s: récupération des métadonnées...\n", suite)
		}

		if _, err := repo.FetchPackages(); err != nil {
			return fmt.Errorf("erreur lors de la récupération des paquets pour %s: %w", suite, err)
		}

		resolved, err := repo.ResolveDependencies(packageSpecs, excludeSet)
		if err != nil {
			return fmt.Errorf("erreur lors de la résolution des dépendances pour %s: %w", suite, err)
		}

		if verbose {
			fmt.Printf("Suite %s: %d paquets à télécharger\n", suite, len(resolved))
		}

		downloader := debian.NewDownloader()
		poolDir := filepath.Join(destDir, "pool", suite)

		for _, pkg := range resolved {
			targetDir := filepath.Join(poolDir, pkg.Section)
			if targetDir == poolDir || pkg.Section == "" {
				targetDir = filepath.Join(poolDir, "main")
			}
			if err := downloader.DownloadToDir(&pkg, targetDir); err != nil {
				return fmt.Errorf("erreur lors du téléchargement de %s: %w", pkg.Name, err)
			}
		}
	}

	return nil
}

func loadPackageSpecs(path string) ([]debian.PackageSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("impossible de lire le fichier XML: %w", err)
	}

	var list xmlPackageList
	if err := xml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("format XML invalide: %w", err)
	}

	specs := make([]debian.PackageSpec, 0, len(list.Packages))
	for _, entry := range list.Packages {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		specs = append(specs, debian.PackageSpec{Name: name, Version: strings.TrimSpace(entry.Version)})
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("aucun paquet valide trouvé dans le XML")
	}

	return specs, nil
}

func parseExcludeDeps(value string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range strings.Split(strings.TrimSpace(value), ",") {
		trimmed := strings.ToLower(strings.TrimSpace(item))
		if trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}
