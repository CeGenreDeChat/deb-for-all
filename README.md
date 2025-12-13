# deb-for-all

**deb-for-all** is a Go library for downloading Debian packages and creating repository mirrors. This project provides both a reusable library and a command-line binary to efficiently handle Debian package retrieval (no install/remove/upgrade operations).

---

## 🚀 Features

### 📦 Package Management
- Read, write, and validate Debian control files
- Download binary and source packages with progress tracking
- Checksum verification and retry mechanisms
- Concurrent downloads for multiple packages

### 🔄 Repository Mirroring
- **Complete mirror creation** of Debian repositories
- Support for multiple distributions (suites), components, and architectures
- Mirror modes: metadata only or with full packages
- Directory structure compliant with Debian standards
- Incremental synchronization and integrity verification

### 🗂️ Repository Management
- Interaction with Debian repositories
- Automatic parsing of Release and Packages files
- Handling of various compression formats (.gz, .xz)
- Multi-architecture support

---

## Setup and Usage

### Prerequisites
- Go 1.20+
- Python 3.8+
- Git
- make

### Steps

```bash
# Clone the repository
git clone https://github.com/CeGenreDeChat/deb-for-all.git
cd deb-for-all

# Build and test
make

# Build for all platforms (Windows/Linux)
make build-all

# Run Robot Framework tests
make test

# Clean binaries and test results
make clean
```

### Available Make Targets
| Command                 | Description                                      |
|-------------------------|--------------------------------------------------|
| `make`                  | Build and test.                                  |
| `make build`            | Build for the local platform.                    |
| `make build-darwin-64`  | Build for apple (arm64) plateforme apple (arm64) |
| `make build-linux-64`   | Build for linux plateforme                       |
| `make build-windows-64` | Build for windows plateforme                     |
| `make test`             | Run Robot Framework tests.                       |
| `make clean`            | Remove binaries and test results.                |

---

## CLI Usage

The `deb-for-all` binary provides several commands for Debian package management.

### Language Configuration

Set the `DEB_FOR_ALL_LANG` environment variable to change the output language:
```bash
# English (default)
export DEB_FOR_ALL_LANG=en

# French
export DEB_FOR_ALL_LANG=fr
```

### Global Flags
- `--keyring` (comma-separated) trusted GPG keyring files for Release/InRelease verification.
- `--no-gpg-verify` disable GPG signature verification (checksum verification remains).

### Commands

#### Download Binary Package
Download a binary package from Debian repositories:
```bash
deb-for-all download -p <package-name> [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--package` | `-p` | Package name (required) | - |
| `--version` | - | Specific version to download | latest |
| `--dest` | `-d` | Destination directory | `./downloads` |
| `--silent` | `-s` | Suppress output | `false` |
| `--verbose` | `-v` | Verbose output | `false` |

**Example:**
```bash
deb-for-all download -p curl --version 7.88.1-10 -d ./packages
```

#### Download Source Package
Download source files for a package:
```bash
deb-for-all download-source -p <package-name> [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--package` | `-p` | Package name (required) | - |
| `--version` | - | Specific version to download | latest |
| `--dest` | `-d` | Destination directory | `./downloads` |
| `--orig-only` | - | Download only the orig tarball | `false` |
| `--silent` | `-s` | Suppress output | `false` |
| `--verbose` | `-v` | Verbose output | `false` |

**Example:**
```bash
deb-for-all download-source -p nginx --orig-only -d ./sources
```

#### Update Package Index Cache
Fetch and cache Release/Packages metadata for suites/components/architectures:
```bash
deb-for-all update --suites bookworm --components main --architectures amd64 --cache ./cache
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--url` | `-u` | Repository URL | `http://deb.debian.org/debian` |
| `--suites` | - | Suites (comma-separated) | `bookworm` |
| `--components` | - | Components (comma-separated) | `main` |
| `--architectures` | - | Architectures (comma-separated) | `amd64` |
| `--cache` | - | Cache directory | `./cache` |
| `--verbose` | `-v` | Verbose output | `false` |

#### Build Custom Repository (with dependencies)
Create a subset repository from an XML package list and download all required packages (with optional dependency exclusions):
```bash
deb-for-all custom-repo --packages-xml ./packages.xml --exclude-deps recommends,suggests --dest ./custom-repo --suites bookworm --components main --architectures amd64
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--packages-xml` | - | XML file containing `<packages><package version="">name</package></packages>` | - |
| `--exclude-deps` | - | Dependency types to exclude (allowed: `depends,pre-depends,recommends,suggests,enhances`) | - |
| `--url` | `-u` | Repository URL | `http://deb.debian.org/debian` |
| `--suites` | - | Suites (comma-separated) | `bookworm` |
| `--components` | - | Components (comma-separated) | `main` |
| `--architectures` | - | Architectures (comma-separated) | `amd64` |
| `--dest` | `-d` | Destination directory | `./downloads` |
| `--keyring` | - | Comma-separated keyrings for GPG verification | - |
| `--no-gpg-verify` | - | Disable signature verification | `false` |
| `--verbose` | `-v` | Verbose output | `false` |

#### Create Mirror
Create a local mirror of a Debian repository:
```bash
deb-for-all mirror [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--url` | `-u` | Repository base URL | `http://deb.debian.org/debian` |
| `--suites` | - | Comma-separated list of suites | `bookworm` |
| `--components` | - | Comma-separated list of components | `main` |
| `--architectures` | - | Comma-separated list of architectures | `amd64` |
| `--dest` | `-d` | Destination directory | `./downloads` |
| `--download-packages` | - | Download actual packages (not just metadata) | `false` |
| `--verbose` | `-v` | Verbose output | `false` |

**Examples:**
```bash
# Mirror metadata only
deb-for-all mirror -u http://deb.debian.org/debian --suites bookworm -d ./mirror

# Mirror with packages
deb-for-all mirror --suites bookworm,bullseye --components main,contrib --download-packages -d ./mirror -v

# Mirror multiple architectures
deb-for-all mirror --architectures amd64,arm64 --suites bookworm -d ./mirror
```

---

## Contributing

Contributions are welcome! Here's how you can help:

1. **Fork** the repository.
2. Create a branch for your feature or fix:
   ```bash
   git checkout -b my-new-feature
   ```
3. **Commit** your changes:
   ```bash
   git commit -m "Add my new feature"
   ```
4. **Push** to your fork:
   ```bash
   git push origin my-new-feature
   ```
5. Open a **Pull Request** on the original repository.

---

## License

This project is licensed under the **[MIT License](LICENSE)**.
