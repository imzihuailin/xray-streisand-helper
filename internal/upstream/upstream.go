package upstream

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const Version = "v0.1.10"
const Commit = "022b6ad3d6126fb0f5ffc1db68c035485f8f03a8"

var SHA256 = map[string]string{
	"amd64": "66530941f82f67024d11a9cd4e17ebedc913093daed53e3f9a172b98e8154720",
	"arm64": "58058453975a22782f4faec8acfb233fb6f7e2360523ef4c88f6fb700b30c187",
}

type Manager struct {
	BaseDir string
	Client  *http.Client
}

func (m Manager) Prepare(ctx context.Context) (string, error) {
	arch := runtime.GOARCH
	want, ok := SHA256[arch]
	if runtime.GOOS != "linux" || !ok {
		return "", fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, arch)
	}
	base := m.BaseDir
	if base == "" {
		base = "/var/lib/xray-streisand-helper/upstream"
	}
	versionDir := filepath.Join(base, Version)
	bin := filepath.Join(versionDir, "xray-installer")
	if verifyVersion(ctx, bin) == nil {
		return bin, nil
	}
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return "", fmt.Errorf("create upstream directory: %w", err)
	}
	asset := fmt.Sprintf("xray-installer_linux_%s.tar.gz", arch)
	u := "https://github.com/manateelazycat/xray-installer/releases/download/" + Version + "/" + asset
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download upstream: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download upstream: HTTP %s", resp.Status)
	}
	tmp, err := os.CreateTemp(versionDir, "archive-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(resp.Body, 100<<20)); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return "", fmt.Errorf("upstream SHA-256 mismatch: got %s", got)
	}
	if err := extractSingle(tmpName, bin); err != nil {
		return "", err
	}
	if err := verifyVersion(ctx, bin); err != nil {
		return "", err
	}
	return bin, nil
}

func extractSingle(archive, target string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open upstream archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	hdr, err := tr.Next()
	if err != nil {
		return fmt.Errorf("read upstream archive: %w", err)
	}
	if hdr.Name != "xray-installer" || !hdr.FileInfo().Mode().IsRegular() || hdr.Size > 100<<20 {
		return fmt.Errorf("unexpected upstream archive entry %q", hdr.Name)
	}
	tmp := target + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.CopyN(out, tr, hdr.Size); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if _, err := tr.Next(); err != io.EOF {
		os.Remove(tmp)
		return fmt.Errorf("upstream archive must contain exactly one file")
	}
	return os.Rename(tmp, target)
}

func verifyVersion(ctx context.Context, bin string) error {
	out, err := exec.CommandContext(ctx, bin, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify upstream version: %w", err)
	}
	text := string(out)
	if !strings.Contains(text, "xray-installer "+Version) || !strings.Contains(text, Commit) {
		return fmt.Errorf("unexpected upstream version output")
	}
	return nil
}
