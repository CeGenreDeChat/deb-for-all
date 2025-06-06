package debian

import (
	"errors"
	"os"
	"strings"
)

type Control struct {
	// Required fields
	Package      string
	Version      string
	Architecture string
	Maintainer   string
	Description  string

	// Optional fields
	Source        string
	Section       string
	Priority      string
	Essential     string
	Depends       []string
	PreDepends    []string
	Recommends    []string
	Suggests      []string
	Enhances      []string
	Breaks        []string
	Conflicts     []string
	Provides      []string
	Replaces      []string
	InstalledSize string
	Homepage      string
	BuiltUsing    string
	PackageType   string

	// Maintainer script fields
	Preinst  string
	Postinst string
	Prerm    string
	Postrm   string

	// Multi-arch support
	MultiArch string

	// Origin and distribution
	Origin string
	Bugs   string

	// Custom fields (X- prefixed)
	CustomFields map[string]string
}

func ReadControl(filePath string) (*Control, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return parseControl(string(data))
}

func WriteControl(filePath string, control *Control) error {
	content := formatControl(control)
	return os.WriteFile(filePath, []byte(content), os.ModePerm)
}

func parseControl(content string) (*Control, error) {
	lines := strings.Split(content, "\n")
	control := &Control{
		CustomFields: make(map[string]string),
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			continue
		}

		field := strings.TrimSpace(line[:colonIndex])
		value := strings.TrimSpace(line[colonIndex+1:])

		switch strings.ToLower(field) {
		case "package":
			control.Package = value
		case "version":
			control.Version = value
		case "architecture":
			control.Architecture = value
		case "maintainer":
			control.Maintainer = value
		case "description":
			control.Description = value
		case "source":
			control.Source = value
		case "section":
			control.Section = value
		case "priority":
			control.Priority = value
		case "essential":
			control.Essential = value
		case "depends":
			control.Depends = parsePackageList(value)
		case "pre-depends":
			control.PreDepends = parsePackageList(value)
		case "recommends":
			control.Recommends = parsePackageList(value)
		case "suggests":
			control.Suggests = parsePackageList(value)
		case "enhances":
			control.Enhances = parsePackageList(value)
		case "breaks":
			control.Breaks = parsePackageList(value)
		case "conflicts":
			control.Conflicts = parsePackageList(value)
		case "provides":
			control.Provides = parsePackageList(value)
		case "replaces":
			control.Replaces = parsePackageList(value)
		case "installed-size":
			control.InstalledSize = value
		case "homepage":
			control.Homepage = value
		case "built-using":
			control.BuiltUsing = value
		case "package-type":
			control.PackageType = value
		case "multi-arch":
			control.MultiArch = value
		case "origin":
			control.Origin = value
		case "bugs":
			control.Bugs = value
		default:
			// Handle custom fields (X- prefixed or unknown fields)
			control.CustomFields[field] = value
		}
	}

	if control.Package == "" || control.Version == "" || control.Architecture == "" || control.Maintainer == "" {
		return nil, errors.New("invalid control file: missing required fields (Package, Version, Architecture, Maintainer)")
	}

	return control, nil
}

func parsePackageList(value string) []string {
	if value == "" {
		return nil
	}

	packages := strings.Split(value, ",")
	for i := range packages {
		packages[i] = strings.TrimSpace(packages[i])
	}
	return packages
}

func formatControl(control *Control) string {
	var sb strings.Builder

	// Required fields
	sb.WriteString("Package: " + control.Package + "\n")
	sb.WriteString("Version: " + control.Version + "\n")
	sb.WriteString("Architecture: " + control.Architecture + "\n")
	sb.WriteString("Maintainer: " + control.Maintainer + "\n")

	// Optional fields
	if control.Source != "" {
		sb.WriteString("Source: " + control.Source + "\n")
	}
	if control.Section != "" {
		sb.WriteString("Section: " + control.Section + "\n")
	}
	if control.Priority != "" {
		sb.WriteString("Priority: " + control.Priority + "\n")
	}
	if control.Essential != "" {
		sb.WriteString("Essential: " + control.Essential + "\n")
	}

	// Package relationships
	if len(control.Depends) > 0 {
		sb.WriteString("Depends: " + strings.Join(control.Depends, ", ") + "\n")
	}
	if len(control.PreDepends) > 0 {
		sb.WriteString("Pre-Depends: " + strings.Join(control.PreDepends, ", ") + "\n")
	}
	if len(control.Recommends) > 0 {
		sb.WriteString("Recommends: " + strings.Join(control.Recommends, ", ") + "\n")
	}
	if len(control.Suggests) > 0 {
		sb.WriteString("Suggests: " + strings.Join(control.Suggests, ", ") + "\n")
	}
	if len(control.Enhances) > 0 {
		sb.WriteString("Enhances: " + strings.Join(control.Enhances, ", ") + "\n")
	}
	if len(control.Breaks) > 0 {
		sb.WriteString("Breaks: " + strings.Join(control.Breaks, ", ") + "\n")
	}
	if len(control.Conflicts) > 0 {
		sb.WriteString("Conflicts: " + strings.Join(control.Conflicts, ", ") + "\n")
	}
	if len(control.Provides) > 0 {
		sb.WriteString("Provides: " + strings.Join(control.Provides, ", ") + "\n")
	}
	if len(control.Replaces) > 0 {
		sb.WriteString("Replaces: " + strings.Join(control.Replaces, ", ") + "\n")
	}

	// Other optional fields
	if control.InstalledSize != "" {
		sb.WriteString("Installed-Size: " + control.InstalledSize + "\n")
	}
	if control.Homepage != "" {
		sb.WriteString("Homepage: " + control.Homepage + "\n")
	}
	if control.BuiltUsing != "" {
		sb.WriteString("Built-Using: " + control.BuiltUsing + "\n")
	}
	if control.PackageType != "" {
		sb.WriteString("Package-Type: " + control.PackageType + "\n")
	}
	if control.MultiArch != "" {
		sb.WriteString("Multi-Arch: " + control.MultiArch + "\n")
	}
	if control.Origin != "" {
		sb.WriteString("Origin: " + control.Origin + "\n")
	}
	if control.Bugs != "" {
		sb.WriteString("Bugs: " + control.Bugs + "\n")
	}

	// Custom fields
	for field, value := range control.CustomFields {
		sb.WriteString(field + ": " + value + "\n")
	}

	// Description comes last
	sb.WriteString("Description: " + control.Description + "\n")

	return sb.String()
}
