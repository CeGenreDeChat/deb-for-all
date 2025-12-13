package commands

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type xmlPackageList struct {
	Packages []xmlPackageEntry `xml:"package"`
}

type xmlPackageEntry struct {
	Name    string `xml:",chardata"`
	Version string `xml:"version,attr"`
}

// BuildCustomRepository builds a custom repository subset from an XML package list,
// resolves dependencies (with optional exclusions), and downloads the resulting packages.
func BuildCustomRepository(baseURL, suites, components, architectures, destDir, packagesXML, excludeDeps string, keyrings []string, skipGPGVerify, verbose bool, localizer *i18n.Localizer) error {
	if packagesXML == "" {
		return fmt.Errorf("packages XML file is required")
	}

	packageSpecs, err := loadPackageSpecs(packagesXML)
	if err != nil {
		return err
	}

	excludeSet, err := parseExcludeDeps(excludeDeps, localizer)
	if err != nil {
		return fmt.Errorf("invalid --exclude-deps value: %w", err)
	}

	suiteList := splitAndTrim(suites)
	componentList := splitAndTrim(components)
	archList := splitAndTrim(architectures)

	if len(suiteList) == 0 {
		return fmt.Errorf("at least one suite is required")
	}
	if len(componentList) == 0 {
		return fmt.Errorf("at least one component is required")
	}
	if len(archList) == 0 {
		return fmt.Errorf("at least one architecture is required")
	}

	if err := os.MkdirAll(destDir, debian.DirPermission); err != nil {
		return fmt.Errorf("unable to create destination directory: %w", err)
	}

	for _, suite := range suiteList {
		repo := debian.NewRepository("custom-repo"+suite, baseURL, "custom repo", suite, componentList, archList)
		repo.SetKeyringPaths(keyrings)
		if skipGPGVerify {
			repo.DisableSignatureVerification()
		}

		if err := validateComponentsAndArchitectures(repo, suite, componentList, archList, localizer); err != nil {
			return err
		}

		if verbose {
			fmt.Printf("Suite %s: fetching metadata...\n", suite)
		}

		if _, err := repo.FetchPackages(); err != nil {
			return fmt.Errorf("failed to fetch packages for %s: %w", suite, err)
		}

		resolved, err := repo.ResolveDependencies(packageSpecs, excludeSet)
		if err != nil {
			return fmt.Errorf("failed to resolve dependencies for %s: %w", suite, err)
		}

		if verbose {
			fmt.Printf("Suite %s: %d packages to download\n", suite, len(resolved))
		}

		downloader := debian.NewDownloader()
		poolDir := filepath.Join(destDir, "pool", suite)

		for _, pkg := range resolved {
			targetDir := filepath.Join(poolDir, pkg.Section)
			if targetDir == poolDir || pkg.Section == "" {
				targetDir = filepath.Join(poolDir, "main")
			}

			destPath := filepath.Join(targetDir, packageFilename(&pkg))
			skip, err := downloader.ShouldSkipDownload(&pkg, destPath)
			if err != nil {
				return fmt.Errorf("failed to check existing file for %s: %w", pkg.Name, err)
			}
			if skip {
				if verbose {
					fmt.Printf("Suite %s: skipping %s (already downloaded, checksum verified)\n", suite, pkg.Name)
				}
				continue
			}

			if err := downloader.DownloadToDir(&pkg, targetDir); err != nil {
				return fmt.Errorf("failed to download %s: %w", pkg.Name, err)
			}
		}
	}

	return nil
}

func loadPackageSpecs(path string) ([]debian.PackageSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read XML file: %w", err)
	}

	var list xmlPackageList
	if err := xml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("invalid XML format: %w", err)
	}

	specs := make([]debian.PackageSpec, 0, len(list.Packages))
	for _, entry := range list.Packages {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		specs = append(specs, debian.PackageSpec{Name: name, Version: strings.TrimSpace(entry.Version)})
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("no valid package found in XML")
	}

	return specs, nil
}

var (
	allowedExcludeDepKinds    = []string{"depends", "pre-depends", "recommends", "suggests", "enhances"}
	allowedExcludeDepKindsSet = map[string]struct{}{
		"depends":     {},
		"pre-depends": {},
		"recommends":  {},
		"suggests":    {},
		"enhances":    {},
	}
)

func parseExcludeDeps(value string, localizer *i18n.Localizer) (map[string]bool, error) {
	set := make(map[string]bool)

	for _, item := range strings.Split(strings.TrimSpace(value), ",") {
		trimmed := strings.ToLower(strings.TrimSpace(item))
		if trimmed == "" {
			continue
		}

		if _, ok := allowedExcludeDepKindsSet[trimmed]; !ok {
			fallback := fmt.Sprintf("unknown dependency kind '%s' (allowed: %s)", trimmed, strings.Join(allowedExcludeDepKinds, ", "))
			msg := localizeMessage(localizer, "error.custom_repo.unknown_dependency_kind", fallback, map[string]any{
				"Kind":    trimmed,
				"Allowed": strings.Join(allowedExcludeDepKinds, ", "),
			})
			return nil, fmt.Errorf("%s", msg)
		}

		set[trimmed] = true
	}

	return set, nil
}

func localizeMessage(localizer *i18n.Localizer, messageID, fallback string, data map[string]any) string {
	if localizer == nil {
		return fallback
	}

	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: messageID, TemplateData: data})
	if err == nil && msg != "" {
		return msg
	}

	return fallback
}
