#!/usr/bin/env bash
set -euo pipefail

VERSION="v0.1.1"
REPO="imzihuailin/xray-streisand-helper"

fail() { printf 'Error: %s\n' "$*" >&2; exit 1; }

[[ "$(uname -s)" == "Linux" ]] || fail "only Linux is supported"
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) fail "supported architectures: amd64, arm64" ;;
esac
command -v curl >/dev/null || fail "curl is required"
command -v sha256sum >/dev/null || fail "sha256sum is required"

asset="xray-streisand-helper_linux_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${VERSION}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "${base}/${asset}" -o "${tmp}/${asset}"
curl -fsSL "${base}/checksums.txt" -o "${tmp}/checksums.txt"
expected="$(awk -v name="$asset" '$2 == name || $2 == "./" name {print; exit}' "${tmp}/checksums.txt")"
[[ -n "$expected" ]] || fail "checksum entry not found"
(cd "$tmp" && printf '%s\n' "$expected" | sha256sum -c - >/dev/null)

entries="$(tar -tzf "${tmp}/${asset}")"
[[ "$entries" == "xray-streisand-helper" ]] || fail "unexpected archive contents"
tar -xzf "${tmp}/${asset}" -C "$tmp"
install -m 0755 "${tmp}/xray-streisand-helper" /usr/local/bin/xray-streisand-helper
exec /usr/local/bin/xray-streisand-helper setup "$@"
