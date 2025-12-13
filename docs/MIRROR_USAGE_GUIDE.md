# Guide d'Utilisation du Module Mirror - deb-for-all

## Introduction

Le module Mirror de **deb-for-all** permet de créer des miroirs locaux de dépôts Debian. Ce guide vous accompagne dans l'utilisation efficace de cette fonctionnalité.

## 🎯 Cas d'Usage

### 1. Administrateur Système
- Réduire la bande passante internet
- Accélérer les téléchargements de paquets pour les postes internes
- Créer des dépôts offline pour des environnements isolés

### 2. Développeur/DevOps
- Environnements de build reproductibles
- Pipelines CI/CD avec dépendances figées
- Tests avec versions spécifiques de paquets

### 3. Organisations
- Conformité et contrôle des paquets
- Audit et traçabilité
- Redondance et haute disponibilité

## 🚀 Démarrage Rapide

### Installation
```bash
go install github.com/CeGenreDeChat/deb-for-all/cmd/deb-for-all@latest
```

### Premier Miroir (métadonnées seulement)
```bash
deb-for-all -command mirror -dest ./my-mirror -verbose
```

## 📋 Configuration Détaillée

### Structure MirrorConfig

```go
type MirrorConfig struct {
    BaseURL          string   // URL source du dépôt
    Suites           []string // Distributions cibles
    Components       []string // Composants du dépôt
    Architectures    []string // Architectures supportées
    DownloadPackages bool     // Inclure les fichiers .deb
    Verbose          bool     // Affichage détaillé
}
```

### Exemples de Configuration

#### Configuration Minimale
```go
config := debian.MirrorConfig{
    BaseURL:       "http://deb.debian.org/debian",
    Suites:        []string{"bookworm"},
    Components:    []string{"main"},
    Architectures: []string{"amd64"},
    DownloadPackages: false,
    Verbose:       true,
}
```

#### Configuration Multi-Distribution
```go
config := debian.MirrorConfig{
    BaseURL:       "http://deb.debian.org/debian",
    Suites:        []string{"bookworm", "bullseye", "buster"},
    Components:    []string{"main", "contrib", "non-free"},
    Architectures: []string{"amd64", "arm64", "i386"},
    DownloadPackages: false,
    Verbose:       true,
}
```

#### Configuration avec Paquets Complets
```go
config := debian.MirrorConfig{
    BaseURL:       "http://deb.debian.org/debian",
    Suites:        []string{"bookworm"},
    Components:    []string{"main"},
    Architectures: []string{"amd64"},
    DownloadPackages: true, // ⚠️ Téléchargement massif
    Verbose:       true,
}
```

## 🛠️ Utilisation en Ligne de Commande

### Commandes de Base

#### Miroir Standard
```bash
deb-for-all -command mirror \
  -dest ./debian-mirror \
  -verbose
```

#### Miroir Personnalisé
```bash
deb-for-all -command mirror \
  -url http://deb.debian.org/debian \
  -suites bookworm,bullseye \
  -components main,contrib \
  -architectures amd64,arm64 \
  -dest ./custom-mirror \
  -verbose
```

#### Miroir avec Paquets
```bash
# ⚠️ ATTENTION: Téléchargement de plusieurs GB
deb-for-all -command mirror \
  -dest ./full-mirror \
  -download-packages \
  -verbose
```

#### Miroir de Sécurité
```bash
deb-for-all -command mirror \
  -url http://security.debian.org/debian-security \
  -suites bookworm-security \
  -components main \
  -dest ./security-mirror \
  -verbose
```

### Options Avancées

#### Mode Silencieux
```bash
deb-for-all -command mirror -dest ./mirror
```

#### Architectures Multiples
```bash
deb-for-all -command mirror \
  -architectures amd64,arm64,i386 \
  -dest ./multi-arch-mirror \
  -verbose
```

