package commands

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func TestCustomRepoSystemdWithoutRecommendsIntegration(t *testing.T) {
	localizer := newTestLocalizerCustom(t)
	destDir := t.TempDir()
	defer silenceStdoutCustom(t)()

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("unable to compute repo root: %v", err)
	}
	packagesPath := filepath.Join(repoRoot, "packages.xml")

	original, _ := os.ReadFile(packagesPath)
	newContent := []byte("<packages>\n    <package>systemd</package>\n</packages>\n")
	if err := os.WriteFile(packagesPath, newContent, debian.FilePermission); err != nil {
		t.Fatalf("unable to write packages.xml: %v", err)
	}
	t.Cleanup(func() {
		if len(original) > 0 {
			_ = os.WriteFile(packagesPath, original, debian.FilePermission)
		} else {
			_ = os.Remove(packagesPath)
		}
	})

	if err := BuildCustomRepository(
		"http://deb.debian.org/debian",
		"bookworm",
		"main",
		"amd64",
		destDir,
		packagesPath,
		"recommends,suggests",
		nil,
		nil,
		true,
		false,
		0,
		localizer,
	); err != nil {
		t.Fatalf("custom-repo build failed: %v", err)
	}

	found := false
	walkErr := filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		if strings.HasPrefix(base, "systemd_") && strings.HasSuffix(base, ".deb") {
			found = true
			return fs.SkipDir
		}

		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk failed: %v", walkErr)
	}
	if !found {
		t.Fatalf("expected systemd package under %s", destDir)
	}
}

func TestCustomRepoSinglePackageNoDependenciesIntegration(t *testing.T) {
	localizer := newTestLocalizerCustom(t)
	destDir := t.TempDir()
	defer silenceStdoutCustom(t)()

	packagesDir := t.TempDir()
	packagesPath := filepath.Join(packagesDir, "packages.xml")

	content := []byte("<packages>\n    <package>hello</package>\n</packages>\n")
	if err := os.WriteFile(packagesPath, content, debian.FilePermission); err != nil {
		t.Fatalf("unable to write packages.xml: %v", err)
	}

	if err := BuildCustomRepository(
		"http://deb.debian.org/debian",
		"bookworm",
		"main",
		"amd64",
		destDir,
		packagesPath,
		"depends,pre-depends,recommends,suggests,enhances",
		nil,
		nil,
		true,
		false,
		0,
		localizer,
	); err != nil {
		t.Fatalf("custom-repo build failed: %v", err)
	}

	var debs []string
	walkErr := filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(path), ".deb") {
			debs = append(debs, path)
		}

		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk failed: %v", walkErr)
	}

	if len(debs) != 1 {
		t.Fatalf("expected 1 downloaded package, got %d", len(debs))
	}

	if base := filepath.Base(debs[0]); !strings.HasPrefix(base, "hello_") {
		t.Fatalf("expected hello package, got %s", base)
	}
}

func newTestLocalizerCustom(t *testing.T) *i18n.Localizer {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	must := func(_ *i18n.MessageFile, err error) {
		if err != nil {
			t.Fatalf("failed to load locale: %v", err)
		}
	}

	must(bundle.LoadMessageFile("../locales/en.toml"))
	must(bundle.LoadMessageFile("../locales/fr.toml"))

	return i18n.NewLocalizer(bundle, "en")
}

func silenceStdoutCustom(t *testing.T) func() {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to silence stdout: %v", err)
	}

	original := os.Stdout
	os.Stdout = writer

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, reader)
		close(done)
	}()

	return func() {
		_ = writer.Close()
		<-done
		os.Stdout = original
	}
}
