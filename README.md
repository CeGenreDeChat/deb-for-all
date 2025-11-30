# deb-for-all

**deb-for-all** is a comprehensive Go library for managing Debian packages and creating repository mirrors. This project provides both a reusable library and a command-line binary to efficiently handle Debian packages.

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

## Installation and Usage

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

# Install, build, and test
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
| `make`                  | Install, build, and test.                        |
| `make build`            | Build for the local platform.                    |
| `make build-darwin-64`  | Build for apple (arm64) plateforme apple (arm64) |
| `make build-linux-64`   | Build for linux plateforme                       |
| `make build-windows-64` | Build for windows plateforme                     |
| `make test`             | Run Robot Framework tests.                       |
| `make clean`            | Remove binaries and test results.                |

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