#### Suites Multiples
```bash
deb-for-all -command mirror \
  -suites bookworm,bookworm-updates,bookworm-backports \
  -dest ./complete-mirror \
  -verbose
```

## 🔧 Utilisation Programmatique

### Exemple Basique

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
        DownloadPackages: false,
        Verbose:          true,
    }

    // Validation
    if err := config.Validate(); err != nil {
        log.Fatalf("Configuration invalide: %v", err)
    }

    // Création du miroir
    mirror := debian.NewMirror(config, "./my-mirror")

    // Clonage
    if err := mirror.Clone(); err != nil {
        log.Fatalf("Erreur lors du clonage: %v", err)
    }

    log.Println("Miroir créé avec succès!")
}
```

### Exemple avec Monitoring

```go
package main

import (
    "fmt"
    "log"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    config := debian.MirrorConfig{
        BaseURL:       "http://deb.debian.org/debian",
        Suites:        []string{"bookworm"},
        Components:    []string{"main"},
        Architectures: []string{"amd64"},
        DownloadPackages: false,
        Verbose:       true,
    }

    mirror := debian.NewMirror(config, "./monitored-mirror")

    // Statut initial
    fmt.Println("=== Statut Initial ===")
    status, err := mirror.GetMirrorStatus()
    if err != nil {
        log.Printf("Erreur statut: %v", err)
    } else {
        printStatus(status)
    }

    // Estimation de taille
    fmt.Println("\n=== Estimation ===")
    size, err := mirror.EstimateMirrorSize()
    if err != nil {
        log.Printf("Erreur estimation: %v", err)
    } else {
        fmt.Printf("Taille estimée: %.2f MB\n", float64(size)/1024/1024)
    }

    // Clonage
    fmt.Println("\n=== Démarrage du Clonage ===")
    if err := mirror.Clone(); err != nil {
        log.Fatalf("Erreur clonage: %v", err)
    }

    // Statut final
    fmt.Println("\n=== Statut Final ===")
    status, err = mirror.GetMirrorStatus()
    if err != nil {
        log.Printf("Erreur statut final: %v", err)
    } else {
        printStatus(status)
    }
}

func printStatus(status map[string]interface{}) {
    for key, value := range status {
        fmt.Printf("%s: %v\n", key, value)
    }
}
```

### Exemple de Synchronisation

```go
package main

import (
    "log"
    "time"
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    config := debian.MirrorConfig{
        BaseURL:       "http://deb.debian.org/debian",
        Suites:        []string{"bookworm"},
        Components:    []string{"main"},
        Architectures: []string{"amd64"},
        DownloadPackages: false,
        Verbose:       true,
    }

    mirror := debian.NewMirror(config, "./sync-mirror")

    // Clonage initial
    log.Println("Clonage initial...")
    if err := mirror.Clone(); err != nil {
        log.Fatalf("Erreur clonage: %v", err)
    }

    // Synchronisation périodique
    ticker := time.NewTicker(6 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            log.Println("Synchronisation...")
            if err := mirror.Sync(); err != nil {
                log.Printf("Erreur sync: %v", err)
            } else {
                log.Println("Synchronisation réussie")
            }
        }
    }
}
```

## 📊 Structure du Miroir

### Arborescence Standard

```
debian-mirror/
├── dists/                          # Métadonnées de distribution
│   └── bookworm/
│       ├── Release                 # Informations de release
│       ├── Release.gpg             # Signature GPG (si disponible)
│       ├── main/
│       │   ├── binary-amd64/
│       │   │   ├── Packages        # Liste des paquets (décompressé)
│       │   │   ├── Packages.gz     # Version compressée
│       │   │   └── Packages.xz     # Version XZ (si disponible)
│       │   └── source/
│       │       ├── Sources         # Paquets sources
│       │       └── Sources.gz
│       ├── contrib/
│       └── non-free/
└── pool/                           # Paquets réels (si DownloadPackages=true)
    └── main/
        ├── a/
        │   └── apt/
        │       └── apt_2.6.1_amd64.deb
        ├── b/
        │   └── bash/
        │       └── bash_5.2.15-2+b2_amd64.deb
        └── ...
