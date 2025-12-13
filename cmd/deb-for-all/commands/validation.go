package commands

import (
	"fmt"
	"strings"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func validateComponentsAndArchitectures(repo *debian.Repository, suite string, components, architectures []string, localizer *i18n.Localizer) error {
	if err := ensureReleaseInfo(repo, localizer); err != nil {
		return fmt.Errorf("suite %s: %w", suite, err)
	}

	info := repo.GetReleaseInfo()
	unknownComponents := findUnknownValues(components, info.Components)
	unknownArchitectures := findUnknownValues(architectures, info.Architectures)

	var parts []string
	if len(unknownComponents) > 0 {
		parts = append(parts, localizeValidation(localizer, "error.validation.unknown_components", fmt.Sprintf("unknown components: %s (available: %s)", strings.Join(unknownComponents, ", "), strings.Join(info.Components, ", ")), map[string]any{
			"Unknown":   strings.Join(unknownComponents, ", "),
			"Available": strings.Join(info.Components, ", "),
		}))
	}
	if len(unknownArchitectures) > 0 {
		parts = append(parts, localizeValidation(localizer, "error.validation.unknown_architectures", fmt.Sprintf("unknown architectures: %s (available: %s)", strings.Join(unknownArchitectures, ", "), strings.Join(info.Architectures, ", ")), map[string]any{
			"Unknown":   strings.Join(unknownArchitectures, ", "),
			"Available": strings.Join(info.Architectures, ", "),
		}))
	}

	if len(parts) > 0 {
		return fmt.Errorf("%s", strings.Join(parts, " ; "))
	}

	return nil
}

func ensureReleaseInfo(repo *debian.Repository, localizer *i18n.Localizer) error {
	if repo.GetReleaseInfo() != nil {
		return nil
	}

	if err := repo.FetchReleaseFile(); err != nil {
		msg := localizeValidation(localizer, "error.validation.fetch_release", "failed to fetch Release file", nil)
		return fmt.Errorf("%s: %w", msg, err)
	}

	if repo.GetReleaseInfo() == nil {
		msg := localizeValidation(localizer, "error.validation.release_unavailable", "Release information unavailable for validation", nil)
		return fmt.Errorf("%s", msg)
	}

	return nil
}

func localizeValidation(localizer *i18n.Localizer, messageID, fallback string, data map[string]any) string {
	if localizer == nil {
		return fallback
	}

	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: messageID, TemplateData: data})
	if err == nil && msg != "" {
		return msg
	}

	return fallback
}

func findUnknownValues(values, allowed []string) []string {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, item := range allowed {
		key := strings.ToLower(strings.TrimSpace(item))
		if key != "" {
			allowedSet[key] = struct{}{}
		}
	}

	var unknown []string
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		if _, ok := allowedSet[key]; !ok {
			unknown = append(unknown, value)
		}
	}

	return unknown
}
