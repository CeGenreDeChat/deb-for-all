package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func CreateMirror(baseURL, suites, components, architectures, destDir string, downloadPkgs, verbose bool) error {
	// Parse comma-separated values
	suiteList := strings.Split(strings.TrimSpace(suites), ",")
	componentList := strings.Split(strings.TrimSpace(components), ",")
	architectureList := strings.Split(strings.TrimSpace(architectures), ",")

	// Trim spaces from each element
	for i, suite := range suiteList {
		suiteList[i] = strings.TrimSpace(suite)
	}
	for i, component := range componentList {
		componentList[i] = strings.TrimSpace(component)
	}
	for i, arch := range architectureList {
		architectureList[i] = strings.TrimSpace(arch)
	}

	// Create mirror configuration
	config := debian.MirrorConfig{
		BaseURL:          baseURL,
		Suites:           suiteList,
		Components:       componentList,
		Architectures:    architectureList,
		DownloadPackages: downloadPkgs,
		Verbose:          verbose,
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration invalide: %v", err)
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %v", err)
	}

	// Create mirror
	mirror := debian.NewMirror(config, destDir)

	if verbose {
		fmt.Println("=== Configuration du Miroir ===")
		info := mirror.GetMirrorInfo()
		for key, value := range info {
			fmt.Printf("%s: %v\n", key, value)
		}
		fmt.Println()
	}

	// Check current status
	if verbose {
		fmt.Println("=== Statut du Miroir ===")
		status, err := mirror.GetMirrorStatus()
		if err != nil {
			fmt.Printf("Erreur lors de la vérification du statut: %v\n", err)
		} else {
			for key, value := range status {
				fmt.Printf("%s: %v\n", key, value)
			}
		}
		fmt.Println()
	}

	// Start mirroring
	if verbose {
		fmt.Println("=== Démarrage du Miroir ===")
	}

	if err := mirror.Clone(); err != nil {
		return fmt.Errorf("erreur lors de la création du miroir: %v", err)
	}

	if verbose {
		fmt.Println("✓ Miroir créé avec succès!")

		// Show final status
		fmt.Println("\n=== Statut Final ===")
		status, err := mirror.GetMirrorStatus()
		if err == nil {
			for key, value := range status {
				fmt.Printf("%s: %v\n", key, value)
			}
		}
	}

	return nil
}
