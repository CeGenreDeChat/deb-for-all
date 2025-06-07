# deb-for-all

[![Go Version](https://img.shields.io/badge/go-1.18+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

**deb-for-all** est une bibliothèque Go complète pour la gestion des paquets Debian et la création de miroirs de dépôts. Ce projet fournit à la fois une bibliothèque réutilisable et un binaire en ligne de commande pour faciliter la manipulation efficace des paquets Debian.

## 🚀 Fonctionnalités

### 📦 Gestion des Paquets
- Lecture, écriture et validation de fichiers de contrôle Debian
- Téléchargement de paquets binaires et sources avec suivi de progression
- Vérification de sommes de contrôle et mécanismes de retry
- Téléchargements concurrents pour plusieurs paquets

### 🔄 Miroir de Dépôts
- **Création complète de miroirs** de dépôts Debian
- Support de plusieurs distributions (suites), composants et architectures
- Modes de miroir : métadonnées seulement ou avec paquets complets
- Structure de répertoires conforme aux standards Debian
- Synchronisation incrémentale et vérification d'intégrité

### 🗂️ Gestion des Dépôts
- Interaction avec les dépôts Debian
- Parsing automatique des fichiers Release et Packages
- Gestion des différents formats de compression (.gz, .xz)
- Support des architectures multiples

## 📥 Installation

### Via Go Install
```bash
go install github.com/CeGenreDeChat/deb-for-all/cmd/deb-for-all@latest
```

### Construction Manuelle
```bash
git clone https://github.com/CeGenreDeChat/deb-for-all.git
cd deb-for-all
make build
```

### En tant que Bibliothèque
```bash
go get github.com/CeGenreDeChat/deb-for-all
```

## 🛠️ Utilisation

### Interface en Ligne de Commande

#### Créer un Miroir (Métadonnées seulement)
```bash
deb-for-all -command mirror -dest ./debian-mirror -verbose
```

#### Créer un Miroir Complet avec Paquets
```bash
# ⚠️ ATTENTION: Téléchargement de plusieurs GB
deb-for-all -command mirror -dest ./debian-mirror -download-packages -verbose
```

#### Configuration Personnalisée de Miroir
```bash
deb-for-all -command mirror \
  -url http://deb.debian.org/debian \
  -suites bookworm,bullseye \
  -components main,contrib \
  -architectures amd64,arm64 \
  -dest ./custom-mirror -verbose
```

#### Téléchargement de Paquets Sources
```bash
deb-for-all -command download-source -package hello -version 2.10-2
```

### Utilisation en tant que Bibliothèque

#### Création d'un Miroir Simple
```go
package main

import (
    "log"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    config := debian.MirrorConfig{
        BaseURL:          "http://deb.debian.org/debian",
        Suites:           []string{"bookworm"},
        Components:       []string{"main"},
        Architectures:    []string{"amd64"},
        DownloadPackages: false, // Métadonnées seulement
        Verbose:          true,
    }

    mirror := debian.NewMirror(config, "./my-mirror")

    if err := mirror.Clone(); err != nil {
        log.Fatal(err)
    }
}
```

#### Téléchargement de Paquets avec Progression
```go
package main

import (
    "fmt"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    downloader := debian.NewDownloader()

    // Créer un paquet source
    sourcePackage := debian.NewSourcePackage(
        "hello", "2.10-2",
        "Maintainer <maintainer@example.com>",
        "Hello package",
        "pool/main/h/hello",
    )

    // Ajouter des fichiers
    sourcePackage.AddFile(
        "hello_2.10-2.dsc",
        "http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2.dsc",
        1950, "", "", "dsc",
    )

    // Télécharger avec progression
    err := downloader.DownloadSourcePackageWithProgress(
        sourcePackage, "./downloads",
        func(filename string, downloaded, total int64) {
            if total > 0 {
                percentage := float64(downloaded) / float64(total) * 100
                fmt.Printf("\r%s: %.1f%%", filename, percentage)
            }
        },
    )

    if err != nil {
        panic(err)
    }
}
```

#### Gestion Avancée de Dépôt
```go
package main

import (
    "fmt"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    repo := debian.NewRepository(
        "debian-main",
        "http://deb.debian.org/debian",
        "Debian Main Repository",
        "bookworm",
        []string{"main"},
        []string{"amd64"},
    )

    // Récupérer les informations de release
    if err := repo.FetchReleaseFile(); err != nil {
        panic(err)
    }

    releaseInfo := repo.GetReleaseInfo()
    fmt.Printf("Distribution: %s\n", releaseInfo.Suite)
    fmt.Printf("Date: %s\n", releaseInfo.Date)
}
```

## 🏗️ Structure du Projet

```
deb-for-all/
├── cmd/deb-for-all/           # Binaire CLI principal
├── pkg/debian/                # Bibliothèque principale
│   ├── control.go            # Gestion des fichiers de contrôle
│   ├── downloader.go         # Téléchargement de paquets
│   ├── mirror.go             # Fonctionnalités de miroir
│   ├── package.go            # Types et fonctions de paquets
│   └── repository.go         # Interaction avec dépôts
├── examples/                  # Exemples d'utilisation
│   ├── basic/                # Exemple basique
│   ├── download/             # Exemple téléchargement
│   ├── mirror/               # Exemple miroir interactif
│   └── repository/           # Exemple dépôt
├── internal/                  # Code interne
└── docs/                     # Documentation
```

## 🎯 Exemples Complets

### Exemple Interactif de Miroir
```bash
make mirror-example
# ou
cd examples/mirror && go run main.go
```

### Tests Rapides
```bash
# Test de miroir basique
make test-mirror

# Test de téléchargement
make test-download

# Construction pour toutes les plateformes
make build-all
```

## 📊 Structure de Miroir Debian

Un miroir créé par **deb-for-all** suit la structure standard Debian :

```
debian-mirror/
├── dists/
│   └── bookworm/
│       ├── Release                    # Métadonnées de distribution
│       └── main/
│           ├── binary-amd64/
│           │   └── Packages.gz        # Liste des paquets
│           └── source/
│               └── Sources.gz         # Paquets sources
└── pool/                              # (si DownloadPackages=true)
    └── main/
        └── [a-z]/
            └── package-name/
                └── package.deb
```

## ⚙️ Options de Configuration

### MirrorConfig
```go
type MirrorConfig struct {
    BaseURL          string   // URL du dépôt source
    Suites           []string // Distributions (bookworm, bullseye, etc.)
    Components       []string // Composants (main, contrib, non-free)
    Architectures    []string // Architectures (amd64, arm64, all)
    DownloadPackages bool     // Télécharger les .deb
    Verbose          bool     // Affichage détaillé
}
```

## 🔧 Développement

### Prérequis
- Go 1.18 ou supérieur
- Make (optionnel mais recommandé)

### Construction
```bash
# Construction simple
go build ./cmd/deb-for-all

# Avec Makefile
make build

# Tests
make test
```

### Contribution
1. Fork le projet
2. Créez une branche pour votre fonctionnalité (`git checkout -b feat/nouvelle-fonctionnalite`)
3. Committez vos changements (`git commit -am 'feat: ajout nouvelle fonctionnalité'`)
4. Push vers la branche (`git push origin feat/nouvelle-fonctionnalite`)
5. Créez une Pull Request

## 📝 Standards de Code

Ce projet suit les standards de commits conventionnels :
- `feat:` nouvelles fonctionnalités
- `fix:` corrections de bugs
- `docs:` documentation
- `refactor:` refactoring de code

## 🔗 Cas d'Usage

### Administrateurs Système
- Création de miroirs locaux pour réduire la bande passante
- Synchronisation automatisée de dépôts
- Archivage de versions spécifiques

### Développeurs
- Intégration dans des outils de build
- Téléchargement automatisé de dépendances
- Création d'environnements de test isolés

### DevOps
- Intégration dans des pipelines CI/CD
- Gestion de dépôts personnalisés
- Déploiement d'infrastructures

## 📄 Licence

Ce projet est sous licence MIT. Voir le fichier [LICENSE](LICENSE) pour plus de détails.

## 🤝 Support

- 📧 Créez une issue sur GitHub pour les bugs
- 💡 Proposez des améliorations via les issues
- 📖 Consultez la documentation dans le dossier `docs/`

---

**deb-for-all** - Simplifiant la gestion des paquets Debian pour tous 🐧
        Filename:     "example-package_1.0.0_amd64.deb",
    }

    // Create a downloader
    downloader := debian.NewDownloader()

    // Simple download to directory
    err := downloader.DownloadToDir(pkg, "./downloads")
    if err != nil {
        fmt.Printf("Download failed: %v\n", err)
    }

    // Silent download (no console output)
    err = downloader.DownloadToDirSilent(pkg, "./downloads")
    if err != nil {
        // Handle error quietly
        log.Printf("Silent download failed: %v", err)
    }
}
```

### Advanced Download with Progress

```go
// Create a downloader with custom settings
downloader := debian.NewDownloader()
downloader.RetryAttempts = 3
downloader.VerifyChecksums = true

