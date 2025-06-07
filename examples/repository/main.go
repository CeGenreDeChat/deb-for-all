package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	fmt.Println("=== EXEMPLE COMPLET: Gestion des dépôts Debian avec vérification Release ===")

	// ========================================
	// PARTIE 1: Tests basiques sans vérification
	// ========================================
	fmt.Println("\n🔧 PARTIE 1: Test de la fonction FetchPackages - Collecte TOUS les paquets")
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
	fmt.Printf("Vérification Release activée: %t\n", repo.IsReleaseVerificationEnabled())
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

	// ========================================
	// PARTIE 2: Vérification Release
	// ========================================
	fmt.Println("\n🔒 PARTIE 2: Test de la vérification des fichiers Release")

	fmt.Println("\n--- Activation de la vérification Release ---")
	repo.EnableReleaseVerification()
	fmt.Printf("Vérification Release activée: %t\n", repo.IsReleaseVerificationEnabled())

	fmt.Println("Récupération du fichier Release...")
	err = repo.FetchReleaseFile()
	if err != nil {
		log.Fatalf("Erreur lors de la récupération du fichier Release: %v", err)
	}

	releaseInfo := repo.GetReleaseInfo()
	if releaseInfo != nil {
		fmt.Println("✓ Fichier Release récupéré avec succès!")
		fmt.Printf("  Origin: %s\n", releaseInfo.Origin)
		fmt.Printf("  Label: %s\n", releaseInfo.Label)
		fmt.Printf("  Suite: %s\n", releaseInfo.Suite)
		fmt.Printf("  Version: %s\n", releaseInfo.Version)
		fmt.Printf("  Codename: %s\n", releaseInfo.Codename)
		fmt.Printf("  Date: %s\n", releaseInfo.Date)
		fmt.Printf("  Description: %s\n", releaseInfo.Description)
		fmt.Printf("  Architectures: %v\n", releaseInfo.Architectures)
		fmt.Printf("  Components: %v\n", releaseInfo.Components)
		fmt.Printf("  Checksums MD5: %d\n", len(releaseInfo.MD5Sum))
		fmt.Printf("  Checksums SHA1: %d\n", len(releaseInfo.SHA1))
		fmt.Printf("  Checksums SHA256: %d\n", len(releaseInfo.SHA256))

		// Afficher quelques exemples de checksums
		if len(releaseInfo.SHA256) > 0 {
			fmt.Println("\n  Exemples de checksums SHA256:")
			maxDisplay := 5
			for i, checksum := range releaseInfo.SHA256 {
				if i >= maxDisplay {
					fmt.Printf("    ... et %d autres\n", len(releaseInfo.SHA256)-maxDisplay)
					break
				}
				fmt.Printf("    %s %d %s\n", checksum.Hash[:16]+"...", checksum.Size, checksum.Filename)
			}
		}
	}

	// ========================================
	// PARTIE 3: Tests de vérification de checksums spécifiques
	// ========================================
	fmt.Println("\n🔍 PARTIE 3: Vérification détaillée des checksums")

	if releaseInfo != nil {
		// Recherche des checksums pour différents formats de Packages
		fmt.Println("\n--- Vérification des checksums pour différents formats ---")
		checksumFiles := []string{
			"main/binary-amd64/Packages",
			"main/binary-amd64/Packages.gz",
			"main/binary-amd64/Packages.xz",
		}

		for _, filename := range checksumFiles {
			var foundSHA256, foundMD5 bool

			for _, checksum := range releaseInfo.SHA256 {
				if checksum.Filename == filename {
					foundSHA256 = true
					fmt.Printf("✓ SHA256 %s: %s (taille: %d)\n", filename, checksum.Hash[:32]+"...", checksum.Size)
					break
				}
			}

			for _, checksum := range releaseInfo.MD5Sum {
				if checksum.Filename == filename {
					foundMD5 = true
					fmt.Printf("✓ MD5 %s: %s (taille: %d)\n", filename, checksum.Hash[:16]+"...", checksum.Size)
					break
				}
			}

			if !foundSHA256 && !foundMD5 {
				fmt.Printf("✗ Aucun checksum trouvé pour %s\n", filename)
			}
		}
	}

	fmt.Println("\n--- Test de récupération avec vérification Release ---")
	// Créer un nouveau repository pour un test propre
	verifiedRepo := debian.NewRepository(
		"debian-verified",
		"http://deb.debian.org/debian",
		"Dépôt Debian avec vérification",
		"bookworm",
		[]string{"main"},
		[]string{"amd64"},
	)

	verifiedRepo.EnableReleaseVerification()
	fmt.Println("Récupération des paquets avec vérification Release activée...")

	verifiedPackages, err := verifiedRepo.FetchPackages()
	if err != nil {
		log.Printf("Erreur lors de la récupération avec vérification: %v", err)
	} else {
		fmt.Printf("✓ %d paquets récupérés avec vérification Release réussie!\n", len(verifiedPackages))
	}

	// ========================================
	// PARTIE 4: Tests avec différentes distributions
	// ========================================
	fmt.Println("\n🌍 PARTIE 4: Test avec différentes distributions")
	distributions := []string{"bullseye", "bookworm", "sid"}

	for _, dist := range distributions {
		fmt.Printf("\nTest avec distribution: %s\n", dist)

		testRepo := debian.NewRepository(
			fmt.Sprintf("debian-%s", dist),
			"http://deb.debian.org/debian",
			fmt.Sprintf("Dépôt Debian %s", dist),
			dist,
			[]string{"main"},
			[]string{"amd64"},
		)

		testRepo.EnableReleaseVerification()

		err := testRepo.FetchReleaseFile()
		if err != nil {
			fmt.Printf("  ❌ Erreur avec %s: %v\n", dist, err)
			continue
		}

		releaseInfo := testRepo.GetReleaseInfo()
		if releaseInfo != nil {
			fmt.Printf("  ✓ Release %s: %s (%s)\n", dist, releaseInfo.Codename, releaseInfo.Version)
		}
	}

	// ========================================
	// PARTIE 5: Test Ubuntu
	// ========================================
	fmt.Println("\n🐧 PARTIE 5: Test avec Ubuntu")
	ubuntuRepo := debian.NewRepository(
		"ubuntu-main",
		"http://archive.ubuntu.com/ubuntu",
		"Ubuntu main repository",
		"jammy", // Ubuntu 22.04 LTS
		[]string{"main"},
		[]string{"amd64"},
	)

	fmt.Println("Test Ubuntu sans vérification Release...")
	ubuntuPackages, err := ubuntuRepo.FetchPackages()
	if err != nil {
		fmt.Printf("❌ Erreur Ubuntu sans vérification: %v\n", err)
	} else {
		fmt.Printf("✓ Ubuntu sans vérification: %d paquets\n", len(ubuntuPackages))
	}

	fmt.Println("Activation de la vérification Release pour Ubuntu...")
	ubuntuRepo.EnableReleaseVerification()

	err = ubuntuRepo.FetchReleaseFile()
	if err != nil {
		fmt.Printf("❌ Erreur récupération Release Ubuntu: %v\n", err)
	} else {
		releaseInfo := ubuntuRepo.GetReleaseInfo()
		fmt.Printf("✓ Ubuntu Release: %s %s (%d checksums SHA256)\n",
			releaseInfo.Origin, releaseInfo.Codename, len(releaseInfo.SHA256))

		fmt.Println("Test récupération avec vérification Ubuntu...")
		verifiedUbuntuPackages, err := ubuntuRepo.FetchPackages()
		if err != nil {
			fmt.Printf("❌ Erreur Ubuntu avec vérification: %v\n", err)
		} else {
			fmt.Printf("✓ Ubuntu avec vérification: %d paquets\n", len(verifiedUbuntuPackages))
		}
	}

	// ========================================
	// PARTIE 6: Tests de gestion d'erreurs (intégré depuis error-handling)
	// ========================================
	fmt.Println("\n⚠️ PARTIE 6: Tests de gestion d'erreurs")

	fmt.Println("\n--- Test avec URL invalide ---")
	badRepo := debian.NewRepository(
		"bad-repo",
		"http://repository-inexistant.example.com/debian",
		"Repository inexistant pour test",
		"bookworm",
		[]string{"main"},
		[]string{"amd64"},
	)

	badRepo.EnableReleaseVerification()
	err = badRepo.FetchReleaseFile()
	if err != nil {
		fmt.Printf("✓ Erreur attendue capturée: %v\n", err)
	} else {
		fmt.Printf("❌ Erreur attendue mais non capturée\n")
	}

	fmt.Println("\n--- Test avec distribution inexistante ---")
	invalidDistRepo := debian.NewRepository(
		"invalid-dist",
		"http://deb.debian.org/debian",
		"Test distribution inexistante",
		"distribution-inexistante-12345",
		[]string{"main"},
		[]string{"amd64"},
	)

	invalidDistRepo.EnableReleaseVerification()
	err = invalidDistRepo.FetchReleaseFile()
	if err != nil {
		fmt.Printf("✓ Erreur distribution inexistante capturée: %v\n", err)
	} else {
		fmt.Printf("❌ Erreur distribution inexistante non capturée\n")
	}

	fmt.Println("\n--- Test de robustesse avec timeout ---")
	// Créer un client HTTP avec timeout court
	client := &http.Client{Timeout: 2 * time.Second}

	// Utilisation directe du client pour demo
	fmt.Println("Simulation de timeout réseau...")
	_, err = client.Get("http://httpbin.org/delay/5")
	if err != nil {
		fmt.Printf("✓ Timeout correctement géré: %v\n", err)
	}

	fmt.Println("\n--- Test avec sections inexistantes ---")
	badSectionRepo := debian.NewRepository(
		"bad-section",
		"http://deb.debian.org/debian",
		"Test section inexistante",
		"bookworm",
		[]string{"section-inexistante"},
		[]string{"amd64"},
	)

	badSectionPackages, err := badSectionRepo.FetchPackages()
	if err != nil {
		fmt.Printf("✓ Erreur section inexistante capturée: %v\n", err)
	} else {
		fmt.Printf("ℹ️ Section inexistante: %d paquets trouvés (comportement inattendu)\n", len(badSectionPackages))
	}

	fmt.Println("\n--- Test avec architecture inexistante ---")
	badArchRepo := debian.NewRepository(
		"bad-arch",
		"http://deb.debian.org/debian",
		"Test architecture inexistante",
		"bookworm",
		[]string{"main"},
		[]string{"arch-inexistante"},
	)

	badArchPackages, err := badArchRepo.FetchPackages()
	if err != nil {
		fmt.Printf("✓ Erreur architecture inexistante capturée: %v\n", err)
	} else {
		fmt.Printf("ℹ️ Architecture inexistante: %d paquets trouvés (comportement inattendu)\n", len(badArchPackages))
	}

	// ========================================
	// PARTIE 7: Tests de recherche
	// ========================================
	fmt.Println("\n🔍 PARTIE 7: Recherche de paquets spécifiques")
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

	// ========================================
	// PARTIE 8: Test de la fonction SearchPackage avancée
	// ========================================
	fmt.Println("\n🔍 PARTIE 8: Test de la fonction SearchPackage avancée")

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

	// Test de recherche sur Ubuntu si disponible
	if len(ubuntuPackages) > 0 {
		fmt.Println("\n--- Tests de recherche sur Ubuntu ---")
		ubuntuSearches := []string{"firefox", "libreoffice", "python3", "inexistant-123456"}

		for _, search := range ubuntuSearches {
			matches, err := ubuntuRepo.SearchPackage(search)
			if err != nil {
				fmt.Printf("  '%s': %v\n", search, err)
			} else {
				fmt.Printf("  '%s': %d correspondances\n", search, len(matches))
			}
		}
	}

	// ========================================
	// PARTIE 9: Configuration personnalisée et tests avancés
	// ========================================
	fmt.Println("\n🔧 PARTIE 9: Configuration personnalisée et tests avancés")

	fmt.Println("\n--- Test avec configuration ARM64 ---")
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

	fmt.Println("\n--- Test multi-sections ---")
	multiSectionRepo := debian.NewRepository(
		"debian-multi-sections",
		"http://deb.debian.org/debian",
		"Test multi-sections",
		"bookworm",
		[]string{"main", "contrib"},
		[]string{"amd64"},
	)

	multiSectionPackages, err := multiSectionRepo.FetchPackages()
	if err != nil {
		fmt.Printf("✗ Erreur multi-sections: %v\n", err)
	} else {
		fmt.Printf("✓ Multi-sections (main+contrib): %d paquets\n", len(multiSectionPackages))
	}

	fmt.Println("\n--- Test multi-architectures ---")
	multiArchRepo := debian.NewRepository(
		"debian-multi-arch",
		"http://deb.debian.org/debian",
		"Test multi-architectures",
		"bookworm",
		[]string{"main"},
		[]string{"amd64", "i386"},
	)

	multiArchPackages, err := multiArchRepo.FetchPackages()
	if err != nil {
		fmt.Printf("✗ Erreur multi-architectures: %v\n", err)
	} else {
		fmt.Printf("✓ Multi-architectures (amd64+i386): %d paquets\n", len(multiArchPackages))
	}

	// ========================================
	// PARTIE 10: Tests de performance et statistiques
	// ========================================
	fmt.Println("\n📊 PARTIE 10: Statistiques et performance")

	fmt.Printf("📈 Statistiques finales:\n")
	fmt.Printf("  - Paquets Debian main/amd64: %d\n", len(packages))
	fmt.Printf("  - Paquets Ubuntu main/amd64: %d\n", len(ubuntuPackages))
	if len(verifiedPackages) > 0 {
		fmt.Printf("  - Paquets avec vérification Release: %d\n", len(verifiedPackages))
	}
	if len(customPackages) > 0 {
		fmt.Printf("  - Paquets ARM64: %d\n", len(customPackages))
	}
	if len(multiSectionPackages) > 0 {
		fmt.Printf("  - Paquets multi-sections: %d\n", len(multiSectionPackages))
	}
	if len(multiArchPackages) > 0 {
		fmt.Printf("  - Paquets multi-architectures: %d\n", len(multiArchPackages))
	}

	fmt.Println("\n✅ Tous les tests sont terminés avec succès!")
	fmt.Println("=== FIN DE L'EXEMPLE COMPLET ===")
}
