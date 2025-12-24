package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func DownloadBinaryPackage(packageName, version, baseURL string, suites, components, architectures []string, destDir string, silent bool, keyrings []string, skipGPGVerify bool, localizer *i18n.Localizer) error {
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
		return fmt.Errorf("package name is required")
	}

	if err := os.MkdirAll(destDir, debian.DirPermission); err != nil {
		return fmt.Errorf("unable to create destination directory: %w", err)
	}

	if len(suites) == 0 {
		suites = []string{"bookworm"}
	}
	if len(components) == 0 {
		components = []string{"main"}
	}
	if len(architectures) == 0 {
		architectures = []string{"amd64"}
	}
	if baseURL == "" {
		baseURL = "http://deb.debian.org/debian"
	}

	repo := debian.NewRepository(
		"download-repo",
		baseURL,
		"Repository for package download",
		suites[0],
		components,
		architectures,
	)

	repo.SetKeyringPaths(keyrings)
	if skipGPGVerify {
		repo.DisableSignatureVerification()
	}

	if !silent {
		fmt.Printf("Recherche du paquet %s", packageName)
		if version != "" {
			fmt.Printf(" version %s", version)
		}
		fmt.Println("...")
	}

	if _, err = repo.FetchPackages(); err != nil {
		return fmt.Errorf("error retrieving packages: %w", err)
	}

	pkgMetadata, err := repo.GetPackageMetadataWithArch(packageName, version, architectures)
	if err != nil {
		return fmt.Errorf("error retrieving metadata for package %s: %w", packageName, err)
	}

	if !silent {
		fmt.Printf("Téléchargement du paquet %s version %s...\n", pkgMetadata.Name, pkgMetadata.Version)
		fmt.Printf("Architecture: %s\n", pkgMetadata.Architecture)
		fmt.Printf("Taille: %d bytes\n", pkgMetadata.Size)
	}

	destPath := filepath.Join(destDir, packageFilename(pkgMetadata))

	// Create downloader
	downloader := debian.NewDownloader()

	skip, err := downloader.ShouldSkipDownload(pkgMetadata, destPath)
	if err != nil {
		return fmt.Errorf("failed to check existing file for %s: %w", pkgMetadata.Name, err)
	}
	if skip {
		if !silent {
			fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "command.download.skip_existing",
				TemplateData: map[string]any{
					"Package": pkgMetadata.Name,
				},
			}))
		}
		return nil
	}

	// Download the package
	if silent {
		err = downloader.DownloadSilent(pkgMetadata, destPath)
	} else {
		fmt.Printf("Téléchargement vers %s...\n", destPath)
		err = downloader.DownloadWithProgress(pkgMetadata, destPath, func(downloaded, total int64) {
			if total > 0 {
				percentage := float64(downloaded) / float64(total) * 100
				fmt.Printf("\rTéléchargement: %.1f%% (%d/%d bytes)", percentage, downloaded, total)
			}
		})
	}

	if err != nil {
		return fmt.Errorf("error downloading: %w", err)
	}

	if !silent {
		fmt.Printf("\n✓ Paquet %s téléchargé avec succès vers %s\n", pkgMetadata.Name, destDir)
	}

	return nil
}
