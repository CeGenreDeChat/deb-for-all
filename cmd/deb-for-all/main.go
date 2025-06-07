package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
	var (
		command     = flag.String("command", "", "Commande à exécuter: download, download-source, mirror")
		packageName = flag.String("package", "", "Nom du paquet")
		version     = flag.String("version", "", "Version du paquet")
		destDir     = flag.String("dest", "./downloads", "Répertoire de destination")
		origOnly    = flag.Bool("orig-only", false, "Télécharger uniquement le tarball original (pour les paquets sources)")
		silent      = flag.Bool("silent", false, "Mode silencieux sans affichage de progression")
		help        = flag.Bool("help", false, "Afficher l'aide")

		// Mirror-specific flags
		baseURL       = flag.String("url", "http://deb.debian.org/debian", "URL du dépôt à mettre en miroir")
		suites        = flag.String("suites", "bookworm", "Suites à mettre en miroir (séparées par des virgules)")
		components    = flag.String("components", "main", "Composants à mettre en miroir (séparés par des virgules)")
		architectures = flag.String("architectures", "amd64", "Architectures à mettre en miroir (séparées par des virgules)")
		downloadPkgs  = flag.Bool("download-packages", false, "Télécharger les paquets .deb (pas seulement les métadonnées)")
		verbose       = flag.Bool("verbose", false, "Affichage verbeux")
	)

	flag.Parse()

	if *help || *command == "" {
		printUsage()
		return
	}

	if err := run(*command, *packageName, *version, *destDir, *origOnly, *silent,
		*baseURL, *suites, *components, *architectures, *downloadPkgs, *verbose); err != nil {
		log.Fatalf("Erreur: %v", err)
	}
}

func printUsage() {
	fmt.Println("deb-for-all - Outil de gestion des paquets Debian")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  deb-for-all -command <commande> [options]")
	fmt.Println("")
	fmt.Println("Commandes:")
	fmt.Println("  download        Télécharger un paquet binaire")
	fmt.Println("  download-source Télécharger un paquet source")
	fmt.Println("  mirror          Créer un miroir d'un dépôt Debian")
	fmt.Println("")
	fmt.Println("Options générales:")
	fmt.Println("  -package <nom>     Nom du paquet")
	fmt.Println("  -version <version> Version du paquet")
	fmt.Println("  -dest <répertoire> Répertoire de destination (défaut: ./downloads)")
	fmt.Println("  -orig-only         Télécharger uniquement le tarball original (paquets sources)")
	fmt.Println("  -silent            Mode silencieux")
	fmt.Println("  -help              Afficher cette aide")
	fmt.Println("")
	fmt.Println("Options pour le miroir:")
	fmt.Println("  -url <URL>               URL du dépôt (défaut: http://deb.debian.org/debian)")
	fmt.Println("  -suites <suites>         Suites séparées par des virgules (défaut: bookworm)")
	fmt.Println("  -components <comps>      Composants séparés par des virgules (défaut: main)")
	fmt.Println("  -architectures <archs>   Architectures séparées par des virgules (défaut: amd64)")
	fmt.Println("  -download-packages       Télécharger les paquets .deb (pas seulement métadonnées)")
	fmt.Println("  -verbose                 Affichage verbeux")
	fmt.Println("")
	fmt.Println("Exemples:")
	fmt.Println("  # Télécharger un paquet source")
	fmt.Println("  deb-for-all -command download-source -package hello -version 2.10-2")
	fmt.Println("")
	fmt.Println("  # Créer un miroir des métadonnées seulement")
	fmt.Println("  deb-for-all -command mirror -dest ./debian-mirror -verbose")
	fmt.Println("")
	fmt.Println("  # Créer un miroir complet avec paquets")
	fmt.Println("  deb-for-all -command mirror -dest ./debian-mirror -download-packages -verbose")
	fmt.Println("")
	fmt.Println("  # Miroir personnalisé")
	fmt.Println("  deb-for-all -command mirror -url http://deb.debian.org/debian \\")
	fmt.Println("             -suites bookworm,bullseye -components main,contrib \\")
	fmt.Println("             -architectures amd64,arm64 -dest ./my-mirror")
}

func run(command, packageName, version, destDir string, origOnly, silent bool,
	baseURL, suites, components, architectures string, downloadPkgs, verbose bool) error {
	switch strings.ToLower(command) {
	case "download-source":
		return downloadSourcePackage(packageName, version, destDir, origOnly, silent)
	case "download":
		return fmt.Errorf("commande download pas encore implémentée")
	case "mirror":
		return createMirror(baseURL, suites, components, architectures, destDir, downloadPkgs, verbose)
	default:
		return fmt.Errorf("commande inconnue: %s", command)
	}
}

func downloadSourcePackage(packageName, version, destDir string, origOnly, silent bool) error {
	if packageName == "" {
		return fmt.Errorf("le nom du paquet est requis")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %v", err)
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
		return fmt.Errorf("erreur lors du téléchargement: %v", err)
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

func createMirror(baseURL, suites, components, architectures, destDir string, downloadPkgs, verbose bool) error {
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
