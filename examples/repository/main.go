package main

import (
	"fmt"
	"log"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	fmt.Println("=== Test de la fonction FetchPackages - Collecte TOUS les paquets ===")
	repo := debian.NewRepository(
		"debian-main",
		"http://deb.debian.org/debian",
		"Dépôt principal Debian",
		"bookworm",
		[]string{"main"},
		[]string{"amd64"},
	)

	fmt.Printf("Récupération des paquets depuis: %s\n", repo.URL)
	fmt.Printf("Distribution: %s\n", repo.Distribution)
	fmt.Printf("Sections: %v\n", repo.Sections)
	fmt.Printf("Architectures: %v\n", repo.Architectures)
	fmt.Println("⚠️ ATTENTION: Cette fonction va maintenant télécharger TOUS les fichiers Packages")
	fmt.Println("de toutes les sections et architectures (peut prendre plusieurs minutes)...")
	fmt.Println("Ceci peut prendre quelques secondes pour télécharger et décompresser...")

	packages, err := repo.FetchPackages()
	if err != nil {
		log.Fatalf("Erreur lors de la récupération des paquets: %v", err)
	}

	fmt.Printf("✓ %d paquets UNIQUES trouvés depuis TOUTES les sections!\n\n", len(packages))

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

	fmt.Println("\n=== Test avec configuration personnalisée ===")

	customRepo := debian.NewRepository(
		"debian-bullseye-arm64",
		"http://deb.debian.org/debian",
		"Dépôt Debian Bullseye pour ARM64",
		"bullseye",
		[]string{"main"},
		[]string{"arm64"},
	)

	fmt.Printf("Configuration personnalisée:\n")
	fmt.Printf("  Distribution: %s\n", customRepo.Distribution)
	fmt.Printf("  Sections: %v\n", customRepo.Sections)
	fmt.Printf("  Architectures: %v\n", customRepo.Architectures)

	fmt.Println("Test de récupération avec configuration personnalisée...")
	customPackages, err := customRepo.FetchPackages()
	if err != nil {
		fmt.Printf("✗ Erreur avec configuration personnalisée: %v\n", err)
	} else {
		fmt.Printf("✓ %d paquets trouvés avec configuration personnalisée\n", len(customPackages))
	}

	fmt.Println("\n=== Test de modification dynamique ===")

	dynamicRepo := debian.NewRepository(
		"debian-dynamic",
		"http://deb.debian.org/debian",
		"Dépôt Debian avec configuration dynamique",
		"bookworm",
		[]string{"main", "contrib", "non-free"},
		[]string{"amd64"},
	)

	fmt.Printf("Configuration initiale: %s, %v, %v\n",
		dynamicRepo.Distribution, dynamicRepo.Sections, dynamicRepo.Architectures)

	dynamicRepo.SetDistribution("bullseye")
	dynamicRepo.SetSections([]string{"main"})
	dynamicRepo.AddArchitecture("i386")

	fmt.Printf("Configuration modifiée: %s, %v, %v\n",
		dynamicRepo.Distribution, dynamicRepo.Sections, dynamicRepo.Architectures)
	fmt.Println("Test de récupération avec configuration dynamique modifiée...")
	dynamicPackages, err := dynamicRepo.FetchPackages()
	if err != nil {
		fmt.Printf("✗ Erreur avec configuration dynamique: %v\n", err)
	} else {
		fmt.Printf("✓ %d paquets trouvés avec configuration dynamique\n", len(dynamicPackages))
	}

	fmt.Println("\n=== Test de la fonction SearchPackage ===")

	searchTerms := []string{"vim", "nano", "curl", "git", "nginx", "apache2"}

	for _, term := range searchTerms {
		fmt.Printf("🔍 Recherche de '%s'...\n", term)

		matches, err := repo.SearchPackage(term)
		if err != nil {
			fmt.Printf("❌ Erreur lors de la recherche de '%s': %v\n", term, err)
			continue
		}

		fmt.Printf("✅ Trouvé %d paquet(s) correspondant(s):\n", len(matches))

		maxDisplay := 5
		if len(matches) > maxDisplay {
			for i, pkg := range matches[:maxDisplay] {
				fmt.Printf("  %d. %s\n", i+1, pkg)
			}
			fmt.Printf("  ... et %d autres\n", len(matches)-maxDisplay)
		} else {
			for i, pkg := range matches {
				fmt.Printf("  %d. %s\n", i+1, pkg)
			}
		}
		fmt.Println()
	}

	fmt.Println("🔍 Test avec un paquet inexistant...")
	_, err = repo.SearchPackage("paquet-inexistant-xyz123")
	if err != nil {
		fmt.Printf("✅ Comportement attendu: %v\n", err)
	} else {
		fmt.Println("❌ Erreur: le paquet inexistant a été trouvé!")
	}
}
