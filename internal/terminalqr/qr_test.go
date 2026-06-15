package terminalqr

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderIncludesQuietZone(t *testing.T) {
	var out bytes.Buffer
	if err := Render(&out, "vless://example", 200); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) < 10 || strings.TrimSpace(lines[0]) != "" || strings.TrimSpace(lines[len(lines)-1]) != "" {
		t.Fatal("missing QR quiet zone")
	}
}

func TestRenderNarrowTerminal(t *testing.T) {
	if err := Render(&bytes.Buffer{}, strings.Repeat("x", 500), 5); err == nil {
		t.Fatal("expected width error")
	}
}

func TestPNG(t *testing.T) {
	png, err := PNG("vless://example", 256)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(png, []byte("\x89PNG")) {
		t.Fatal("not a PNG")
	}
}
