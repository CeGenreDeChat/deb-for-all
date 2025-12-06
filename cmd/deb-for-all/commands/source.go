package commands

import (
	"fmt"
	"os"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func DownloadSourcePackage(packageName, version, destDir string, origOnly, silent bool, localizer *i18n.Localizer) error {
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

	downloader := debian.NewDownloader()

	sourcePackage := createExampleSourcePackage(packageName, version)

	if !silent {
		fmt.Printf("Téléchargement du paquet source %s (%s)...\n", packageName, version)
	}

	var err error
	if origOnly {
		if !silent {
			fmt.Println("Téléchargement du tarball original uniquement...")
		}
		err = downloader.DownloadOrigTarball(sourcePackage, destDir)
	} else {
		if silent {
			err = downloader.DownloadSourcePackageSilent(sourcePackage, destDir)
		} else {
			err = downloader.DownloadSourcePackageWithProgress(sourcePackage, destDir, func(filename string, downloaded, total int64) {
				if total > 0 {
					percentage := float64(downloaded) / float64(total) * 100
					fmt.Printf("\r%s: %.1f%% (%d/%d bytes)", filename, percentage, downloaded, total)
				}
			})
		}
	}

	if err != nil {
		return fmt.Errorf("erreur lors du téléchargement: %w", err)
	}

	if !silent {
		fmt.Printf("\n✓ Paquet source %s téléchargé avec succès vers %s\n", packageName, destDir)
	}

	return nil
}

func createExampleSourcePackage(packageName, version string) *debian.SourcePackage {
	sourcePackage := debian.NewSourcePackage(
		packageName,
		version,
		"Example Maintainer <maintainer@example.com>",
		fmt.Sprintf("Source package for %s", packageName),
		fmt.Sprintf("pool/main/%s/%s", string(packageName[0]), packageName),
	)

	baseURL := "http://deb.debian.org/debian/pool/main"

	switch packageName {
	case "hello":
		if version == "" {
			version = "2.10-2"
		}
		sourcePackage.AddFile(
			fmt.Sprintf("hello_%s.dsc", version),
			fmt.Sprintf("%s/h/hello/hello_%s.dsc", baseURL, version),
			1950, "", "", "dsc",
		)
		sourcePackage.AddFile(
			"hello_2.10.orig.tar.gz",
			fmt.Sprintf("%s/h/hello/hello_2.10.orig.tar.gz", baseURL),
			725946, "", "", "orig",
		)
		sourcePackage.AddFile(
			fmt.Sprintf("hello_%s.debian.tar.xz", version),
			fmt.Sprintf("%s/h/hello/hello_%s.debian.tar.xz", baseURL, version),
			7124, "", "", "debian",
		)

	case "curl":
		if version == "" {
			version = "7.74.0-1.3+deb11u7"
		}
		sourcePackage.AddFile(
			fmt.Sprintf("curl_%s.dsc", version),
			fmt.Sprintf("%s/c/curl/curl_%s.dsc", baseURL, version),
			2356, "", "", "dsc",
		)
		sourcePackage.AddFile(
			"curl_7.74.0.orig.tar.gz",
			fmt.Sprintf("%s/c/curl/curl_7.74.0.orig.tar.gz", baseURL),
			4194863, "", "", "orig",
		)
		sourcePackage.AddFile(
			fmt.Sprintf("curl_%s.debian.tar.xz", version),
			fmt.Sprintf("%s/c/curl/curl_%s.debian.tar.xz", baseURL, version),
			35684, "", "", "debian",
		)

	default:
		if version == "" {
			version = "1.0-1"
		}
		sourcePackage.AddFile(
			fmt.Sprintf("%s_%s.dsc", packageName, version),
			fmt.Sprintf("%s/%s/%s/%s_%s.dsc", baseURL, string(packageName[0]), packageName, packageName, version),
			1000, "", "", "dsc",
		)
		sourcePackage.AddFile(
			fmt.Sprintf("%s_1.0.orig.tar.gz", packageName),
			fmt.Sprintf("%s/%s/%s/%s_1.0.orig.tar.gz", baseURL, string(packageName[0]), packageName, packageName),
			10000, "", "", "orig",
		)
		sourcePackage.AddFile(
			fmt.Sprintf("%s_%s.debian.tar.xz", packageName, version),
			fmt.Sprintf("%s/%s/%s/%s_%s.debian.tar.xz", baseURL, string(packageName[0]), packageName, packageName, version),
			5000, "", "", "debian",
		)
	}
	return sourcePackage
}
