#!/bin/sh

set -eu
umask 022

repository="Payhon/AppKernia"
requested_version="${AKONE_VERSION:-latest}"
install_dir="${AKONE_INSTALL_DIR:-}"
install_dir_explicit='false'
[ -n "$install_dir" ] && install_dir_explicit='true'
temporary_dir=""
temporary_target=""

usage() {
  cat <<'EOF'
Install akone from an official GitHub Release.

Usage: install.sh [--version X.Y.Z[-preview.N]] [--install-dir DIR | --prefix DIR]

Environment:
  AKONE_VERSION       Version without the leading v (default: latest stable)
  AKONE_INSTALL_DIR   Exact destination directory (default: $HOME/.local/bin)
EOF
}

fail() {
  printf 'akone.install.error=%s\n' "$1" >&2
  exit 1
}

cleanup() {
  if [ -n "$temporary_target" ]; then rm -f "$temporary_target"; fi
  if [ -n "$temporary_dir" ]; then rm -rf "$temporary_dir"; fi
}
trap cleanup EXIT HUP INT TERM

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || fail '--version requires a value'
      requested_version="${2#v}"
      shift 2
      ;;
    --install-dir)
      [ "$#" -ge 2 ] || fail '--install-dir requires a value'
      install_dir="$2"
      install_dir_explicit='true'
      shift 2
      ;;
    --prefix)
      [ "$#" -ge 2 ] || fail '--prefix requires a value'
      install_dir="${2%/}/bin"
      install_dir_explicit='true'
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *) fail "unknown argument: $1" ;;
  esac
done

command -v curl >/dev/null 2>&1 || fail 'curl is required'
command -v tar >/dev/null 2>&1 || fail 'tar is required'
command -v install >/dev/null 2>&1 || fail 'install is required'
[ "$install_dir_explicit" = 'true' ] || {
  [ -n "${HOME:-}" ] || fail 'HOME is empty; pass --install-dir'
  install_dir="${HOME%/}/.local/bin"
}
[ -n "$install_dir" ] || fail 'HOME is empty; pass --install-dir'

if [ "$requested_version" != 'latest' ] && ! printf '%s\n' "$requested_version" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'; then
  fail "invalid version: $requested_version"
fi

case "$(uname -s)" in
  Darwin) release_os='darwin' ;;
  Linux) release_os='linux' ;;
  *) fail "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) release_arch='amd64' ;;
  arm64|aarch64) release_arch='arm64' ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

temporary_dir="$(mktemp -d "${TMPDIR:-/tmp}/akone.XXXXXX")"
checksums="$temporary_dir/checksums.txt"
if [ "$requested_version" = 'latest' ]; then
  release_base="https://github.com/$repository/releases/latest/download"
else
  release_base="https://github.com/$repository/releases/download/v$requested_version"
fi

curl --fail --location --proto '=https' --retry 3 --silent --show-error \
  "$release_base/checksums.txt" --output "$checksums"

suffix="_${release_os}_${release_arch}.tar.gz"
if [ "$requested_version" = 'latest' ]; then
  archive_name="$(awk -v suffix="$suffix" '
    {
      name=$2
      sub(/^\*/, "", name)
      if (length(name) > length(suffix) && substr(name, length(name) - length(suffix) + 1) == suffix) print name
    }
  ' "$checksums")"
else
  expected_archive="akone_${requested_version}${suffix}"
  archive_name="$(awk -v expected="$expected_archive" '
    { name=$2; sub(/^\*/, "", name); if (name == expected) print name }
  ' "$checksums")"
fi
[ -n "$archive_name" ] || fail "release has no artifact for ${release_os}/${release_arch}"
case "$archive_name" in *'/'*|*'..'*) fail 'release checksum contains an unsafe filename' ;; esac
[ "$(printf '%s\n' "$archive_name" | wc -l | tr -d ' ')" = '1' ] || fail "release has duplicate artifacts for ${release_os}/${release_arch}"

archive_version="${archive_name#akone_}"
archive_version="${archive_version%$suffix}"
if [ "$archive_name" != "akone_${archive_version}${suffix}" ] || ! printf '%s\n' "$archive_version" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'; then
  fail 'release checksum contains an invalid archive version'
fi
if [ "$requested_version" != 'latest' ] && [ "$archive_version" != "$requested_version" ]; then
  fail "release archive version $archive_version does not match requested version $requested_version"
fi

archive="$temporary_dir/$archive_name"
curl --fail --location --proto '=https' --retry 3 --silent --show-error \
  "$release_base/$archive_name" --output "$archive"

expected_checksum="$(awk -v expected="$archive_name" '{ name=$2; sub(/^\*/, "", name); if (name == expected) print tolower($1) }' "$checksums")"
[ -n "$expected_checksum" ] || fail 'artifact checksum is missing'
if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum="$(sha256sum "$archive" | awk '{ print tolower($1) }')"
else
  command -v shasum >/dev/null 2>&1 || fail 'sha256sum or shasum is required'
  actual_checksum="$(shasum -a 256 "$archive" | awk '{ print tolower($1) }')"
fi
[ "$actual_checksum" = "$expected_checksum" ] || fail "checksum mismatch for $archive_name"

if tar -tzf "$archive" | grep -Eq '(^/|(^|/)\.\.(/|$))'; then
  fail 'archive contains an unsafe path'
fi

extract_dir="$temporary_dir/extract"
mkdir -p "$extract_dir"
tar -xzf "$archive" -C "$extract_dir"
binary="$extract_dir/akone"
[ -f "$binary" ] || fail 'archive does not contain akone'
[ ! -L "$binary" ] || fail 'archive contains a symbolic-link executable'
chmod 0755 "$binary"
version_output="$("$binary" version --json)" || fail 'downloaded akone failed its version check'
binary_version="$(printf '%s\n' "$version_output" | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[ -n "$binary_version" ] || fail 'downloaded akone returned no version'
[ "$(printf '%s\n' "$binary_version" | wc -l | tr -d ' ')" = '1' ] || fail 'downloaded akone returned an ambiguous version'
[ "$binary_version" = "$archive_version" ] || fail "downloaded akone version $binary_version does not match archive version $archive_version"

mkdir -p "$install_dir"
temporary_target="$(mktemp "$install_dir/.akone.XXXXXX")"
install -m 0755 "$binary" "$temporary_target"
mv -f "$temporary_target" "$install_dir/akone"
temporary_target=""

printf 'akone.install.path=%s\n' "$install_dir/akone"
case ":${PATH:-}:" in
  *":$install_dir:"*) ;;
  *) printf 'akone.install.hint=add %s to PATH\n' "$install_dir" ;;
esac