```

### Tailles Typiques

| Configuration | Métadonnées | Avec Paquets |
|---------------|-------------|--------------|
| main/amd64 seul | ~50 MB | ~65 GB |
| main+contrib/amd64 | ~60 MB | ~75 GB |
| main/multi-arch | ~150 MB | ~200 GB |
| Complet (toutes suites) | ~500 MB | ~500 GB |

## ⚡ Optimisation et Performance

### Recommandations Générales

1. **Commencez petit** : Une suite, un composant, une architecture
2. **Testez d'abord** : Mode métadonnées avant les paquets complets
3. **Monitoring** : Utilisez `GetMirrorStatus()` régulièrement
4. **Espace disque** : Vérifiez avec `EstimateMirrorSize()`

### Optimisations Réseau

```go
// Configuration optimisée pour la bande passante
config := debian.MirrorConfig{
    BaseURL:       "http://deb.debian.org/debian", // Serveur proche
    Suites:        []string{"bookworm"},            // Suite unique
    Components:    []string{"main"},                // Composant essentiel
    Architectures: []string{"amd64"},               // Architecture cible
    DownloadPackages: false,                        // Métadonnées uniquement
    Verbose:       false,                           // Moins de logs
}
```

### Optimisations Stockage

```bash
# Utiliser des liens symboliques pour économiser l'espace
# (pour plusieurs miroirs partageant des paquets communs)
ln -s /shared/pool/main/a/apt/apt_2.6.1_amd64.deb ./mirror1/pool/main/a/apt/
```

## 🔐 Sécurité et Intégrité

### Vérification d'Intégrité

```go
// Vérifier l'intégrité après création
if err := mirror.VerifyMirrorIntegrity("bookworm"); err != nil {
    log.Fatalf("Intégrité compromise: %v", err)
}
```

### Bonnes Pratiques

1. **Vérification des sommes** : Activée automatiquement
2. **Sources fiables** : Utilisez des URLs officielles
3. **Monitoring** : Surveillez les modifications inattendues
4. **Sauvegardes** : Planifiez des sauvegardes régulières

## 🔄 Maintenance

### Script de Maintenance Automatisée

```bash
#!/bin/bash
# mirror-maintenance.sh

MIRROR_PATH="/var/lib/debian-mirror"
LOG_FILE="/var/log/debian-mirror.log"

echo "$(date): Démarrage maintenance miroir" >> $LOG_FILE

# Synchronisation
deb-for-all -command mirror \
  -dest "$MIRROR_PATH" \
  -verbose >> $LOG_FILE 2>&1

if [ $? -eq 0 ]; then
    echo "$(date): Synchronisation réussie" >> $LOG_FILE
else
    echo "$(date): ERREUR synchronisation" >> $LOG_FILE
    exit 1
fi

# Nettoyage des anciens logs
find /var/log -name "debian-mirror.log.*" -mtime +30 -delete

echo "$(date): Maintenance terminée" >> $LOG_FILE
```

### Cron Configuration

```bash
# Synchronisation quotidienne à 2h du matin
0 2 * * * /usr/local/bin/mirror-maintenance.sh

