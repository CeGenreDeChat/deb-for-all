package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/CeGenreDeChat/deb-for-all/cmd/deb-for-all/commands"
)

func main() {
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

func run(command, packageName, version, destDir string, origOnly, silent bool,
	baseURL, suites, components, architectures string, downloadPkgs, verbose bool) error {
	switch strings.ToLower(command) {
	case "download-source":
		return commands.DownloadSourcePackage(packageName, version, destDir, origOnly, silent)
	case "download":
		return commands.DownloadBinaryPackage(packageName, version, destDir, silent)
	case "mirror":
		return commands.CreateMirror(baseURL, suites, components, architectures, destDir, downloadPkgs, verbose)
	default:
		return fmt.Errorf("commande inconnue: %s", command)
	}
}
