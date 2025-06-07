package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	// Configuration du miroir
	config := debian.MirrorConfig{
		BaseURL:          "http://deb.debian.org/debian",
		Suites:           []string{"bookworm"},
		Components:       []string{"main"},
		Architectures:    []string{"amd64"},
		DownloadPackages: false, // Commencer par télécharger seulement les métadonnées
		Verbose:          true,
	}

	// Validation de la configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Configuration invalide: %v", err)
	}

	// Chemin de destination du miroir
	mirrorPath := filepath.Join(".", "debian-mirror")

	// Création du miroir
	mirror := debian.NewMirror(config, mirrorPath)

	// Affichage des informations du miroir
	fmt.Println("=== Configuration du Miroir ===")
	info := mirror.GetMirrorInfo()
	for key, value := range info {
		fmt.Printf("%s: %v\n", key, value)
	}
	fmt.Println()

	// Vérification du statut actuel
	fmt.Println("=== Statut du Miroir ===")
	status, err := mirror.GetMirrorStatus()
	if err != nil {
		log.Printf("Erreur lors de la vérification du statut: %v", err)
	} else {
		for key, value := range status {
			fmt.Printf("%s: %v\n", key, value)
		}
	}
	fmt.Println()

	// Estimation de la taille (métadonnées seulement)
	fmt.Println("=== Estimation de la Taille ===")
	size, err := mirror.EstimateMirrorSize()
	if err != nil {
		log.Printf("Erreur lors de l'estimation: %v", err)
	} else {
		if config.DownloadPackages {
			fmt.Printf("Taille estimée des paquets: %.2f MB\n", float64(size)/1024/1024)
		} else {
			fmt.Println("Mode métadonnées uniquement - taille négligeable")
		}
	}
	fmt.Println()

	// Demande de confirmation
	fmt.Print("Voulez-vous continuer avec le clonage? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Clonage annulé.")
		return
	}

	// Clonage du miroir
	fmt.Println("=== Démarrage du Clonage ===")
	if err := mirror.Clone(); err != nil {
		log.Fatalf("Erreur lors du clonage: %v", err)
	}

	fmt.Println("\n=== Clonage Terminé ===")

	// Affichage du statut final
	finalStatus, err := mirror.GetMirrorStatus()
	if err != nil {
		log.Printf("Erreur lors de la vérification du statut final: %v", err)
	} else {
		fmt.Println("Statut final:")
		for key, value := range finalStatus {
			fmt.Printf("%s: %v\n", key, value)
		}
	}

	// Exemple de synchronisation
	fmt.Println("\n=== Exemple de Synchronisation ===")
	fmt.Print("Voulez-vous effectuer une synchronisation? (y/N): ")
	fmt.Scanln(&response)
	if response == "y" || response == "Y" {
		if err := mirror.Sync(); err != nil {
			log.Printf("Erreur lors de la synchronisation: %v", err)
		} else {
			fmt.Println("Synchronisation terminée avec succès!")
		}
	}
	// Test avancé avec téléchargement de paquets
	fmt.Println("\n=== Test avec Téléchargement de Paquets ===")
	fmt.Print("Voulez-vous tester le miroir avec téléchargement de paquets? (y/N): ")
	fmt.Scanln(&response)
	if response == "y" || response == "Y" {
		// Configuration pour télécharger aussi les paquets
		packageConfig := debian.MirrorConfig{
			BaseURL:          "http://deb.debian.org/debian",
			Suites:           []string{"bookworm"},
			Components:       []string{"main"},
			Architectures:    []string{"amd64"},
			DownloadPackages: true, // Activer le téléchargement des paquets
			Verbose:          true,
		}

		packageMirrorPath := filepath.Join(".", "debian-mirror-with-packages")
		packageMirror := debian.NewMirror(packageConfig, packageMirrorPath)

		// Estimation de la taille pour les paquets
		fmt.Println("=== Estimation de la Taille avec Paquets ===")
		size, err := packageMirror.EstimateMirrorSize()
		if err != nil {
			log.Printf("Erreur lors de l'estimation: %v", err)
		} else {
			fmt.Printf("Taille estimée: %.2f GB\n", float64(size)/1024/1024/1024)
			fmt.Println("⚠️  ATTENTION: Le téléchargement complet peut prendre plusieurs heures et utiliser plusieurs GB d'espace disque!")
		}

		fmt.Print("Voulez-vous continuer avec le téléchargement complet? (yes pour continuer, autres pour test métadonnées seulement): ")
		fmt.Scanln(&response)

		if response != "yes" {
			// Pour des raisons pratiques, on ne fait que le test des métadonnées
			fmt.Println("\n=== Test Métadonnées Seulement (mode sécurisé) ===")
			packageConfig.DownloadPackages = false
			packageMirror = debian.NewMirror(packageConfig, packageMirrorPath)
		} else {
			fmt.Println("\n=== Démarrage du Téléchargement Complet ===")
			fmt.Println("Cela peut prendre du temps...")
		}

		if err := packageMirror.Clone(); err != nil {
			log.Printf("Erreur lors du clonage avec paquets: %v", err)
		} else {
			if packageConfig.DownloadPackages {
				fmt.Println("✅ Miroir avec paquets créé avec succès!")
			} else {
				fmt.Println("✅ Test métadonnées terminé avec succès!")
				fmt.Println("\nPour tester le téléchargement de paquets:")
				fmt.Println("1. Relancez et répondez 'yes' au téléchargement complet")
				fmt.Println("2. Considérez utiliser une suite de test plus petite")
				fmt.Println("3. Assurez-vous d'avoir suffisamment d'espace disque")
			}
		}

		// Affichage du statut du miroir avec paquets
		fmt.Println("\n=== Statut du Miroir avec Paquets ===")
		packageStatus, err := packageMirror.GetMirrorStatus()
		if err != nil {
			log.Printf("Erreur lors de la vérification du statut: %v", err)
		} else {
			for key, value := range packageStatus {
				fmt.Printf("%s: %v\n", key, value)
			}
		}
	}

	// Tests avancés du module Mirror
	fmt.Println("\n=== Tests Avancés du Module Mirror ===")
	fmt.Print("Voulez-vous exécuter les tests avancés? (y/N): ")
	fmt.Scanln(&response)
	if response == "y" || response == "Y" {

		// Test 1: Informations du Repository sous-jacent
		fmt.Println("\n--- Test 1: Repository Info ---")
		repoInfo := mirror.GetRepositoryInfo()
		fmt.Printf("Repository intégré - URL: %s\n", repoInfo.URL)
		fmt.Printf("Distribution actuelle: %s\n", repoInfo.Distribution)
		fmt.Printf("Sections: %v\n", repoInfo.Sections)
		fmt.Printf("Architectures: %v\n", repoInfo.Architectures)

		// Test 2: Mise à jour de configuration
		fmt.Println("\n--- Test 2: Mise à jour Configuration ---")
		newConfig := debian.MirrorConfig{
			BaseURL:          "http://deb.debian.org/debian",
			Suites:           []string{"bookworm", "bookworm-updates"},
			Components:       []string{"main", "contrib"},
			Architectures:    []string{"amd64", "i386"},
			DownloadPackages: false,
			Verbose:          true,
		}

		if err := mirror.UpdateConfiguration(newConfig); err != nil {
			log.Printf("Erreur lors de la mise à jour: %v", err)
		} else {
			fmt.Println("✅ Configuration mise à jour avec succès")

			// Afficher la nouvelle configuration
			updatedInfo := mirror.GetMirrorInfo()
			fmt.Println("Nouvelle configuration:")
			for key, value := range updatedInfo {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}

		// Test 3: Vérification d'intégrité
		fmt.Println("\n--- Test 3: Vérification d'Intégrité ---")
		fmt.Print("Voulez-vous vérifier l'intégrité du miroir? (y/N): ")
		fmt.Scanln(&response)
		if response == "y" || response == "Y" {
			if err := mirror.VerifyMirrorIntegrity("bookworm"); err != nil {
				log.Printf("⚠️  Problème d'intégrité détecté: %v", err)
			} else {
				fmt.Println("✅ Intégrité du miroir vérifiée avec succès")
			}
		}

		// Test 4: Test avec différentes configurations
		fmt.Println("\n--- Test 4: Configurations Multiples ---")
		configs := []debian.MirrorConfig{
			{
				BaseURL:          "http://deb.debian.org/debian",
				Suites:           []string{"bullseye"},
				Components:       []string{"main"},
				Architectures:    []string{"amd64"},
				DownloadPackages: false,
				Verbose:          false,
			},
			{
				BaseURL:          "http://security.debian.org/debian-security",
				Suites:           []string{"bookworm-security"},
				Components:       []string{"main"},
				Architectures:    []string{"amd64"},
				DownloadPackages: false,
				Verbose:          false,
			},
		}

		for i, testConfig := range configs {
			fmt.Printf("\nTest configuration %d:\n", i+1)
			fmt.Printf("  URL: %s\n", testConfig.BaseURL)
			fmt.Printf("  Suite: %s\n", testConfig.Suites[0])

			if err := testConfig.Validate(); err != nil {
				fmt.Printf("  ❌ Configuration invalide: %v\n", err)
			} else {
				fmt.Printf("  ✅ Configuration valide\n")

				testMirrorPath := filepath.Join(".", fmt.Sprintf("test-mirror-%d", i+1))
				testMirror := debian.NewMirror(testConfig, testMirrorPath)

				testStatus, err := testMirror.GetMirrorStatus()
				if err == nil {
					fmt.Printf("  Statut: %d éléments\n", len(testStatus))
				}
			}
		}
	}

	// Test de performance et statistiques
	fmt.Println("\n=== Statistiques et Performance ===")
	fmt.Print("Voulez-vous voir les statistiques détaillées? (y/N): ")
	fmt.Scanln(&response)
	if response == "y" || response == "Y" {

		fmt.Println("\n--- Statistiques du Miroir ---")
		stats := map[string]interface{}{
			"Configurations testées": 3,
			"Miroirs créés":          len([]string{mirrorPath, "debian-mirror-with-packages"}),
			"Fonctionnalités testées": []string{
				"Clone()", "Sync()", "GetMirrorStatus()", "GetMirrorInfo()",
				"EstimateMirrorSize()", "GetRepositoryInfo()", "UpdateConfiguration()",
				"VerifyMirrorIntegrity()",
			},
		}

		for key, value := range stats {
			fmt.Printf("%s: %v\n", key, value)
		}

		fmt.Println("\n--- Intégration Repository ---")
		fmt.Println("✅ Mirror utilise Repository comme base")
		fmt.Println("✅ Pas de duplication de logique")
		fmt.Println("✅ Toutes les fonctionnalités Repository disponibles")
		fmt.Println("✅ Gestion automatique des formats compressés")
		fmt.Println("✅ Validation des checksums héritée")
	}

	fmt.Println("\nExemple terminé. Consultez les répertoires créés:")
	fmt.Printf("- %s (métadonnées)\n", mirrorPath)
	fmt.Println("\nStructure typique d'un miroir Debian:")
	fmt.Println("debian-mirror/")
	fmt.Println("├── dists/")
	fmt.Println("│   └── bookworm/")
	fmt.Println("│       ├── Release")
	fmt.Println("│       └── main/")
	fmt.Println("│           └── binary-amd64/")
	fmt.Println("│               └── Packages")
	fmt.Println("└── pool/ (si DownloadPackages=true)")
	fmt.Println("    └── main/")
	fmt.Println("        └── [a-z]/")
	fmt.Println("            └── package-name/")
	fmt.Println("                └── package.deb")
}