// Progress callback
progressCallback := func(downloaded, total int64) {
    if total > 0 {
        percentage := float64(downloaded) / float64(total) * 100
        fmt.Printf("\rProgress: %.1f%%", percentage)
    }
}

// Download with progress reporting
err := downloader.DownloadWithProgress(pkg, "./downloads/package.deb", progressCallback)
if err != nil {
    fmt.Printf("Download failed: %v\n", err)
}
```

### Silent Download for Clean Integration

```go
// Perfect for integration into applications without console pollution
func downloadQuietly(packageURL, destDir string) error {
    pkg := &debian.Package{
        Name:        "my-package",
        DownloadURL: packageURL,
    }

    downloader := debian.NewDownloader()
    return downloader.DownloadToDirSilent(pkg, destDir)
}
```

### Repository Usage

```go
// Create a repository
repo := debian.NewRepository(
    "debian-main",
    "http://deb.debian.org/debian",
    "Main Debian Repository",
    "bookworm",                              // Distribution
    []string{"main", "contrib", "non-free"}, // Sections
    []string{"amd64"},                       // Architectures
)

// Check if a package is available
available, err := repo.CheckPackageAvailability("curl", "7.74.0-1.3", "amd64")
if err != nil {
    fmt.Printf("Error: %v\n", err)
} else {
    fmt.Printf("Package available: %v\n", available)
}

