package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func CreateMirror(baseURL, suites, components, architectures, destDir string, downloadPkgs, verbose bool, keyrings []string, skipGPGVerify bool, rateLimit int, localizer *i18n.Localizer) error {
	if verbose {
		fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "command.mirror.start",
			TemplateData: map[string]any{
				"URL": baseURL,
			},
		}))
		fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "command.mirror.details",
			TemplateData: map[string]any{
				"Suites":        suites,
				"Components":    components,
				"Architectures": architectures,
				"Dest":          destDir,
			},
		}))
	}

	suiteList := splitAndTrim(suites)
	componentList := splitAndTrim(components)
	architectureList := splitAndTrim(architectures)

	if len(suiteList) == 0 {
		return fmt.Errorf("at least one suite is required")
	}
	if len(componentList) == 0 {
		return fmt.Errorf("at least one component is required")
	}
	if len(architectureList) == 0 {
		return fmt.Errorf("at least one architecture is required")
	}

	// Create mirror configuration
	config := debian.MirrorConfig{
		BaseURL:          baseURL,
		Suites:           suiteList,
		Components:       componentList,
		Architectures:    architectureList,
		DownloadPackages: downloadPkgs,
		Verbose:          verbose,
		KeyringPaths:     keyrings,
		SkipGPGVerify:    skipGPGVerify,
		RateDelay:        time.Duration(rateLimit) * time.Second,
	}

	for _, suite := range suiteList {
		repo := debian.NewRepository("mirror-validate"+suite, baseURL, "mirror validation", suite, componentList, architectureList)
		repo.SetKeyringPaths(keyrings)
		if skipGPGVerify {
			repo.DisableSignatureVerification()
		}

		if err := validateComponentsAndArchitectures(repo, suite, componentList, architectureList, localizer); err != nil {
			return fmt.Errorf("invalid suite %s: %w", suite, err)
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, debian.DirPermission); err != nil {
		return fmt.Errorf("unable to create destination directory: %w", err)
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
			fmt.Printf("Error checking status: %v\n", err)
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
		return fmt.Errorf("failed to create mirror: %w", err)
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
