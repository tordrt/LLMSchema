#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
	echo "usage: $0 <dist-directory> <version>" >&2
	exit 2
fi

dist_dir=$1
version=$2

if [[ ! -f "$dist_dir/checksums.txt" ]]; then
	echo "missing $dist_dir/checksums.txt" >&2
	exit 1
fi

(
	cd "$dist_dir"
	sha256sum --check checksums.txt
)

platforms=(
	"Darwin arm64 tar.gz llmschema"
	"Darwin x86_64 tar.gz llmschema"
	"Linux arm64 tar.gz llmschema"
	"Linux x86_64 tar.gz llmschema"
	"Windows arm64 zip llmschema.exe"
	"Windows x86_64 zip llmschema.exe"
)

for platform in "${platforms[@]}"; do
	read -r os arch extension binary_name <<<"$platform"
	archive="$dist_dir/LLMSchema_${version}_${os}_${arch}.${extension}"

	if [[ ! -f "$archive" ]]; then
		echo "missing release archive: $archive" >&2
		exit 1
	fi

	case "$extension" in
		tar.gz)
			if ! tar -tzf "$archive" | grep -Fxq "$binary_name"; then
				echo "$archive does not contain $binary_name at its root" >&2
				exit 1
			fi
			;;
		zip)
			if ! unzip -Z1 "$archive" | grep -Fxq "$binary_name"; then
				echo "$archive does not contain $binary_name at its root" >&2
				exit 1
			fi
			;;
	esac
done

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

case "$(uname -m)" in
	x86_64 | amd64)
		host_arch=x86_64
		;;
	arm64 | aarch64)
		host_arch=arm64
		;;
	*)
		echo "unsupported smoke-test architecture: $(uname -m)" >&2
		exit 1
		;;
esac

linux_archive="$dist_dir/LLMSchema_${version}_Linux_${host_arch}.tar.gz"
tar -xzf "$linux_archive" -C "$temp_dir"
binary="$temp_dir/llmschema"

if [[ ! -x "$binary" ]]; then
	echo "$linux_archive contains a non-executable llmschema binary" >&2
	exit 1
fi

expected_version="llmschema version $version"
actual_version=$("$binary" --version)
if [[ "$actual_version" != "$expected_version" ]]; then
	echo "version output = '$actual_version', want '$expected_version'" >&2
	exit 1
fi

"$binary" --help >/dev/null
"$binary" --db-url "sqlite://$temp_dir/smoke.db" >"$temp_dir/schema.md"

if ! grep -Fq "# Database Schema" "$temp_dir/schema.md"; then
	echo "SQLite smoke test did not produce schema Markdown" >&2
	exit 1
fi

echo "Release archives and Linux $host_arch CLI passed smoke tests."
