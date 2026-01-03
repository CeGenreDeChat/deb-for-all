package commands

import (
	"fmt"
	"strings"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func UpdateCache(baseURL, suites, components, architectures, cacheDir string, verbose bool, keyrings, keyringDirs []string, skipGPGVerify bool, localizer *i18n.Localizer) error {
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

	if verbose {
		fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "command.update.start",
			TemplateData: map[string]any{
				"URL":           baseURL,
				"Suites":        strings.Join(suiteList, ","),
				"Components":    strings.Join(componentList, ","),
				"Architectures": strings.Join(architectureList, ","),
				"Dest":          cacheDir,
			},
		}))
	}

	for _, suite := range suiteList {
		repo := debian.NewRepository("cache-"+suite, baseURL, "cache update", suite, componentList, architectureList)
		repo.SetKeyringPathsWithDirs(keyrings, keyringDirs)
		if skipGPGVerify {
			repo.DisableSignatureVerification()
		}

		if err := validateComponentsAndArchitectures(repo, suite, componentList, architectureList, localizer); err != nil {
			return fmt.Errorf("validation failed for suite %s: %w", suite, err)
		}

		if verbose {
			fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "command.update.suite",
				TemplateData: map[string]any{
					"Suite": suite,
				},
			}))
		}

		if err := repo.FetchAndCachePackages(cacheDir); err != nil {
			return fmt.Errorf("failed to update cache for suite %s: %w", suite, err)
		}
	}

	if verbose {
		fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "command.update.success",
			TemplateData: map[string]any{
				"Dest": cacheDir,
			},
		}))
	}

	return nil
}

func splitAndTrim(value string) []string {
	raw := strings.Split(strings.TrimSpace(value), ",")
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
