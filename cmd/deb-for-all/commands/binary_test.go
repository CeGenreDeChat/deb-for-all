package commands

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/CeGenreDeChat/deb-for-all/pkg/debian"
)

func TestDownloadBinaryWithVersionIntegration(t *testing.T) {
	t.Helper()
	defer silenceStdoutBinary(t)()

	downloader := debian.NewDownloader()
	pkg := &debian.Package{
		Name:         "hello",
		Version:      "2.10-2",
		Architecture: "amd64",
		DownloadURL:  "http://deb.debian.org/debian/pool/main/h/hello/hello_2.10-2_amd64.deb",
		Filename:     "hello_2.10-2_amd64.deb",
	}

	destDir := t.TempDir()
	if err := downloader.DownloadToDir(pkg, destDir); err != nil {
		t.Fatalf("failed to download binary package: %v", err)
	}

	destPath := filepath.Join(destDir, pkg.Filename)
	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("downloaded file missing: %v", err)
	}
}

func silenceStdoutBinary(t *testing.T) func() {
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
