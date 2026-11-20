#!/usr/bin/env bash
set -euo pipefail
umask 022

if [[ $# != 1 || -z $1 ]]; then
    echo 'usage: bash packaging/build-deb.sh OUTPUT_DIRECTORY' >&2
    exit 2
fi
. /etc/os-release
if [[ ${ID:-} != ubuntu || ${VERSION_ID:-} != 26.04 ]]; then
    echo 'Build on Ubuntu 26.04 with the target architecture and toolchain.' >&2
    exit 2
fi
arch=$(dpkg --print-architecture)
case "$arch:$(uname -m)" in
    amd64:x86_64|arm64:aarch64) ;;
    *) echo 'A native amd64 or arm64 Ubuntu environment is required.' >&2; exit 2 ;;
esac
for tool in cmake clang++-21 pkg-config dpkg-shlibdeps dpkg-deb; do
    command -v "$tool" >/dev/null || { echo "Missing build tool: $tool" >&2; exit 2; }
done

export LC_ALL=C TZ=UTC
export SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH:-0}
[[ $SOURCE_DATE_EPOCH =~ ^[0-9]+$ ]] || { echo 'Invalid SOURCE_DATE_EPOCH' >&2; exit 2; }
source_dir=$(cd -- "$(dirname -- "$0")/.." && pwd -P)
mkdir -p -- "$1"
output_dir=$(cd -- "$1" && pwd -P)
temp_dir=$(mktemp -d)
target_temp=
cleanup() {
    rm -rf -- "$temp_dir"
    if [[ -n $target_temp ]]; then rm -f -- "$target_temp"; fi
}
trap cleanup EXIT

pkg=symphony-node-keyring
version='0.1.1~dev1'
prefix=/opt/symphony
stage="$temp_dir/stage"
build_options=(-DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_COMPILER=clang++-21
    -DBUILD_TESTING=OFF "-DCMAKE_INSTALL_PREFIX=$prefix")
for component in linux-secret-service linux-node-keyring; do
    project="$source_dir/../$component"
    build="$temp_dir/build-$component"
    cmake -S "$project" -B "$build" "${build_options[@]}" \
        "-DCMAKE_PREFIX_PATH=$stage$prefix" \
        "-DCMAKE_CXX_FLAGS=-ffile-prefix-map=$(cd "$project" && pwd -P)=/usr/src/symphony/$component -ffile-prefix-map=$temp_dir=/usr/src/symphony/build"
    cmake --build "$build" --parallel 2
    DESTDIR="$stage" cmake --install "$build"
done

# Resolve the minimum runtime versions from the packages providing the symbols
# used by all native executables. Both SDKs are static and included in the deb.
mkdir -p "$temp_dir/debian" "$stage/DEBIAN"
cat > "$temp_dir/debian/control" <<CONTROL
Source: $pkg
Section: utils
Priority: optional
Maintainer: Local Symphony package builder <builder@localhost>
Standards-Version: 4.7.0

Package: $pkg
Architecture: any
Description: Symphony Linux node keyring runtime and native SDKs
CONTROL
# The SDK specifically requires libsecret's supported 0.21.2 behavior.
# The symbols used on Ubuntu also exist in releases as old as 0.7.
echo 'libsecret-1 0 libsecret-1-0 (>= 0.21.2)' > "$temp_dir/debian/shlibs.local"
executables=("$stage$prefix"/libexec/symphony/ssiag-node-keyring/0.1.1-dev/symphony-node-keyring-*)
depends=$(cd "$temp_dir" && dpkg-shlibdeps -O "${executables[@]/#/-e}")
[[ $depends == shlibs:Depends=?* ]] || { echo 'Could not determine runtime dependencies' >&2; exit 1; }
cat > "$stage/DEBIAN/control" <<CONTROL
Package: $pkg
Version: $version
Section: utils
Priority: optional
Architecture: $arch
Maintainer: Local Symphony package builder <builder@localhost>
Homepage: https://github.com/QuanuX/Symphony
Depends: ${depends#shlibs:Depends=}, openssh-client, dbus-daemon, gnome-keyring, libglib2.0-bin, systemd (>= 250)
Suggests: openssh-server, libsecret-1-dev, pkg-config, cryptsetup-bin, systemd-cryptsetup, libtss2-esys-3.0.2-0t64
Description: Symphony Linux node keyring runtime and native SDKs
 Contains the scoped Linux Secret Service SDK, node bootstrap SDK, OpenSSH
 provisioning tools and optional hardware-volume profile for a local keyring.
CONTROL
find "$stage" -exec touch -h -d "@$SOURCE_DATE_EPOCH" {} +
target="$output_dir/${pkg}_${version}_${arch}.deb"
target_temp=$(mktemp "$output_dir/.symphony-keyring-deb.XXXXXXXX")
dpkg-deb --root-owner-group -Zxz --threads-max=1 --build "$stage" "$target_temp"
chmod 0644 "$target_temp"
if ! ln -- "$target_temp" "$target"; then
    cmp -s -- "$target_temp" "$target" || { echo 'Different package already exists at destination' >&2; exit 1; }
fi
sha256sum -- "$target"
