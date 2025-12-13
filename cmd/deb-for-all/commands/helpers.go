package commands

import "github.com/CeGenreDeChat/deb-for-all/pkg/debian"

// packageFilename returns the expected .deb filename for a package, honoring metadata when present.
func packageFilename(pkg *debian.Package) string {
	if pkg.Filename != "" {
		return pkg.Filename
	}
	return pkg.Name + "_" + pkg.Version + "_" + pkg.Architecture + ".deb"
}
