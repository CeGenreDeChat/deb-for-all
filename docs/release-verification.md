# Vérification des fichiers Release

Ce document décrit la fonctionnalité de vérification des fichiers Release implémentée dans le projet deb-for-all.

## Vue d'ensemble

La vérification des fichiers Release permet de s'assurer de l'intégrité des paquets téléchargés depuis les dépôts Debian/Ubuntu en vérifiant leurs checksums contre les valeurs publiées dans le fichier Release officiel du dépôt.

## Fonctionnalités

### 1. Structures de données

#### `ReleaseFile`
Représente le contenu parsé d'un fichier Release :
```go
type ReleaseFile struct {
    Origin        string           // Ex: "Debian", "Ubuntu"
    Label         string           // Ex: "Debian", "Ubuntu"
    Suite         string           // Ex: "stable", "testing"
    Version       string           // Ex: "12.11"
    Codename      string           // Ex: "bookworm", "jammy"
    Date          string           // Date de création du Release
    Description   string           // Description du dépôt
    Architectures []string         // Architectures supportées
    Components    []string         // Composants (main, contrib, non-free, etc.)
    MD5Sum        []FileChecksum   // Checksums MD5
    SHA1          []FileChecksum   // Checksums SHA1 (rare)
    SHA256        []FileChecksum   // Checksums SHA256 (recommandé)
}
```

#### `FileChecksum`
Représente une entrée de checksum :
```go
type FileChecksum struct {
    Hash     string // Valeur du hash
    Size     int64  // Taille du fichier en octets
    Filename string // Chemin relatif du fichier
}
```

### 2. Méthodes d'API

#### Configuration
```go
// Activer la vérification Release
repo.EnableReleaseVerification()

// Désactiver la vérification Release
repo.DisableReleaseVerification()

// Vérifier si la vérification est activée
enabled := repo.IsReleaseVerificationEnabled()
```

#### Récupération et parsing
```go
// Télécharger et parser le fichier Release
err := repo.FetchReleaseFile()

// Obtenir les informations Release
releaseInfo := repo.GetReleaseInfo() // peut être nil
```

#### Vérification des paquets
Quand `VerifyRelease` est activé, `FetchPackages()` vérifie automatiquement les checksums de tous les fichiers Packages téléchargés.

### 3. Algorithmes de vérification

La vérification suit cette priorité :
1. **SHA256** (recommandé et le plus sécurisé)
2. **MD5** (fallback si SHA256 non disponible)
3. **SHA1** (rarement utilisé)

Si aucun checksum n'est trouvé pour un fichier, une erreur est retournée.

## Utilisation

### Exemple basique
```go
repo := debian.NewRepository(
    "debian-main",
    "http://deb.debian.org/debian",
    "Dépôt principal Debian",
    "bookworm",
    []string{"main"},
    []string{"amd64"},
)

// Activer la vérification Release
repo.EnableReleaseVerification()

// Les paquets seront automatiquement vérifiés
packages, err := repo.FetchPackages()
if err != nil {
    log.Fatal(err)
}
```

### Exemple avec informations Release
```go
repo.EnableReleaseVerification()

// Récupérer explicitement le fichier Release
err := repo.FetchReleaseFile()
if err != nil {
    log.Fatal(err)
}

releaseInfo := repo.GetReleaseInfo()
fmt.Printf("Origin: %s\n", releaseInfo.Origin)
fmt.Printf("Codename: %s\n", releaseInfo.Codename)
fmt.Printf("Checksums SHA256: %d\n", len(releaseInfo.SHA256))
```

## Sécurité

### Avantages
- **Intégrité** : Vérification que les fichiers n'ont pas été corrompus ou modifiés
- **Authenticité** : Les checksums proviennent du dépôt officiel
- **Traçabilité** : Chaque fichier est vérifié individuellement

### Limitations
- **Pas de vérification GPG** : Les signatures GPG des fichiers Release ne sont pas vérifiées (amélioration future)
- **Confiance en HTTPS** : Dépend de la sécurité du transport HTTPS
- **Algorithmes** : MD5 est considéré comme faible cryptographiquement (mais toujours utilisé par certains dépôts)

## Gestion d'erreurs

### Types d'erreurs possibles
1. **Réseau** : Échec de téléchargement du fichier Release
2. **Parsing** : Fichier Release malformé
3. **Checksum manquant** : Aucun checksum trouvé pour un fichier
4. **Checksum invalide** : Le checksum calculé ne correspond pas

### Exemple de gestion
```go
packages, err := repo.FetchPackages()
if err != nil {
    if strings.Contains(err.Error(), "checksum") {
        log.Printf("Erreur de vérification: %v", err)
        // Peut-être désactiver la vérification et réessayer
    } else {
        log.Printf("Autre erreur: %v", err)
    }
}
```

## Compatibilité

### Dépôts testés
- ✅ **Debian** (bookworm, bullseye, sid)
- ✅ **Ubuntu** (jammy, focal)
- ✅ **Formats** : Packages, Packages.gz, Packages.xz

### Rétrocompatibilité
- Par défaut, `VerifyRelease` est `false` pour maintenir la compatibilité
- L'API existante fonctionne sans changements
- Les applications existantes peuvent activer la vérification graduellement

## Exemples complets

Voir les exemples dans le répertoire `examples/` :
- `examples/release-verification/` : Test complet de la fonctionnalité
- `examples/checksum-test/` : Vérification détaillée des checksums
- `examples/error-handling/` : Gestion des erreurs et cas limites

## Développement futur

### Améliorations possibles
1. **Vérification GPG** : Vérifier les signatures `Release.gpg`
2. **Cache** : Mettre en cache les fichiers Release pour éviter les re-téléchargements
3. **Parallélisation** : Vérifier les checksums en parallèle
4. **Métadonnées étendues** : Parser plus d'informations du fichier Release
5. **Support InRelease** : Support des fichiers InRelease (Release + signature GPG intégrée)

### API futures potentielles
```go
// Vérification GPG (future)
repo.EnableGPGVerification(keyring)

// Cache Release (future)
repo.SetReleaseCache(cachePath, ttl)

// Vérification parallèle (future)
repo.SetParallelVerification(workers)
```
