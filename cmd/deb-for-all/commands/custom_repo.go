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
// If gpgKeyPath is provided, the Release files will be signed with the GPG key.
func BuildCustomRepository(baseURL, suites, components, architectures, destDir, packagesXML, excludeDeps string, keyrings, keyringDirs []string, skipGPGVerify, verbose bool, rateLimit int, includeSources bool, gpgKeyPath, gpgPassphrase string, localizer *i18n.Localizer) error {
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
		repo.SetKeyringPathsWithDirs(keyrings, keyringDirs)
		if skipGPGVerify {
			repo.DisableSignatureVerification()
		}

		packageMetadata := make(map[string]map[string][]debian.Package)
		sourceMetadata := make(map[string][]debian.SourcePackage)
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

			// Download source packages if requested
			if includeSources {
				resolvedSlice := make([]debian.Package, 0, len(resolved))
				for _, pkg := range resolved {
					resolvedSlice = append(resolvedSlice, pkg)
				}
				srcPkgs, err := downloadSourcePackages(repo, resolvedSlice, destDir, component, downloader, verbose, suite)
				if err != nil {
					return fmt.Errorf("failed to download source packages for %s/%s: %w", suite, component, err)
				}
				sourceMetadata[component] = append(sourceMetadata[component], srcPkgs...)
			}
		}

		if err := debian.WritePackagesMetadata(metadataRoot, suite, packageMetadata); err != nil {
			return err
		}

		if includeSources && len(sourceMetadata) > 0 {
			if err := debian.WriteSourcesMetadata(metadataRoot, suite, sourceMetadata); err != nil {
				return err
			}
		}

		// Build signing config if GPG key is provided
		var signingConfig *debian.ReleaseSigningConfig
		if gpgKeyPath != "" {
			signingConfig = &debian.ReleaseSigningConfig{
				PrivateKeyPath: gpgKeyPath,
				Passphrase:     gpgPassphrase,
			}
			if verbose {
				fmt.Printf("Suite %s: signing Release files with GPG key %s\n", suite, gpgKeyPath)
			}
		} else if verbose {
			fmt.Printf("Suite %s: no GPG key provided, Release files will be unsigned\n", suite)
		}

		if err := debian.WriteSignedReleaseFiles(metadataRoot, suite, componentList, archList, includeSources && len(sourceMetadata) > 0, signingConfig); err != nil {
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

// downloadSourcePackages downloads source packages corresponding to the resolved binary packages.
func downloadSourcePackages(repo *debian.Repository, resolved []debian.Package, destDir, component string, downloader *debian.Downloader, verbose bool, suite string) ([]debian.SourcePackage, error) {
	// Get unique source package names from binary packages
	sourceNames := make(map[string]struct{})
	for _, pkg := range resolved {
		srcName := pkg.Source
		if srcName == "" {
			srcName = pkg.Package
		}
		// Strip version info if present (e.g., "foo (>= 1.0)" -> "foo")
		if idx := strings.Index(srcName, " "); idx != -1 {
			srcName = srcName[:idx]
		}
		sourceNames[srcName] = struct{}{}
	}

	// Fetch source packages metadata
	if _, err := repo.FetchSources(); err != nil {
		return nil, fmt.Errorf("failed to fetch source packages: %w", err)
	}
	sourcePkgs := repo.GetAllSourceMetadata()

	var result []debian.SourcePackage
	for srcName := range sourceNames {
		srcPkg, found := findSourcePackage(sourcePkgs, srcName)
		if !found {
			if verbose {
				fmt.Printf("Suite %s component %s: source package %s not found, skipping\n", suite, component, srcName)
			}
			continue
		}

		if verbose {
			fmt.Printf("Suite %s component %s: downloading source %s\n", suite, component, srcName)
		}

		// Download all files for this source package
		updatedFiles := make([]debian.SourceFile, 0, len(srcPkg.Files))
		for _, file := range srcPkg.Files {
			relPath := filepath.ToSlash(filepath.Join("pool", component, poolPrefix(srcName), srcName, file.Name))

			targetPath := filepath.Join(destDir, filepath.FromSlash(relPath))
			targetDir := filepath.Dir(targetPath)

			if err := os.MkdirAll(targetDir, debian.DirPermission); err != nil {
				return nil, fmt.Errorf("unable to create pool directory %s: %w", targetDir, err)
			}

			// Check if file already exists with correct checksum
			if info, statErr := os.Stat(targetPath); statErr == nil && info.Size() == file.Size {
				if verbose {
					fmt.Printf("Suite %s component %s: skipping source file %s (already exists)\n", suite, component, file.Name)
				}
				updatedFiles = append(updatedFiles, file)
				continue
			}

			downloadURL := file.URL
			if downloadURL == "" {
				downloadURL = fmt.Sprintf("%s/%s", repo.URL, relPath)
			}
			if err := downloader.DownloadURL(downloadURL, targetPath); err != nil {
				return nil, fmt.Errorf("failed to download source file %s: %w", file.Name, err)
			}

			updatedFiles = append(updatedFiles, file)
		}

		srcPkg.Files = updatedFiles
		srcPkg.Directory = filepath.ToSlash(filepath.Join("pool", component, poolPrefix(srcName), srcName))
		result = append(result, srcPkg)
	}

	return result, nil
}

// findSourcePackage finds a source package by name in the list.
func findSourcePackage(packages []debian.SourcePackage, name string) (debian.SourcePackage, bool) {
	for _, pkg := range packages {
		if pkg.Name == name {
			return pkg, true
		}
	}
	return debian.SourcePackage{}, false
}

// poolPrefix returns the pool prefix for a package name (lib* uses 4-char, others 1-char).
func poolPrefix(name string) string {
	if strings.HasPrefix(name, "lib") && len(name) > 3 {
		return name[:4]
	}
	return name[:1]
}
