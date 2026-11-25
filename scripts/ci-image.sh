#!/usr/bin/env bash
# Print the container image reference CI uses for an OpenWrt lab target
# (openwrt-21-02, openwrt-22-03, openwrt-23-05, openwrt-24-10, openwrt-25-12).
# The reference lives in ci/images/Dockerfile.<name>, where Dependabot can
# update it; this is the one place scripts and workflows read it from.
set -euo pipefail
name=${1:?usage: ci-image.sh openwrt-<21-02|22-03|23-05|24-10|25-12>}
file="$(dirname "$0")/../ci/images/Dockerfile.$name"
ref=$(sed -n 's/^FROM //p' "$file")
[ -n "$ref" ] || { echo "no FROM line in $file" >&2; exit 1; }
echo "$ref"
