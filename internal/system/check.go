package system

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func Platform() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("only Linux is supported")
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return fmt.Errorf("unsupported architecture %s", runtime.GOARCH)
	}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("read /etc/os-release: %w", err)
	}
	text := strings.ToLower(string(data))
	if !strings.Contains(text, "id=debian") && !strings.Contains(text, "id=ubuntu") {
		return fmt.Errorf("only Debian and Ubuntu are supported")
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return fmt.Errorf("systemd is required")
	}
	return nil
}

func ServiceActive(ctx context.Context, r Runner) error {
	out, err := r.Run(ctx, "systemctl", "is-active", "--quiet", "xray.service")
	if err != nil {
		return fmt.Errorf("xray.service is not active: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func Port443OwnedByXray(ctx context.Context, r Runner) error {
	out, err := r.Run(ctx, "ss", "-ltnp", "sport", "=", ":443")
	if err != nil {
		return fmt.Errorf("inspect TCP 443: %w", err)
	}
	text := strings.ToLower(string(out))
	if !strings.Contains(text, ":443") {
		return fmt.Errorf("TCP 443 is not listening")
	}
	if !strings.Contains(text, "xray") {
		return fmt.Errorf("TCP 443 is not owned by Xray")
	}
	return nil
}

func Port443Conflict(ctx context.Context, r Runner) error {
	out, err := r.Run(ctx, "ss", "-ltnp", "sport", "=", ":443")
	if err != nil {
		return nil
	}
	text := strings.ToLower(string(out))
	if !strings.Contains(text, ":443") {
		return nil
	}
	if strings.Contains(text, "xray") {
		return nil
	}
	return fmt.Errorf("TCP 443 is already occupied by a non-Xray process")
}

func DNSMatches(ctx context.Context, domain, publicIP string) error {
	ip := net.ParseIP(publicIP)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("metadata public IPv4 is invalid")
	}
	addrs, err := net.DefaultResolver.LookupIP(ctx, "ip4", domain)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", domain, err)
	}
	for _, addr := range addrs {
		if addr.Equal(ip) {
			return nil
		}
	}
	return fmt.Errorf("DNS A records for %s do not include public IPv4 %s", domain, publicIP)
}

func ExistingXrayKnown(ctx context.Context, r Runner) error {
	path, err := exec.LookPath("xray")
	if err != nil {
		return nil
	}
	out, err := r.Run(ctx, path, "version")
	if err != nil || !strings.Contains(strings.ToLower(string(out)), "xray") {
		return fmt.Errorf("an unknown program named xray is installed at %s", path)
	}
	return nil
}

func IsRoot() bool { return os.Geteuid() == 0 }

func ReadDomain(in *bufio.Reader) (string, error) {
	line, err := in.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	domain := strings.TrimSpace(line)
	if domain == "" {
		return "", fmt.Errorf("domain cannot be empty")
	}
	return domain, nil
}
