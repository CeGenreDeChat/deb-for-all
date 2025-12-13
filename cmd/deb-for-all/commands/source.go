package commands

import (
	"fmt"
	"os"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func DownloadSourcePackage(packageName, version, baseURL string, suites, components, architectures []string, destDir string, origOnly, silent bool, localizer *i18n.Localizer) error {
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

	if packageName == "" {
		return fmt.Errorf("le nom du paquet est requis")
	}

	if err := os.MkdirAll(destDir, debian.DirPermission); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %w", err)
	}

	if len(suites) == 0 {
		suites = []string{"bookworm"}
	}
	if len(components) == 0 {
		components = []string{"main"}
	}
	if len(architectures) == 0 {
		architectures = []string{"source"}
	}
	if baseURL == "" {
		baseURL = "http://deb.debian.org/debian"
	}

	repo := debian.NewRepository(
		"download-source-repo",
		baseURL,
		"Repository for source package download",
		suites[0],
		components,
		architectures,
	)

	// Signature verification for sources is disabled until CLI flags mirror the binary command.
	repo.DisableSignatureVerification()

	if !silent {
		fmt.Printf("Recherche du paquet source %s", packageName)
		if version != "" {
			fmt.Printf(" version %s", version)
		}
		fmt.Println("...")
	}

	if _, err := repo.FetchSources(); err != nil {
		return fmt.Errorf("erreur lors de la récupération des paquets source: %w", err)
	}

	sourcePackage, err := repo.GetSourcePackageMetadata(packageName, version)
	if err != nil {
		return fmt.Errorf("erreur lors de la récupération des métadonnées pour le paquet source %s: %w", packageName, err)
	}

	if version == "" {
		version = sourcePackage.Version
	}

	if !silent {
		fmt.Printf("Téléchargement du paquet source %s version %s...\n", sourcePackage.Name, sourcePackage.Version)
		fmt.Printf("Répertoire pool: %s\n", sourcePackage.Directory)
	}

	downloadFn := func(sp *debian.SourcePackage) error {
		downloader := debian.NewDownloader()
		if silent {
			return downloader.DownloadSourcePackageSilent(sp, destDir)
		}
		return downloader.DownloadSourcePackageWithProgress(sp, destDir, func(filename string, downloaded, total int64) {
			if total <= 0 {
				return
			}
			percentage := float64(downloaded) / float64(total) * 100
			fmt.Printf("\r%s: %.1f%% (%d/%d bytes)", filename, percentage, downloaded, total)
		})
	}

	var downloadErr error
	if origOnly {
		orig := sourcePackage.GetOrigTarball()
		if orig == nil {
			return fmt.Errorf("aucun tarball original trouvé pour %s", packageName)
		}

		if !silent {
			fmt.Println("Téléchargement du tarball original uniquement...")
		}

		single := *sourcePackage
		single.Files = []debian.SourceFile{*orig}
		downloadErr = downloadFn(&single)
	} else {
		downloadErr = downloadFn(sourcePackage)
	}

	if downloadErr != nil {
		return fmt.Errorf("erreur lors du téléchargement: %w", downloadErr)
	}

	if !silent {
		fmt.Printf("\n✓ Paquet source %s téléchargé avec succès vers %s\n", packageName, destDir)
	}

	return nil
}
