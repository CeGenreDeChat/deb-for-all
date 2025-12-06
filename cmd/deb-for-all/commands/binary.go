package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func DownloadBinaryPackage(packageName, version, destDir string, silent bool, localizer *i18n.Localizer) error {
	if !silent {
		fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "command.download.start",
			TemplateData: map[string]any{
				"Package": packageName,
				"Version": version,
				"Dest":    destDir,
			},
		}))
	}

	var err error

	if packageName == "" {
		return fmt.Errorf("le nom du paquet est requis")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %v", err)
	}

	// Create repository to search for the package
	repo := debian.NewRepository(
		"download-repo",
		"http://deb.debian.org/debian",
		"Repository for package download",
		"bookworm",        // default suite
		[]string{"main"},  // default component
		[]string{"amd64"}, // default architecture
	)

	if !silent {
		fmt.Printf("Recherche du paquet %s", packageName)
		if version != "" {
			fmt.Printf(" version %s", version)
		}
		fmt.Println("...")
	}

	if _, err = repo.FetchPackages(); err != nil {
		return fmt.Errorf("erreur lors de la récupération des paquets: %v", err)
	}

	var pkgMetadata *debian.Package

	// Get package metadata
	if pkgMetadata, err = repo.GetPackageMetadata(packageName); err != nil {
		return fmt.Errorf("erreur lors de la récupération des métadonnées pour le paquet %s: %v", packageName, err)
	} else if pkgMetadata == nil {
		return fmt.Errorf("impossible de récupérer les métadonnées pour le paquet %s", packageName)
	}

	// Filter by version if specified
	if version != "" && pkgMetadata.Version != version {
		return fmt.Errorf("version %s non trouvée pour le paquet %s (version disponible: %s)",
			version, packageName, pkgMetadata.Version)
	}

	if !silent {
		fmt.Printf("Téléchargement du paquet %s version %s...\n", pkgMetadata.Name, pkgMetadata.Version)
		fmt.Printf("Architecture: %s\n", pkgMetadata.Architecture)
		fmt.Printf("Taille: %d bytes\n", pkgMetadata.Size)
	}

	filepath := filepath.Join(destDir, pkgMetadata.Name+"_"+pkgMetadata.Version+"_"+pkgMetadata.Architecture+".deb")

	// Create downloader
	downloader := debian.NewDownloader()

	// Download the package
	if silent {
		err = downloader.DownloadSilent(pkgMetadata, filepath)
	} else {
		fmt.Printf("Téléchargement vers %s...\n", filepath)
		err = downloader.DownloadWithProgress(pkgMetadata, filepath, func(downloaded, total int64) {
			if total > 0 {
				percentage := float64(downloaded) / float64(total) * 100
				fmt.Printf("\rTéléchargement: %.1f%% (%d/%d bytes)", percentage, downloaded, total)
			}
		})
	}

	if err != nil {
		return fmt.Errorf("erreur lors du téléchargement: %v", err)
	}

	if !silent {
		fmt.Printf("\n✓ Paquet %s téléchargé avec succès vers %s\n", pkgMetadata.Name, destDir)
	}

	return nil
}
