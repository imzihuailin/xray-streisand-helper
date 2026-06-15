package config

import (
	"net/url"
	"strings"
	"testing"
)

const validYAML = `global-client-fingerprint: chrome
proxies:
  - name: direct
    type: direct
  - name: "节点 #1"
    type: vless
    server: 192.0.2.10
    port: 443
    uuid: 12345678-1234-4234-9234-1234567890ab
    network: tcp
    udp: true
    tls: true
    flow: xtls-rprx-vision
    servername: example.com
    reality-opts:
      public-key: ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghi012345
      short-id: 0123456789abcdef
    skip-cert-verify: false
`

func TestParseAndLink(t *testing.T) {
	p, fp, err := Parse([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	link, err := Link(p, fp)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "vless" || u.Host != "192.0.2.10:443" || u.Fragment != "节点 #1" {
		t.Fatalf("unexpected link: %s", link)
	}
	if u.Query().Get("security") != "reality" || u.Query().Get("fp") != "chrome" {
		t.Fatalf("missing query values: %s", link)
	}
}

func TestLinkIPv6AndSpecialName(t *testing.T) {
	p, fp, err := Parse([]byte(strings.ReplaceAll(validYAML, "192.0.2.10", "2001:db8::1")))
	if err != nil {
		t.Fatal(err)
	}
	p.Name = `中文 / ? #`
	link, err := Link(p, fp)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(link)
	if u.Host != "[2001:db8::1]:443" || u.Fragment != p.Name {
		t.Fatalf("unexpected IPv6 link: %s", link)
	}
}

func TestLinkInternationalDomain(t *testing.T) {
	p, fp, err := Parse([]byte(strings.ReplaceAll(validYAML, "192.0.2.10", "例子.测试")))
	if err != nil {
		t.Fatal(err)
	}
	p.ServerName = "伪装.测试"
	link, err := Link(p, fp)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(link)
	if u.Hostname() != "xn--fsqu00a.xn--0zwm56d" || u.Query().Get("sni") != "xn--npq571l.xn--0zwm56d" {
		t.Fatalf("international domains were not normalized: %s", link)
	}
}

func TestStrictValidation(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"two vless": func(s string) string {
			return s + strings.Replace(s[strings.Index(s, "  - name: \"节点"):], "节点 #1", "other", 1)
		},
		"wrong network": func(s string) string { return strings.Replace(s, "network: tcp", "network: ws", 1) },
		"bad uuid":      func(s string) string { return strings.Replace(s, "12345678-1234-4234-9234-1234567890ab", "bad", 1) },
		"bad sid":       func(s string) string { return strings.Replace(s, "0123456789abcdef", "xyz", 1) },
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Parse([]byte(mutate(validYAML))); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestMatchMetadata(t *testing.T) {
	p, _, err := Parse([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	m := Metadata{Domain: p.Server, NodeName: p.Name, Port: p.Port, UUID: p.UUID, PublicKey: p.Reality.PublicKey, ShortID: p.Reality.ShortID, DestHost: p.ServerName}
	if err := MatchMetadata(p, m); err != nil {
		t.Fatal(err)
	}
	m.UUID = "different"
	if err := MatchMetadata(p, m); err == nil {
		t.Fatal("expected mismatch")
	}
}