// Download from repository
err = repo.DownloadPackage("curl", "7.74.0-1.3", "amd64", "./downloads")
if err != nil {
    fmt.Printf("Download failed: %v\n", err)
}
```

### Complete Package Discovery

```go
// NEW: FetchPackages now collects ALL packages from ALL configured sections
repo := debian.NewRepository("debian-complete", "http://deb.debian.org/debian", "Debian",
    "bookworm", []string{"main", "contrib", "non-free"}, []string{"amd64"})

// This will download and parse Packages files from ALL sections
packages, err := repo.FetchPackages()
if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
}

fmt.Printf("Found %d unique packages across all sections!\n", len(packages))
// Typical result: 80,000+ packages from main + contrib + non-free

// Search for specific packages in the complete list
searchFor := []string{"firefox", "chromium", "docker", "kubernetes"}
for _, search := range searchFor {
    for _, pkg := range packages {
        if pkg == search {
            fmt.Printf("✅ %s available\n", search)
            break
        }
    }
}
```

### Multiple Package Downloads

```go
packages := []*debian.Package{
    {Name: "package1", DownloadURL: "https://example.com/package1.deb"},
    {Name: "package2", DownloadURL: "https://example.com/package2.deb"},
    {Name: "package3", DownloadURL: "https://example.com/package3.deb"},
}

downloader := debian.NewDownloader()
errors := downloader.DownloadMultiple(packages, "./downloads", 5) // Max 5 concurrent downloads

// Handle any errors
for _, err := range errors {
    fmt.Printf("Error: %v\n", err)
}
```

You can find more examples in the `examples/` directory:
- `examples/basic/` - Basic usage example
- `examples/download/` - Real download examples with Debian packages

## Migration Guide (v2.0.0)

**⚠️ BREAKING CHANGES in v2.0.0**

This version introduces a major architectural change that improves code organization by following the Single Responsibility Principle.

### What Changed

All download methods have been **removed** from the `Package` struct and **centralized** in the `Downloader` struct:

- ❌ `pkg.Download()` - **REMOVED**
- ❌ `pkg.DownloadSilent()` - **REMOVED**
- ❌ `pkg.DownloadToFile()` - **REMOVED**
- ❌ `pkg.DownloadToFileSilent()` - **REMOVED**

### Migration Steps

**Before (v1.x):**
```go
pkg := &debian.Package{...}
err := pkg.Download("./downloads")           // Old API
err := pkg.DownloadSilent("./downloads")     // Old API
```

**After (v2.0.0+):**
```go
pkg := &debian.Package{...}
downloader := debian.NewDownloader()
err := downloader.DownloadToDir(pkg, "./downloads")      // New API
err := downloader.DownloadToDirSilent(pkg, "./downloads") // New API
```

This change provides:
- ✅ Clear separation of concerns
- ✅ Better testability
- ✅ Centralized download configuration
- ✅ No more code duplication

For detailed migration information, see [REFACTORING_SUMMARY.md](docs/REFACTORING_SUMMARY.md).

## Command-Line Tool

The project also includes a command-line tool. To run the tool, navigate to the `cmd/deb-for-all` directory and execute:

```bash
go run main.go
```

## Documentation

API documentation is available in the `docs/api.md` file. This documentation provides detailed information about the functions and types exported by the library.

## Contributing

Contributions are welcome! Please read the [CONTRIBUTING.md](CONTRIBUTING.md) file for guidelines on how to contribute to this project.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.