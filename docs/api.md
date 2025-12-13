# Documentation API - deb-for-all

## Vue d'ensemble

Cette documentation fournit un aperçu complet de l'API de la bibliothèque de gestion des paquets Debian. Elle décrit les types, structures et fonctions disponibles pour gérer les paquets Debian, créer des miroirs de dépôts et interagir avec les dépôts.

## Table des Matières

- [Types Principaux](#types-principaux)
- [Module Mirror](#module-mirror)
- [Module Repository](#module-repository)
- [Module Downloader](#module-downloader)
- [Module Package](#module-package)
- [Module Control](#module-control)
- [Exemples d'Utilisation](#exemples-dutilisation)

## Types Principaux

### Package

```go
type Package struct {
    Name         string // Nom du paquet
    Version      string // Version du paquet
    Architecture string // Architecture du paquet
    Maintainer   string // Responsable du paquet
    Description  string // Description du paquet
    DownloadURL  string // URL de téléchargement du paquet
    Filename     string // Nom du fichier .deb
    Size         int64  // Taille du paquet en bytes
    MD5sum       string // Somme de contrôle MD5
    SHA1         string // Somme de contrôle SHA1
    SHA256       string // Somme de contrôle SHA256
}
```

### SourcePackage

```go
type SourcePackage struct {
    Name        string           // Nom du paquet source
    Version     string           // Version
    Maintainer  string           // Responsable
    Description string           // Description
    Directory   string           // Répertoire dans le pool
    Files       []SourcePackageFile // Fichiers associés
}

type SourcePackageFile struct {
    Name string // Nom du fichier
    URL  string // URL de téléchargement
    Size int64  // Taille en bytes
    MD5  string // Somme MD5
    SHA1 string // Somme SHA1
    SHA256 string // Somme SHA256
    Type string // Type de fichier (dsc, orig, debian)
}
```

### ControlFile

```go
type ControlFile struct {
    Package      string            // Nom du paquet
    Version      string            // Version
    Maintainer   string            // Responsable
    Architecture string            // Architecture
    Depends      []string          // Dépendances
    Description  string            // Description
    Section      string            // Section
    Priority     string            // Priorité
    Homepage     string            // Page d'accueil
    Fields       map[string]string // Champs supplémentaires
}
```

## Module Mirror

### MirrorConfig

```go
type MirrorConfig struct {
    BaseURL          string   // URL du dépôt à mettre en miroir
    Suites           []string // Suites (distributions) à mettre en miroir
    Components       []string // Composants du dépôt (main, contrib, non-free)
    Architectures    []string // Architectures cibles (amd64, arm64, all)
    DownloadPackages bool     // Télécharger les paquets .deb
    Verbose          bool     // Affichage verbeux
}
```

**Méthodes :**

```go
func (c *MirrorConfig) Validate() error
```
Valide la configuration du miroir.

### Mirror

```go
type Mirror struct {
    // Champs privés
}
```

**Constructeur :**

```go
func NewMirror(config MirrorConfig, basePath string) *Mirror
```
Crée une nouvelle instance de Mirror.

**Méthodes principales :**

```go
func (m *Mirror) Clone() error
```
Crée un miroir complet du dépôt configuré.

```go
func (m *Mirror) Sync() error
```
Effectue une synchronisation incrémentale du miroir.

```go
func (m *Mirror) GetMirrorInfo() map[string]interface{}
```
Retourne les informations de configuration du miroir.

```go
func (m *Mirror) GetMirrorStatus() (map[string]interface{}, error)
```
Retourne le statut actuel du miroir (nombre de fichiers, taille, etc.).

```go
func (m *Mirror) EstimateMirrorSize() (int64, error)
```
Estime la taille totale des paquets à télécharger.

```go
func (m *Mirror) GetRepositoryInfo() *Repository
```
Retourne l'instance Repository sous-jacente.

```go
func (m *Mirror) UpdateConfiguration(config MirrorConfig) error
```
Met à jour la configuration du miroir.

```go
func (m *Mirror) VerifyMirrorIntegrity(suite string) error
```
Vérifie l'intégrité des fichiers téléchargés.

## Module Repository

### Repository

```go
type Repository struct {
    ID           string   // Identifiant unique
    URL          string   // URL du dépôt
    Name         string   // Nom du dépôt
    Distribution string   // Distribution (bookworm, bullseye)
    Sections     []string // Sections (main, contrib, non-free)
    Architectures []string // Architectures supportées
}
```

**Constructeur :**

```go
func NewRepository(id, url, name, distribution string, sections, architectures []string) *Repository
```

**Méthodes :**

```go
func (r *Repository) FetchReleaseFile() error
```
Récupère et parse le fichier Release du dépôt.

```go
func (r *Repository) GetReleaseInfo() *ReleaseFile
```
Retourne les informations du fichier Release.

```go
func (r *Repository) SetDistribution(distribution string)
```
Change la distribution du dépôt.

```go
func (r *Repository) SetSections(sections []string)
```
Définit les sections du dépôt.

```go
func (r *Repository) SetArchitectures(architectures []string)
```
Définit les architectures supportées.

```go
func (r *Repository) SearchPackage(packageName string) ([]*Package, error)
```
Recherche un paquet dans le dépôt.

```go
func (r *Repository) GetPackageInfo(packageName, version, architecture string) (*Package, error)
```
Récupère les informations détaillées d'un paquet.

### ReleaseFile

```go
type ReleaseFile struct {
    Origin        string            // Origine du dépôt
    Label         string            // Label
    Suite         string            // Suite (nom de code)
    Codename      string            // Nom de code
    Date          string            // Date de publication
    Description   string            // Description
    Architectures []string          // Architectures disponibles
    Components    []string          // Composants disponibles
    MD5Sum        []FileHash        // Sommes MD5
    SHA1          []FileHash        // Sommes SHA1
    SHA256        []FileHash        // Sommes SHA256
}

type FileHash struct {
    Hash     string // Somme de contrôle
    Size     int64  // Taille du fichier
    Filename string // Nom du fichier
}
```

## Module Downloader

### Downloader

```go
type Downloader struct {
    // Configuration interne
}
```

**Constructeur :**

```go
func NewDownloader() *Downloader
```

**Méthodes pour paquets binaires :**

```go
func (d *Downloader) DownloadPackage(pkg *Package, destDir string) error
```
Télécharge un paquet binaire.

```go
func (d *Downloader) DownloadPackageWithProgress(pkg *Package, destDir string, progressCallback func(downloaded, total int64)) error
```
Télécharge avec suivi de progression.

**Méthodes pour paquets sources :**

```go
func (d *Downloader) DownloadSourcePackage(srcPkg *SourcePackage, destDir string) error
```
Télécharge tous les fichiers d'un paquet source.

```go
func (d *Downloader) DownloadSourcePackageWithProgress(srcPkg *SourcePackage, destDir string, progressCallback func(filename string, downloaded, total int64)) error
```
Télécharge avec progression détaillée.

```go
func (d *Downloader) DownloadSourcePackageSilent(srcPkg *SourcePackage, destDir string) error
```
Télécharge en mode silencieux.

```go
func (d *Downloader) DownloadOrigTarball(srcPkg *SourcePackage, destDir string) error
```
Télécharge uniquement le tarball original.

**Méthodes pour fichiers génériques :**

```go
func (d *Downloader) DownloadFile(url, destPath string) error
```
Télécharge un fichier depuis une URL.

```go
func (d *Downloader) DownloadFileWithProgress(url, destPath string, progressCallback func(downloaded, total int64)) error
```
Télécharge avec progression.

## Module Package

### Fonctions utilitaires

```go
func NewSourcePackage(name, version, maintainer, description, directory string) *SourcePackage
```
Crée un nouveau paquet source.

```go
func (sp *SourcePackage) AddFile(name, url string, size int64, md5, sha1, sha256, fileType string)
```
Ajoute un fichier au paquet source.

```go
func (sp *SourcePackage) GetFilesByType(fileType string) []SourcePackageFile
```
Récupère les fichiers par type (dsc, orig, debian).

```go
func (sp *SourcePackage) GetTotalSize() int64
```
Calcule la taille totale du paquet source.

## Module Control

### Fonctions de manipulation

```go
func ParseControlFile(content string) (*ControlFile, error)
```
Parse le contenu d'un fichier de contrôle.

```go
func (cf *ControlFile) String() string
```
Convertit le fichier de contrôle en chaîne.

```go
func (cf *ControlFile) Validate() error
```
Valide les champs obligatoires.

```go
func (cf *ControlFile) AddField(key, value string)
```
Ajoute un champ personnalisé.

```go
func (cf *ControlFile) GetField(key string) string
```
Récupère la valeur d'un champ.

## Exemples d'Utilisation

### Exemple 1: Création d'un miroir simple

```go
config := debian.MirrorConfig{
    BaseURL:          "http://deb.debian.org/debian",
    Suites:           []string{"bookworm"},
    Components:       []string{"main"},
    Architectures:    []string{"amd64"},
    DownloadPackages: false,
    Verbose:          true,
}

mirror := debian.NewMirror(config, "./debian-mirror")
if err := mirror.Clone(); err != nil {
    log.Fatal(err)
}
```

### Exemple 2: Téléchargement avec progression

```go
downloader := debian.NewDownloader()

pkg := &debian.Package{
    Name:        "hello",
    DownloadURL: "http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2_amd64.deb",
    Filename:    "hello_2.10-2_amd64.deb",
}

err := downloader.DownloadPackageWithProgress(pkg, "./downloads",
    func(downloaded, total int64) {
        if total > 0 {
            fmt.Printf("\rProgress: %.2f%%", float64(downloaded)/float64(total)*100)
        }
    })
```

### Exemple 3: Recherche dans un dépôt

```go
repo := debian.NewRepository(
    "debian",
    "http://deb.debian.org/debian",
    "Debian Repository",
    "bookworm",
    []string{"main"},
    []string{"amd64"},
)

err := repo.FetchReleaseFile()
if err != nil {
    log.Fatal(err)
}

packages, err := repo.SearchPackage("hello")
if err != nil {
    log.Fatal(err)
}

for _, pkg := range packages {
    fmt.Printf("Package: %s, Version: %s\n", pkg.Name, pkg.Version)
}
```

### Exemple 4: Validation de configuration

```go
config := debian.MirrorConfig{
    BaseURL:       "http://deb.debian.org/debian",
    Suites:        []string{"bookworm"},
    Components:    []string{"main"},
    Architectures: []string{"amd64"},
}

if err := config.Validate(); err != nil {
    log.Fatalf("Configuration invalide: %v", err)
}
```

## Types de Retour et Gestion d'Erreurs

### Codes d'Erreur Communs

- **Erreurs de réseau** : Problèmes de connectivité, timeouts
- **Erreurs de parsing** : Fichiers Release ou Packages malformés
- **Erreurs de validation** : Configuration ou paramètres invalides
- **Erreurs de système de fichiers** : Permissions, espace disque
- **Erreurs de sommes de contrôle** : Fichiers corrompus

### Bonnes Pratiques

1. **Toujours vérifier les erreurs** retournées par les fonctions
2. **Utiliser la validation** avant de commencer des opérations coûteuses
3. **Implémenter une gestion de retry** pour les opérations réseau
4. **Vérifier l'espace disque** avant de créer des miroirs complets
5. **Utiliser le mode verbeux** pour le debugging

## Performance et Optimisation

### Conseils pour les Miroirs

- Utilisez `DownloadPackages: false` pour ne télécharger que les métadonnées
- Limitez les architectures aux besoins réels
- Utilisez la synchronisation incrémentale avec `Sync()`
- Surveillez l'espace disque disponible

### Conseils pour les Téléchargements

- Utilisez les callbacks de progression pour les gros fichiers
- Implémentez une logique de retry pour la robustesse
- Vérifiez les sommes de contrôle après téléchargement
- Utilisez les téléchargements concurrents avec prudence

Cette documentation couvre l'ensemble de l'API disponible dans **deb-for-all**. Pour des exemples plus détaillés, consultez le dossier `examples/` du projet.
    Description string
}
```

### Repository

```go
type Repository struct {
    Name        string // Nom du dépôt
    URL         string // URL du dépôt
    Description string // Description du dépôt
}
```

### Downloader

```go
type Downloader struct {
    UserAgent      string
    Timeout        time.Duration
    RetryAttempts  int
    VerifyChecksums bool
}
```

### DownloadInfo

```go
type DownloadInfo struct {
    URL           string
    ContentLength int64
    ContentType   string
    LastModified  string
}
```

## Functions

### ManagePackages

```go
func ManagePackages(pkg Package) error
```
Not provided: deb-for-all is scoped to downloading and mirroring packages only and does not perform installation, removal, or upgrade operations.

### ReadControlFile

```go
func ReadControlFile(path string) (ControlFile, error)
```
This function reads a Debian control file from the specified path and returns a ControlFile struct.

### WriteControlFile

```go
func WriteControlFile(path string, control ControlFile) error
```
This function writes a ControlFile struct to the specified path.

### SearchPackage

```go
func SearchPackage(name string) ([]Package, error)
```
This function searches for packages by name and returns a list of matching packages.

### Package Methods

Package struct contains only metadata. All download functionality has been moved to Downloader for better separation of concerns.

#### GetDownloadInfo

```go
func (p *Package) GetDownloadInfo() (*DownloadInfo, error)
```
Retrieves download information without downloading the file.

### Repository Methods

#### DownloadPackage

```go
func (r *Repository) DownloadPackage(packageName, version, architecture, destDir string) error
```
Downloads a package from the repository.

#### DownloadPackageByURL

```go
func (r *Repository) DownloadPackageByURL(packageURL, destDir string) error
```
Downloads a package from a specific URL.

#### CheckPackageAvailability

```go
func (r *Repository) CheckPackageAvailability(packageName, version, architecture string) (bool, error)
```
Checks if a package is available in the repository.

#### FetchPackages

```go
func (r *Repository) FetchPackages() ([]string, error)
```
**⚠️ CHANGEMENT MAJEUR**: Cette méthode collecte maintenant TOUS les paquets disponibles depuis TOUTES les sections et architectures configurées dans le repository, au lieu de s'arrêter au premier succès.

**Comportement**:
- Parcourt toutes les distributions configurées (avec fallback sur bookworm, bullseye, buster)
- Télécharge les fichiers Packages de TOUTES les sections spécifiées
- Traite TOUTES les architectures configurées
- Déduplique automatiquement les noms de paquets
- Supporte les formats non-compressé, .gz, et .xz
- Retourne la liste complète des paquets uniques trouvés

**Utilisation recommandée**:
```go
repo := debian.NewRepository("debian", "http://deb.debian.org/debian", "Debian",
    "bookworm", []string{"main", "contrib"}, []string{"amd64"})

// Récupère TOUS les paquets de main ET contrib pour amd64
packages, err := repo.FetchPackages()
// Résultat typique: 80,000+ paquets uniques
```

### Downloader Methods

#### NewDownloader

```go
func NewDownloader() *Downloader
```
Creates a new Downloader instance with default settings.

#### DownloadWithProgress

```go
func (d *Downloader) DownloadWithProgress(pkg *Package, destPath string, progressCallback func(downloaded, total int64)) error
```
Downloads a package with progress reporting.

#### DownloadWithChecksum

```go
func (d *Downloader) DownloadWithChecksum(pkg *Package, destPath, checksum, checksumType string) error
```
Downloads a package and verifies its checksum.

#### DownloadMultiple

```go
func (d *Downloader) DownloadMultiple(packages []*Package, destDir string, maxConcurrent int) []error
```
Downloads multiple packages concurrently.

#### GetFileSize

```go
func (d *Downloader) GetFileSize(url string) (int64, error)
```
Gets the file size from a URL without downloading.

#### DownloadSilent

```go
func (d *Downloader) DownloadSilent(pkg *Package, destPath string) error
```
Downloads a package silently without any console output or progress reporting. Ideal for integration into Go code without polluting the output.

#### DownloadToDir

```go
func (d *Downloader) DownloadToDir(pkg *Package, destDir string) error
```
Downloads a package to a directory with automatic filename generation based on package metadata.

#### DownloadToDirSilent

```go
func (d *Downloader) DownloadToDirSilent(pkg *Package, destDir string) error
```
Downloads a package to a directory silently with automatic filename generation. No console output.

## Error Handling

The library defines custom error types to handle specific errors related to package management and downloading. These errors can be used to provide more context when an operation fails.

## Usage Examples

### Basic Package Download

```go
package main

import (
    "fmt"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    pkg := &debian.Package{
        Name:         "example-package",
        Version:      "1.0.0",
        Architecture: "amd64",
        DownloadURL:  "https://example.com/package.deb",
        Filename:     "example-package_1.0.0_amd64.deb",
    }

    // Use Downloader for all download operations
    downloader := debian.NewDownloader()
    err := downloader.DownloadToDir(pkg, "./downloads")
    if err != nil {
        fmt.Printf("Download failed: %v\n", err)
    }
}
```

### Advanced Download with Progress

```go
downloader := debian.NewDownloader()
downloader.RetryAttempts = 3

progressCallback := func(downloaded, total int64) {
    percentage := float64(downloaded) / float64(total) * 100
    fmt.Printf("\rProgress: %.1f%%", percentage)
}

err := downloader.DownloadWithProgress(pkg, "./downloads/package.deb", progressCallback)
if err != nil {
    fmt.Printf("Download failed: %v\n", err)
}
```

### Repository Usage

```go
repo := debian.NewRepository(
    "debian-main",
    "http://deb.debian.org/debian",
    "Main Debian Repository",
    "bookworm",                              // Distribution
    []string{"main", "contrib", "non-free"}, // Sections
    []string{"amd64"},                       // Architectures
)

// Check availability
available, err := repo.CheckPackageAvailability("curl", "7.74.0-1.3", "amd64")
if err != nil {
    fmt.Printf("Error checking availability: %v\n", err)
}

// Download from repository
err = repo.DownloadPackage("curl", "7.74.0-1.3", "amd64", "./downloads")
if err != nil {
    fmt.Printf("Download failed: %v\n", err)
}
```

### Multiple Package Download

```go
packages := []*debian.Package{
    {Name: "package1", DownloadURL: "https://example.com/package1.deb"},
    {Name: "package2", DownloadURL: "https://example.com/package2.deb"},
}

downloader := debian.NewDownloader()
errors := downloader.DownloadMultiple(packages, "./downloads", 5)
for _, err := range errors {
    fmt.Printf("Error: %v\n", err)
}
```

### Silent Download for Integration

```go
package main

import (
    "log"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func downloadPackageQuietly(name, url, destDir string) error {
    pkg := &debian.Package{
        Name:        name,
        DownloadURL: url,
    }

    // Silent download without any console output
    return pkg.DownloadSilent(destDir)
}

func main() {
    // Perfect for integration into business logic
    err := downloadPackageQuietly("hello",
        "http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2_amd64.deb",
        "./downloads")
    if err != nil {
        log.Printf("Failed to download package: %v", err)
        return
    }

    // Continue with your business logic...
    log.Println("Package downloaded successfully")
}
```

### Silent Download with Downloader

```go
downloader := debian.NewDownloader()
downloader.RetryAttempts = 3
downloader.VerifyChecksums = false

pkg := &debian.Package{
    Name:        "example",
    DownloadURL: "https://example.com/package.deb",
}

// Silent download with retry logic but no console output
err := downloader.DownloadSilent(pkg, "./downloads/package.deb")
if err != nil {
    // Handle error silently or log it
    log.Printf("Download failed: %v", err)
}
```

## Conclusion

This API provides a comprehensive set of functions and types for managing and downloading Debian packages. The library supports various download scenarios including basic downloads, progress reporting, checksum verification, concurrent downloads, and repository-based downloads. For more detailed usage and examples, please refer to the documentation in the `examples` directory.