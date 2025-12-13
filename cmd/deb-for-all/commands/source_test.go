package commands

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func TestDownloadSourcePackageOrigOnlyIntegration(t *testing.T) {
	localizer := newTestLocalizerSource(t)
	destDir := t.TempDir()
	defer silenceStdoutSource(t)()

	if err := DownloadSourcePackage(
		"hello",
		"",
		"http://deb.debian.org/debian",
		[]string{"bookworm"},
		[]string{"main"},
		[]string{"source"},
		destDir,
		true,
		true,
		localizer,
	); err != nil {
		t.Fatalf("download source (orig-only) failed: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(destDir, "hello_*orig.tar.*"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected orig tarball in %s", destDir)
	}
}

func newTestLocalizerSource(t *testing.T) *i18n.Localizer {
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

func silenceStdoutSource(t *testing.T) func() {
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