# Vérification d'intégrité hebdomadaire
0 3 * * 0 deb-for-all -command verify-mirror -dest /var/lib/debian-mirror
```

## 🐛 Dépannage

### Problèmes Courants

#### Erreur de Connectivité
```
Erreur: failed to download Release file: network timeout
```
**Solution** : Vérifiez la connectivité et essayez un autre miroir.

#### Espace Disque Insuffisant
```
Erreur: no space left on device
```
**Solution** : Libérez de l'espace ou utilisez `DownloadPackages: false`.

#### Configuration Invalide
```
Erreur: BaseURL is required
```
**Solution** : Vérifiez que tous les champs obligatoires sont remplis.

### Debug Mode

```go
config.Verbose = true // Active les logs détaillés
```

### Logging Personnalisé

```go
// Redirection des logs vers un fichier
logFile, err := os.OpenFile("mirror.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
    log.Fatal(err)
}
defer logFile.Close()
log.SetOutput(logFile)
```

## 📈 Monitoring et Métriques

### Métriques de Base

```go
status, _ := mirror.GetMirrorStatus()
fmt.Printf("Fichiers: %v\n", status["file_count"])
fmt.Printf("Taille: %v bytes\n", status["total_size"])
fmt.Printf("Initialisé: %v\n", status["initialized"])
```

### Intégration Prometheus

```go
package main

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

var (
    mirrorFiles = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "debian_mirror_files_total",
            Help: "Total number of files in mirror",
        },
        []string{"mirror_name"},
    )

    mirrorSize = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "debian_mirror_size_bytes",
            Help: "Total size of mirror in bytes",
        },
        []string{"mirror_name"},
    )
)

func init() {
    prometheus.MustRegister(mirrorFiles)
    prometheus.MustRegister(mirrorSize)
}

func updateMetrics(mirrorName string, mirror *debian.Mirror) {
    status, err := mirror.GetMirrorStatus()
    if err != nil {
        return
    }

    if fileCount, ok := status["file_count"].(int); ok {
        mirrorFiles.WithLabelValues(mirrorName).Set(float64(fileCount))
    }

    if totalSize, ok := status["total_size"].(int64); ok {
        mirrorSize.WithLabelValues(mirrorName).Set(float64(totalSize))
    }
}
```

## 🌐 Configuration Multi-Miroirs

### Exemple Entreprise

```go
package main

import (
    "github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func main() {
    mirrors := map[string]debian.MirrorConfig{
        "production": {
            BaseURL:       "http://deb.debian.org/debian",
            Suites:        []string{"bookworm"},
            Components:    []string{"main"},
            Architectures: []string{"amd64"},
            DownloadPackages: true,
        },
        "development": {
            BaseURL:       "http://deb.debian.org/debian",
            Suites:        []string{"bookworm", "bookworm-updates"},
            Components:    []string{"main", "contrib"},
            Architectures: []string{"amd64", "arm64"},
            DownloadPackages: false,
        },
        "security": {
            BaseURL:       "http://security.debian.org/debian-security",
            Suites:        []string{"bookworm-security"},
            Components:    []string{"main"},
            Architectures: []string{"amd64"},
            DownloadPackages: true,
        },
    }

    for name, config := range mirrors {
        createMirror(name, config)
    }
}

func createMirror(name string, config debian.MirrorConfig) {
    mirror := debian.NewMirror(config, fmt.Sprintf("./mirrors/%s", name))
    if err := mirror.Clone(); err != nil {
        log.Printf("Erreur miroir %s: %v", name, err)
    } else {
        log.Printf("Miroir %s créé avec succès", name)
    }
}
```

## 🎓 Bonnes Pratiques

### 1. Planification
- Estimez l'espace disque requis avec `EstimateMirrorSize()`
- Commencez par des tests avec métadonnées uniquement
- Planifiez la bande passante pour les téléchargements complets

### 2. Sécurité
- Utilisez HTTPS quand disponible
- Vérifiez l'intégrité avec `VerifyMirrorIntegrity()`
- Surveillez les modifications non autorisées

### 3. Performance
- Utilisez des serveurs miroirs géographiquement proches
- Planifiez les synchronisations en dehors des heures de pointe
- Monitoreez l'utilisation des ressources

### 4. Maintenance
- Automatisez les synchronisations avec cron
- Mettez en place des alertes sur les échecs
- Documentez vos configurations personnalisées

Ce guide couvre l'utilisation complète du module Mirror. Pour des cas d'usage spécifiques ou des questions avancées, consultez les exemples dans le dossier `examples/` du projet.
