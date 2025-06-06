package main

import (
	"fmt"
	"log"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	fmt.Println("=== Test de la fonction FetchPackages avec décompression ===")

	repo := debian.NewRepository(
		"debian-main",
		"http://deb.debian.org/debian",
		"Dépôt principal Debian",
	)

	fmt.Printf("Récupération des paquets depuis: %s\n", repo.URL)
	fmt.Println("La fonction essaiera les formats: non-compressé, .gz, .xz")
	fmt.Println("Ceci peut prendre quelques secondes pour télécharger et décompresser...")

	packages, err := repo.FetchPackages()
	if err != nil {
		log.Fatalf("Erreur lors de la récupération des paquets: %v", err)
	}

	fmt.Printf("✓ %d paquets trouvés (décompression réussie!)\n\n", len(packages))

	fmt.Println("Premiers 20 paquets:")
	for i, pkg := range packages {
		if i >= 20 {
			break
		}
		fmt.Printf("  %d. %s\n", i+1, pkg)
	}

	if len(packages) > 20 {
		fmt.Printf("\n... et %d autres paquets\n", len(packages)-20)
	}

	fmt.Println("\n=== Recherche de paquets spécifiques ===")
	searchPackages := []string{"hello", "curl", "vim", "git", "python3"}

	for _, searchPkg := range searchPackages {
		found := false
		for _, pkg := range packages {
			if pkg == searchPkg {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("✓ %s trouvé\n", searchPkg)
		} else {
			fmt.Printf("✗ %s non trouvé\n", searchPkg)
		}
	}
}
