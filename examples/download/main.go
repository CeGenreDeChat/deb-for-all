package main

import (
	"fmt"
	"log"
	"os"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	fmt.Println("=== Exemple de téléchargement de paquets Debian ===")

	downloadDir := "./downloads"
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Fatal("Erreur lors de la création du répertoire:", err)
	}

	fmt.Println("\n1. Téléchargement avec downloader avancé")
	downloader := debian.NewDownloader()
	downloader.RetryAttempts = 2 // Réduire pour les tests

	testPkg := &debian.Package{
		Name:         "hello",
		Version:      "2.10-2",
		Architecture: "amd64",
		DownloadURL:  "http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2_amd64.deb",
		Filename:     "hello_2.10-2_amd64.deb",
	}

	fmt.Printf("Tentative de téléchargement de %s...\n", testPkg.Name)
	err := downloader.DownloadWithProgress(testPkg, fmt.Sprintf("%s/%s", downloadDir, testPkg.Filename), func(downloaded, total int64) {
		if total > 0 {
			percentage := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgrès: %.1f%% (%d/%d bytes)", percentage, downloaded, total)
		}
	})

	if err != nil {
		fmt.Printf("\nErreur: %v\n", err)
	} else {
		fmt.Printf("\nTéléchargement terminé avec succès!\n")
	}

	fmt.Println("\n2. Informations sur un fichier distant")
	size, sizeErr := downloader.GetFileSize("http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2_amd64.deb")
	if sizeErr != nil {
		fmt.Printf("Erreur lors de la récupération de la taille: %v\n", sizeErr)
	} else {
		fmt.Printf("Taille du fichier: %d bytes\n", size)
	}

	fmt.Println("\n3. Utilisation d'un dépôt Debian")
	repo := debian.NewRepository("debian-official", "http://deb.debian.org/debian", "Dépôt officiel Debian")

	available, availErr := repo.CheckPackageAvailability("hello", "2.10-2", "amd64")
	if availErr != nil {
		fmt.Printf("Erreur lors de la vérification: %v\n", availErr)
	} else {
		fmt.Printf("Paquet hello disponible: %t\n", available)
	}
	fmt.Println("\n=== Exemple terminé ===")
	fmt.Printf("Vérifiez le répertoire %s pour les fichiers téléchargés.\n", downloadDir)
}
