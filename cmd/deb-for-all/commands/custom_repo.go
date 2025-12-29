package commands

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
func BuildCustomRepository(baseURL, suites, components, architectures, destDir, packagesXML, excludeDeps string, keyrings []string, skipGPGVerify, verbose bool, rateLimit int, localizer *i18n.Localizer) error {
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

	metadataRoot := filepath.Join(destDir, "dists")
	if err := os.MkdirAll(metadataRoot, debian.DirPermission); err != nil {
		return fmt.Errorf("unable to create metadata directory: %w", err)
	}

	for _, suite := range suiteList {
		repo := debian.NewRepository("custom-repo"+suite, baseURL, "custom repo", suite, componentList, archList)
		repo.SetKeyringPaths(keyrings)
		if skipGPGVerify {
			repo.DisableSignatureVerification()
		}

		packageMetadata := make(map[string]map[string][]debian.Package)
		downloader := debian.NewDownloader()
		downloader.RateDelay = time.Duration(rateLimit) * time.Second

		for _, component := range componentList {
			repo.SetSections([]string{component})

			if err := validateComponentsAndArchitectures(repo, suite, []string{component}, archList, localizer); err != nil {
				return err
			}

			if verbose {
				fmt.Printf("Suite %s component %s: fetching metadata...\n", suite, component)
			}

			if _, err := repo.FetchPackages(); err != nil {
				return fmt.Errorf("failed to fetch packages for %s/%s: %w", suite, component, err)
			}

			resolved, err := repo.ResolveDependencies(packageSpecs, excludeSet)
			if err != nil {
				return fmt.Errorf("failed to resolve dependencies for %s/%s: %w", suite, component, err)
			}

			if verbose {
				fmt.Printf("Suite %s component %s: %d packages to download\n", suite, component, len(resolved))
			}

			for _, pkg := range resolved {
				arch := pkg.Architecture
				if arch == "" {
					arch = archList[0]
				}

				if _, ok := packageMetadata[component]; !ok {
					packageMetadata[component] = make(map[string][]debian.Package)
				}

				relPath := pkg.Filename
				if relPath == "" {
					filename := filepath.Base(packageFilename(&pkg))
					relPath = filepath.ToSlash(filepath.Join("pool", component, filename))
				}

				targetPath := filepath.Join(destDir, filepath.FromSlash(relPath))
				targetDir := filepath.Dir(targetPath)

				skip, err := downloader.ShouldSkipDownload(&pkg, targetPath)
				if err != nil {
					return fmt.Errorf("failed to check existing file for %s: %w", pkg.Name, err)
				}
				if skip {
					if verbose {
						fmt.Printf("Suite %s component %s: skipping %s (already downloaded, checksum verified)\n", suite, component, pkg.Name)
					}
					pkg.Filename = filepath.ToSlash(relPath)
					packageMetadata[component][arch] = append(packageMetadata[component][arch], pkg)
					continue
				}

				if err := os.MkdirAll(targetDir, debian.DirPermission); err != nil {
					return fmt.Errorf("unable to create pool directory %s: %w", targetDir, err)
				}

				if err := downloader.DownloadWithProgress(&pkg, targetPath, nil); err != nil {
					return fmt.Errorf("failed to download %s: %w", pkg.Name, err)
				}

				pkg.Filename = filepath.ToSlash(relPath)
				packageMetadata[component][arch] = append(packageMetadata[component][arch], pkg)
			}
		}

		if err := debian.WritePackagesMetadata(metadataRoot, suite, packageMetadata); err != nil {
			return err
		}

		if err := debian.WriteReleaseFiles(metadataRoot, suite, componentList, archList); err != nil {
			return fmt.Errorf("failed to write Release files for suite %s: %w", suite, err)
		}
	}

	return nil
}

func formatPackagesFile(packages []debian.Package) string {
	var sb strings.Builder

	for _, pkg := range packages {
		writeField := func(name, value string) {
			if value != "" {
				sb.WriteString(name)
				sb.WriteString(": ")
				sb.WriteString(value)
				sb.WriteString("\n")
			}
		}

		writeField("Package", pkg.Package)
		writeField("Version", pkg.Version)
		writeField("Architecture", pkg.Architecture)
		writeField("Maintainer", pkg.Maintainer)
		writeField("Section", pkg.Section)
		writeField("Priority", pkg.Priority)
		writeField("Filename", pkg.Filename)
		if pkg.Size > 0 {
			sb.WriteString("Size: ")
			sb.WriteString(fmt.Sprintf("%d\n", pkg.Size))
		}
		writeField("MD5sum", pkg.MD5sum)
		writeField("SHA256", pkg.SHA256)
		writeListField(&sb, "Depends", pkg.Depends)
		writeListField(&sb, "Pre-Depends", pkg.PreDepends)
		writeListField(&sb, "Recommends", pkg.Recommends)
		writeListField(&sb, "Suggests", pkg.Suggests)
		writeListField(&sb, "Breaks", pkg.Breaks)
		writeListField(&sb, "Conflicts", pkg.Conflicts)
		writeListField(&sb, "Provides", pkg.Provides)
		writeListField(&sb, "Replaces", pkg.Replaces)

		if pkg.Description != "" {
			sb.WriteString("Description: ")
			sb.WriteString(pkg.Description)
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func writeListField(sb *strings.Builder, name string, values []string) {
	if len(values) == 0 {
		return
	}

	sb.WriteString(name)
	sb.WriteString(": ")
	sb.WriteString(strings.Join(values, ", "))
	sb.WriteString("\n")
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
