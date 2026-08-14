#!/bin/sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    echo "usage: prepare-real-adapter.sh ABSOLUTE_PREFIX [user|system]" >&2
    exit 64
fi

prefix=$1
scope=${2:-user}
case "$prefix" in
    /*) ;;
    *) echo "integration prefix must be absolute" >&2; exit 64 ;;
esac
case "$scope" in
    user|system) ;;
    *) echo "integration scope must be user or system" >&2; exit 64 ;;
esac

package_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$package_root"
swift build -c release >&2
binary_dir=$(swift build -c release --show-bin-path)
exec "$binary_dir/symphony-ssiag-provider-macos-keychain" \
    install --scope "$scope" --prefix "$prefix"
