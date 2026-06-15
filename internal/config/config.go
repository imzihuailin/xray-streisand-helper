package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
	"gopkg.in/yaml.v3"
)

const DefaultYAML = "/usr/local/etc/xray-installer/proxy.yaml"
const DefaultMetadata = "/usr/local/etc/xray-installer/install-result.json"

var uuidRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var keyRE = regexp.MustCompile(`^[A-Za-z0-9_-]{32,64}$`)
var shortIDRE = regexp.MustCompile(`(?i)^[0-9a-f]{2,16}$`)

type Reality struct {
	PublicKey string `yaml:"public-key"`
	ShortID   string `yaml:"short-id"`
}

type Proxy struct {
	Name           string  `yaml:"name"`
	Type           string  `yaml:"type"`
	Server         string  `yaml:"server"`
	Port           int     `yaml:"port"`
	UUID           string  `yaml:"uuid"`
	Network        string  `yaml:"network"`
	UDP            bool    `yaml:"udp"`
	TLS            bool    `yaml:"tls"`
	Flow           string  `yaml:"flow"`
	ServerName     string  `yaml:"servername"`
	Reality        Reality `yaml:"reality-opts"`
	SkipCertVerify bool    `yaml:"skip-cert-verify"`
}

type Document struct {
	Fingerprint string  `yaml:"global-client-fingerprint"`
	Proxies     []Proxy `yaml:"proxies"`
}

type Metadata struct {
	Domain    string `json:"domain"`
	NodeName  string `json:"node_name"`
	PublicIP  string `json:"public_ip"`
	Port      int    `json:"port"`
	UUID      string `json:"uuid"`
	PublicKey string `json:"public_key"`
	ShortID   string `json:"short_id"`
	DestHost  string `json:"dest_host"`
}

func Load(path string) (Proxy, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Proxy{}, "", fmt.Errorf("read %s: %w", path, err)
	}
	return Parse(data)
}

func Parse(data []byte) (Proxy, string, error) {
	var doc Document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Proxy{}, "", fmt.Errorf("parse YAML: %w", err)
	}
	var matches []Proxy
	for _, p := range doc.Proxies {
		if strings.EqualFold(p.Type, "vless") {
			matches = append(matches, p)
		}
	}
	if len(matches) != 1 {
		return Proxy{}, "", fmt.Errorf("expected exactly one VLESS proxy, found %d", len(matches))
	}
	if err := Validate(matches[0], doc.Fingerprint); err != nil {
		return Proxy{}, "", err
	}
	return matches[0], doc.Fingerprint, nil
}

func Validate(p Proxy, fingerprint string) error {
	var problems []string
	if strings.TrimSpace(p.Name) == "" {
		problems = append(problems, "name is empty")
	}
	if strings.TrimSpace(p.Server) == "" || strings.ContainsAny(p.Server, "/?#@") {
		problems = append(problems, "server is invalid")
	}
	if p.Port < 1 || p.Port > 65535 {
		problems = append(problems, "port is invalid")
	}
	if !uuidRE.MatchString(p.UUID) {
		problems = append(problems, "uuid is invalid")
	}
	if p.Network != "tcp" {
		problems = append(problems, "network must be tcp")
	}
	if !p.UDP || !p.TLS {
		problems = append(problems, "udp and tls must be enabled")
	}
	if p.Flow != "xtls-rprx-vision" {
		problems = append(problems, "flow must be xtls-rprx-vision")
	}
	if strings.TrimSpace(p.ServerName) == "" {
		problems = append(problems, "servername is empty")
	}
	if !keyRE.MatchString(p.Reality.PublicKey) {
		problems = append(problems, "Reality public key is invalid")
	}
	if !shortIDRE.MatchString(p.Reality.ShortID) || len(p.Reality.ShortID)%2 != 0 {
		problems = append(problems, "Reality short ID is invalid")
	}
	if p.SkipCertVerify {
		problems = append(problems, "skip-cert-verify must be false")
	}
	if fingerprint == "" {
		problems = append(problems, "global-client-fingerprint is empty")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func Link(p Proxy, fingerprint string) (string, error) {
	if err := Validate(p, fingerprint); err != nil {
		return "", err
	}
	host, err := normalizeHost(p.Server)
	if err != nil {
		return "", fmt.Errorf("normalize server: %w", err)
	}
	sni, err := normalizeHost(p.ServerName)
	if err != nil {
		return "", fmt.Errorf("normalize servername: %w", err)
	}
	u := &url.URL{
		Scheme:   "vless",
		User:     url.User(p.UUID),
		Host:     net.JoinHostPort(host, strconv.Itoa(p.Port)),
		Fragment: p.Name,
	}
	q := url.Values{}
	q.Set("encryption", "none")
	q.Set("flow", p.Flow)
	q.Set("fp", fingerprint)
	q.Set("pbk", p.Reality.PublicKey)
	q.Set("security", "reality")
	q.Set("sid", p.Reality.ShortID)
	q.Set("sni", sni)
	q.Set("type", p.Network)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func normalizeHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}
	if net.ParseIP(host) != nil {
		return host, nil
	}
	return idna.Lookup.ToASCII(host)
}

func LoadMetadata(path string) (Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, fmt.Errorf("read %s: %w", path, err)
	}
	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return Metadata{}, fmt.Errorf("parse metadata: %w", err)
	}
	return m, nil
}

func MatchMetadata(p Proxy, m Metadata) error {
	checks := []struct {
		name string
		a    string
		b    string
	}{
		{"domain", p.Server, m.Domain},
		{"node name", p.Name, m.NodeName},
		{"uuid", p.UUID, m.UUID},
		{"public key", p.Reality.PublicKey, m.PublicKey},
		{"short ID", p.Reality.ShortID, m.ShortID},
		{"SNI/destination", p.ServerName, m.DestHost},
		{"port", strconv.Itoa(p.Port), strconv.Itoa(m.Port)},
	}
	for _, c := range checks {
		if c.a != c.b {
			return fmt.Errorf("%s mismatch between YAML and metadata", c.name)
		}
	}
	return nil
}
