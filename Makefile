# Makefile for deb-for-all project

.PHONY: all build clean test examples mirror-example package-example help

# Variables
BINARY_NAME = deb-for-all
BUILD_DIR = bin
EXAMPLES_DIR = examples

# Check for Windows
ifeq ($(OS),Windows_NT)
    BINARY_EXT = .exe
    RM_CMD = del /Q
    MKDIR_CMD = if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
else
    BINARY_EXT =
    RM_CMD = rm -rf
    MKDIR_CMD = mkdir -p $(BUILD_DIR)
endif

all: build

# Build the main binary
build:
	$(MKDIR_CMD)
	go build -o $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) ./cmd/deb-for-all

# Clean build artifacts
clean:
	go clean
	$(RM_CMD) $(BUILD_DIR)/*$(BINARY_EXT)

# Run tests
test:
	go test ./pkg/... -v

# Run the built binary
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT)

# Install the binary to $GOPATH/bin
install:
	go install ./cmd/deb-for-all

# Run the mirror example
mirror-example:
	cd $(EXAMPLES_DIR)/mirror && go run main.go

# Run package example
package-example:
	cd $(EXAMPLES_DIR)/package && go run main.go

# Show examples of usage
examples: help-examples

# Build for multiple platforms
build-all: build-linux build-windows build-darwin

build-linux:
	$(MKDIR_CMD)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/deb-for-all

build-windows:
	$(MKDIR_CMD)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/deb-for-all

build-darwin:
	$(MKDIR_CMD)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/deb-for-all

# Quick test of mirror functionality
test-mirror: build
	./$(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) -command mirror -dest ./test-mirror -verbose

# Test package download
test-download: build
	./$(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) -command download-source -package hello -version 2.10-2 -dest ./test-downloads -verbose

# Show help
help:
	@echo "deb-for-all Makefile"
	@echo "Available targets:"
	@echo "  build          - Build the main binary"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  run            - Build and run the binary"
	@echo "  install        - Install to GOPATH/bin"
	@echo "  mirror-example - Run the mirror example"
	@echo "  package-example- Run the package example"
	@echo "  build-all      - Build for all platforms"
	@echo "  test-mirror    - Quick test of mirror functionality"
	@echo "  test-download  - Test package download"
	@echo "  help-examples  - Show usage examples"
	@echo "  help           - Show this help"

help-examples:
	@echo "deb-for-all Usage Examples:"
	@echo ""
	@echo "# Mirror a Debian repository (metadata only)"
	@echo "make test-mirror"
	@echo "$(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) -command mirror -dest ./debian-mirror -verbose"
	@echo ""
	@echo "# Mirror with packages (WARNING: Large download)"
	@echo "$(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) -command mirror -dest ./debian-mirror -download-packages -verbose"
	@echo ""
	@echo "# Custom mirror configuration"
	@echo "$(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) -command mirror \\"
	@echo "  -url http://deb.debian.org/debian \\"
	@echo "  -suites bookworm,bullseye \\"
	@echo "  -components main,contrib \\"
	@echo "  -architectures amd64,arm64 \\"
	@echo "  -dest ./custom-mirror -verbose"
	@echo ""
	@echo "# Download a source package"
	@echo "make test-download"
	@echo "$(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) -command download-source -package hello -version 2.10-2"
	@echo ""
	@echo "# Run interactive examples"
	@echo "make mirror-example"