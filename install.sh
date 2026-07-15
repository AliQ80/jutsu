#!/bin/sh
# Installs the latest (or pinned $VERSION) jutsu release for this machine's OS/arch.
# Usage: curl -fsSL https://raw.githubusercontent.com/AliQ80/jutsu/main/install.sh | sh
set -e

repo="AliQ80/jutsu"
install_dir="${INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*)
		echo "error: unsupported OS '$(uname -s)'. Download a release manually from https://github.com/$repo/releases" >&2
		exit 1
		;;
esac

case "$(uname -m)" in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*)
		echo "error: unsupported architecture '$(uname -m)'. Download a release manually from https://github.com/$repo/releases" >&2
		exit 1
		;;
esac

if [ -n "$VERSION" ]; then
	tag="$VERSION"
else
	tag=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | grep '"tag_name":' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
	if [ -z "$tag" ]; then
		echo "error: couldn't resolve the latest release tag" >&2
		exit 1
	fi
fi

version="${tag#v}"
archive="jutsu_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/$repo/releases/download/$tag"

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

echo "Downloading $archive ($tag)..."
curl -fsSL -o "$workdir/$archive" "$base_url/$archive"
curl -fsSL -o "$workdir/checksums.txt" "$base_url/checksums.txt"

echo "Verifying checksum..."
(
	cd "$workdir"
	if command -v sha256sum >/dev/null 2>&1; then
		grep " $archive\$" checksums.txt | sha256sum -c -
	elif command -v shasum >/dev/null 2>&1; then
		grep " $archive\$" checksums.txt | shasum -a 256 -c -
	else
		echo "error: neither sha256sum nor shasum found, can't verify download" >&2
		exit 1
	fi
)

tar -xzf "$workdir/$archive" -C "$workdir" jutsu

mkdir -p "$install_dir"
if [ ! -w "$install_dir" ]; then
	echo "error: $install_dir isn't writable. Re-run with a writable INSTALL_DIR, or install with sudo:" >&2
	echo "  sudo INSTALL_DIR=$install_dir sh -c 'curl -fsSL https://raw.githubusercontent.com/$repo/main/install.sh | sh'" >&2
	exit 1
fi

mv "$workdir/jutsu" "$install_dir/jutsu"
chmod +x "$install_dir/jutsu"

echo "Installed jutsu $tag to $install_dir/jutsu"

case ":$PATH:" in
	*":$install_dir:"*) ;;
	*) echo "warning: $install_dir is not on your \$PATH" >&2 ;;
esac

if ! command -v jj >/dev/null 2>&1; then
	echo "warning: jj (Jujutsu) not found on \$PATH -- jutsu needs it. See https://jj-vcs.github.io/jj/latest/install-and-setup/" >&2
fi
