package main

import (
	"fmt"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	pkg := debian.Package{
		Name:         "example-package",
		Version:      "1.0.0",
		Architecture: "amd64",
		Maintainer:   "Your Name <youremail@example.com>",
		Description:  "This is an example Debian package.",
		DownloadURL:  "https://example.com/packages/example-package_1.0.0_amd64.deb",
		Filename:     "example-package_1.0.0_amd64.deb",
		Size:         1024000, // 1MB
	}

	fmt.Printf("Package Name: %s\n", pkg.Name)
	fmt.Printf("Version: %s\n", pkg.Version)
	fmt.Printf("Architecture: %s\n", pkg.Architecture)
	fmt.Printf("Maintainer: %s\n", pkg.Maintainer)
	fmt.Printf("Description: %s\n", pkg.Description)
	fmt.Printf("Download URL: %s\n", pkg.DownloadURL)
	fmt.Printf("Size: %d bytes\n", pkg.Size)

	fmt.Println("\n=== Exemple de téléchargement ===")

	downloader := debian.NewDownloader()

	progressCallback := func(downloaded, total int64) {
		if total > 0 {
			percentage := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgrès: %.1f%% (%d/%d bytes)", percentage, downloaded, total)
		}
	}

	// Note: This will fail because the URL is fake, but demonstrates the API
	err := downloader.DownloadWithProgress(&pkg, "./downloads/example-package.deb", progressCallback)
	if err != nil {
		fmt.Printf("\nErreur de téléchargement (attendue avec URL fictive): %v\n", err)
	}

	fmt.Println("\n=== Exemple avec dépôt ===")

	repo := debian.NewRepository("debian-main", "http://deb.debian.org/debian", "Dépôt principal Debian")
	fmt.Printf("Repository: %s (%s)\n", repo.Name, repo.URL)

	available, err := repo.CheckPackageAvailability("curl", "7.68.0-1", "amd64")
	if err != nil {
		fmt.Printf("Erreur lors de la vérification de disponibilité: %v\n", err)
	} else {
		fmt.Printf("Package curl disponible: %v\n", available)
	}

	fmt.Println("\n=== Informations de téléchargement ===")
	info, err := pkg.GetDownloadInfo()
	if err != nil {
		fmt.Printf("Erreur lors de la récupération des infos: %v\n", err)
	} else {
		fmt.Printf("URL: %s\n", info.URL)
		fmt.Printf("Taille: %d bytes\n", info.ContentLength)
		fmt.Printf("Type de contenu: %s\n", info.ContentType)
	}

	fmt.Println("\nExemple terminé avec succès.")
}
