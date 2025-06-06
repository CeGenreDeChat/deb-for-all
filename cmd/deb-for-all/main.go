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
		command     = flag.String("command", "", "Commande à exécuter: download, download-source")
		packageName = flag.String("package", "", "Nom du paquet")
		version     = flag.String("version", "", "Version du paquet")
		destDir     = flag.String("dest", "./downloads", "Répertoire de destination")
		origOnly    = flag.Bool("orig-only", false, "Télécharger uniquement le tarball original (pour les paquets sources)")
		silent      = flag.Bool("silent", false, "Mode silencieux sans affichage de progression")
		help        = flag.Bool("help", false, "Afficher l'aide")
	)

	flag.Parse()

	if *help || *command == "" {
		printUsage()
		return
	}

	if err := run(*command, *packageName, *version, *destDir, *origOnly, *silent); err != nil {
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
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -package <nom>     Nom du paquet")
	fmt.Println("  -version <version> Version du paquet")
	fmt.Println("  -dest <répertoire> Répertoire de destination (défaut: ./downloads)")
	fmt.Println("  -orig-only         Télécharger uniquement le tarball original (paquets sources)")
	fmt.Println("  -silent            Mode silencieux")
	fmt.Println("  -help              Afficher cette aide")
	fmt.Println("")
	fmt.Println("Exemples:")
	fmt.Println("  # Télécharger un paquet source")
	fmt.Println("  deb-for-all -command download-source -package hello -version 2.10-2")
	fmt.Println("")
	fmt.Println("  # Télécharger uniquement le tarball original")
	fmt.Println("  deb-for-all -command download-source -package hello -version 2.10-2 -orig-only")
	fmt.Println("")
	fmt.Println("  # Téléchargement silencieux")
	fmt.Println("  deb-for-all -command download-source -package hello -version 2.10-2 -silent")
}

func run(command, packageName, version, destDir string, origOnly, silent bool) error {
	switch strings.ToLower(command) {
	case "download-source":
		return downloadSourcePackage(packageName, version, destDir, origOnly, silent)
	case "download":
		return fmt.Errorf("commande download pas encore implémentée")
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
