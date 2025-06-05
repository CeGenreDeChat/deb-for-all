package debian

// Package représente un paquet Debian.
type Package struct {
    Name        string // Nom du paquet
    Version     string // Version du paquet
    Architecture string // Architecture du paquet
    Maintainer  string // Responsable du paquet
    Description string // Description du paquet
}

// NewPackage crée une nouvelle instance de Package.
func NewPackage(name, version, architecture, maintainer, description string) *Package {
    return &Package{
        Name:        name,
        Version:     version,
        Architecture: architecture,
        Maintainer:  maintainer,
        Description: description,
    }
}

// String retourne une représentation sous forme de chaîne du paquet.
func (p *Package) String() string {
    return p.Name + " (" + p.Version + ") - " + p.Description
}