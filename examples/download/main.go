package main

import (
	"fmt"
	"log"
	"os"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	fmt.Println("=== Exemples de gestion des paquets Debian ===")

	// Test de la fonction FetchPackages
	fmt.Println("\n=== 1. Test de FetchPackages ===")
	testFetchPackages()

	fmt.Println("\n=== 2. Téléchargement de paquets ===")

	downloadDir := "./downloads"
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Fatal("Erreur lors de la création du répertoire:", err)
	}

	fmt.Println("\n1. Téléchargement avec downloader avancé")
	downloader := debian.NewDownloader()
	downloader.RetryAttempts = 2

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

	fmt.Println("\n4. Utilisation d'un dépôt Debian")
	repo := debian.NewRepository(
		"debian-official",
		"http://deb.debian.org/debian",
		"Dépôt officiel Debian",
		"bookworm",                              // Distribution
		[]string{"main", "contrib", "non-free"}, // Sections
		[]string{"amd64"},                       // Architectures
	)

	available, availErr := repo.CheckPackageAvailability("hello", "2.10-2", "amd64")
	if availErr != nil {
		fmt.Printf("Erreur lors de la vérification: %v\n", availErr)
	} else {
		fmt.Printf("Paquet hello disponible: %t\n", available)
	}

	fmt.Println("\n5. Test de téléchargement silencieux")
	fmt.Printf("Téléchargement silencieux de %s...", testPkg.Name)
	err = downloader.DownloadSilent(testPkg, fmt.Sprintf("%s/hello_silent.deb", downloadDir))
	if err != nil {
		fmt.Printf(" ❌ Erreur: %v\n", err)
	} else {
		fmt.Printf(" ✅ Succès (sans affichage de progression)\n")
	}

	fmt.Printf("Test méthode Package.DownloadSilent()...")
	err = testPkg.DownloadSilent(downloadDir)
	if err != nil {
		fmt.Printf(" ❌ Erreur: %v\n", err)
	} else {
		fmt.Printf(" ✅ Succès\n")
	}

	fmt.Println("\n3. Téléchargement de paquets sources")

	fmt.Println("\n=== 3.1 Paquet source hello ===")
	helloSource := debian.NewSourcePackage(
		"hello",
		"2.10-2",
		"Santiago Vila <sanvila@debian.org>",
		"example package based on GNU hello",
		"pool/main/h/hello",
	)

	helloSource.AddFile(
		"hello_2.10-2.dsc",
		"http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2.dsc",
		1335,
		"",
		"",
		"dsc",
	)

	helloSource.AddFile(
		"hello_2.10.orig.tar.gz",
		"http://deb.debian.org/debian/pool/main/h/hello/hello_2.10.orig.tar.gz",
		725946,
		"",
		"",
		"orig",
	)

	helloSource.AddFile(
		"hello_2.10-2.debian.tar.xz",
		"http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2.debian.tar.xz",
		6132,
		"",
		"",
		"debian",
	)

	sourceDestDir := fmt.Sprintf("%s/hello-source", downloadDir)
	fmt.Printf("Téléchargement du paquet source %s...\n", helloSource.String())

	err = downloader.DownloadSourcePackageWithProgress(helloSource, sourceDestDir, func(filename string, downloaded, total int64) {
		if total > 0 {
			progress := float64(downloaded) / float64(total) * 100
			fmt.Printf("\r%s: %.1f%% (%d/%d bytes)", filename, progress, downloaded, total)
		}
	})

	if err != nil {
		fmt.Printf("\nErreur lors du téléchargement: %v\n", err)
	} else {
		fmt.Println("\n✓ Téléchargement du paquet source hello terminé")
	}

	fmt.Println("\n=== 3.2 Téléchargement du tarball original uniquement ===")
	origDestDir := fmt.Sprintf("%s/hello-orig-only", downloadDir)
	fmt.Printf("Téléchargement du tarball original vers %s...\n", origDestDir)

	err = downloader.DownloadOrigTarball(helloSource, origDestDir)
	if err != nil {
		fmt.Printf("Erreur lors du téléchargement du tarball original: %v\n", err)
	} else {
		fmt.Println("✓ Téléchargement du tarball original terminé")
	}

	fmt.Println("\n=== 3.3 Téléchargement silencieux ===")
	silentDestDir := fmt.Sprintf("%s/hello-silent", downloadDir)
	fmt.Print("Téléchargement silencieux...")

	err = downloader.DownloadSourcePackageSilent(helloSource, silentDestDir)
	if err != nil {
		fmt.Printf(" ❌ Erreur: %v\n", err)
	} else {
		fmt.Println(" ✓ Téléchargement silencieux terminé")
	}

	fmt.Println("\n=== 3.4 Informations sur le paquet source ===")
	fmt.Printf("Nom: %s\n", helloSource.Name)
	fmt.Printf("Version: %s\n", helloSource.Version)
	fmt.Printf("Maintainer: %s\n", helloSource.Maintainer)
	fmt.Printf("Description: %s\n", helloSource.Description)
	fmt.Printf("Nombre de fichiers: %d\n", len(helloSource.Files))

	fmt.Println("\nFichiers du paquet source:")
	for _, file := range helloSource.Files {
		fmt.Printf("  - %s (%s, %d bytes)\n", file.Name, file.Type, file.Size)
	}

	if origFile := helloSource.GetOrigTarball(); origFile != nil {
		fmt.Printf("\nTarball original: %s\n", origFile.Name)
	}

	if debianFile := helloSource.GetDebianTarball(); debianFile != nil {
		fmt.Printf("Tarball Debian: %s\n", debianFile.Name)
	}

	if dscFile := helloSource.GetDSCFile(); dscFile != nil {
		fmt.Printf("Fichier DSC: %s\n", dscFile.Name)
	}

	fmt.Println("\n=== Résumé des téléchargements ===")
	fmt.Printf("Répertoire de téléchargement: %s\n", downloadDir)
	fmt.Println("Vérifiez le contenu du répertoire pour voir les fichiers téléchargés.")
}

func testFetchPackages() {
	fmt.Println("Création d'un dépôt Debian pour tester FetchPackages...")

	repo := debian.NewRepository(
		"debian-main",
		"http://deb.debian.org/debian",
		"Dépôt principal Debian",
		"bookworm",                              // Distribution
		[]string{"main", "contrib", "non-free"}, // Sections
		[]string{"amd64"},                       // Architectures
	)

	fmt.Printf("Récupération des paquets depuis: %s\n", repo.URL)
	fmt.Println("Ceci peut prendre quelques secondes...")

	// Récupérer la liste des paquets
	packages, err := repo.FetchPackages()
	if err != nil {
		fmt.Printf("❌ Erreur lors de la récupération des paquets: %v\n", err)
		return
	}

	fmt.Printf("✓ %d paquets trouvés\n", len(packages))

	if len(packages) > 0 {
		fmt.Println("Premiers 10 paquets:")
		for i, pkg := range packages {
			if i >= 10 {
				break
			}
			fmt.Printf("  %d. %s\n", i+1, pkg)
		}

		if len(packages) > 10 {
			fmt.Printf("... et %d autres paquets\n", len(packages)-10)
		}
	}

	// Rechercher quelques paquets spécifiques
	fmt.Println("\nRecherche de paquets spécifiques:")
	searchPackages := []string{"hello", "curl", "vim"}

	for _, searchPkg := range searchPackages {
		found := false
		for _, pkg := range packages {
			if pkg == searchPkg {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("  ✓ %s trouvé\n", searchPkg)
		} else {
			fmt.Printf("  ✗ %s non trouvé\n", searchPkg)
		}
	}
}
