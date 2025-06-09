package main

import (
	"fmt"
	"log"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	fmt.Println("=== Test du parsing complet des paquets Debian ===")

	// Créer un repository Debian
	repo := debian.NewRepository(
		"debian-test",
		"http://deb.debian.org/debian",
		"Debian Official Repository",
		"bookworm",
		[]string{"main"},
		[]string{"amd64"},
	)

	// Désactiver la vérification Release pour ce test
	repo.DisableReleaseVerification()

	// Récupérer quelques paquets pour tester
	fmt.Println("Récupération des métadonnées des paquets...")
	packages, err := repo.FetchPackages()
	if err != nil {
		log.Fatalf("Erreur lors de la récupération des paquets: %v", err)
	}

	fmt.Printf("✅ %d paquets trouvés\n\n", len(packages))

	// Afficher les métadonnées complètes de quelques paquets
	metadata := repo.GetAllPackageMetadata()

	fmt.Println("=== Métadonnées détaillées des premiers paquets ===")

	count := 0
	for _, pkg := range metadata {
		if count >= 3 { // Limiter à 3 paquets pour l'exemple
			break
		}

		// Afficher seulement les paquets avec des métadonnées intéressantes
		if pkg.Section != "" || len(pkg.Depends) > 0 || pkg.Homepage != "" {
			fmt.Printf("\n📦 Paquet: %s\n", pkg.Name)
			fmt.Printf("   Version: %s\n", pkg.Version)
			fmt.Printf("   Architecture: %s\n", pkg.Architecture)
			fmt.Printf("   Section: %s\n", pkg.Section)
			fmt.Printf("   Priorité: %s\n", pkg.Priority)
			fmt.Printf("   Maintainer: %s\n", pkg.Maintainer)

			if pkg.Homepage != "" {
				fmt.Printf("   Homepage: %s\n", pkg.Homepage)
			}

			if pkg.Essential != "" {
				fmt.Printf("   Essential: %s\n", pkg.Essential)
			}

			if len(pkg.Depends) > 0 {
				fmt.Printf("   Dépendances (%d): %v\n", len(pkg.Depends), pkg.Depends[:min(3, len(pkg.Depends))])
			}

			if len(pkg.Recommends) > 0 {
				fmt.Printf("   Recommandations (%d): %v\n", len(pkg.Recommends), pkg.Recommends[:min(2, len(pkg.Recommends))])
			}

			if pkg.InstalledSize != "" {
				fmt.Printf("   Taille installée: %s\n", pkg.InstalledSize)
			}

			if pkg.MultiArch != "" {
				fmt.Printf("   Multi-Arch: %s\n", pkg.MultiArch)
			}

			if pkg.Tag != "" {
				fmt.Printf("   Tags: %s\n", pkg.Tag)
			}

			if len(pkg.CustomFields) > 0 {
				fmt.Printf("   Champs personnalisés: %d\n", len(pkg.CustomFields))
				for key, value := range pkg.CustomFields {
					fmt.Printf("     %s: %s\n", key, value)
				}
			}

			fmt.Printf("   Description: %.100s...\n", pkg.Description)
			count++
		}
	}

	fmt.Println("\n✅ Test du parsing complet terminé avec succès !")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
