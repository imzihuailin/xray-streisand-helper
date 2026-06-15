package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func archive(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractSingle(t *testing.T) {
	target := filepath.Join(t.TempDir(), "xray-installer")
	if err := extractSingle(archive(t, map[string]string{"xray-installer": "#!/bin/sh\n"}), target); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte("#!/bin/sh\n")) {
		t.Fatal("extracted content mismatch")
	}
	if info, _ := os.Stat(target); info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestExtractRejectsUnexpectedContents(t *testing.T) {
	tests := []map[string]string{
		{"other": "data"},
		{"xray-installer": "data", "extra": "data"},
	}
	for _, entries := range tests {
		if err := extractSingle(archive(t, entries), filepath.Join(t.TempDir(), "bin")); err == nil {
			t.Fatal("expected archive validation error")
		}
	}
}
