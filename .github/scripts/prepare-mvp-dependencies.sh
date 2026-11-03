#!/bin/sh
# Private, checksum-verified test dependencies. This does not publish packages.
set -eu
if [ "$#" -ne 1 ]; then
    echo "usage: $0 /absolute/new-work-directory" >&2
    exit 2
fi
case "$1" in /*) ;; *) echo 'work directory must be absolute' >&2; exit 2;; esac
task_work=$1
if [ -e "$task_work" ]; then
    echo 'work directory must not already exist' >&2
    exit 2
fi
umask 077
mkdir -p "$task_work"
task_prefix=$task_work/prefix
curl --fail --location --silent --show-error \
    https://curl.se/download/curl-8.7.1.tar.xz -o "$task_work/curl-8.7.1.tar.xz"
curl --fail --location --silent --show-error \
    https://github.com/meta/zstd/releases/download/v1.5.7/zstd-1.5.7.tar.gz -o "$task_work/zstd-1.5.7.tar.gz"
# zstd's digest is the publisher's v1.5.7 .tar.gz.sha256 release asset.
printf '%s  %s\n' \
    6fea2aac6a4610fbd0400afb0bcddbe7258a64c63f1f68e5855ebc0c659710cd "$task_work/curl-8.7.1.tar.xz" \
    eb33e51f49a15e023950cd7825ca74a4a2b43db8354825ac24fc1b7ee09e6fa3 "$task_work/zstd-1.5.7.tar.gz" \
    | shasum -a 256 --check
tar -xJf "$task_work/curl-8.7.1.tar.xz" -C "$task_work"
tar -xzf "$task_work/zstd-1.5.7.tar.gz" -C "$task_work"
case "$(uname -s)" in
    Darwin) set -- -DCURL_USE_OPENSSL=OFF -DCURL_USE_SECTRANSP=ON ;;
    Linux) set -- -DCURL_USE_OPENSSL=ON ;;
    *) echo 'unsupported test host' >&2; exit 2 ;;
esac
cmake -S "$task_work/curl-8.7.1" -B "$task_work/curl-build" \
    -DCMAKE_POLICY_VERSION_MINIMUM=3.5 -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_INSTALL_PREFIX="$task_prefix" -DCMAKE_INSTALL_LIBDIR=lib \
    -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
    -DBUILD_CURL_EXE=OFF -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=ON \
    -DCURL_USE_LIBPSL=OFF -DCURL_DISABLE_LDAP=ON "$@"
cmake --build "$task_work/curl-build" --parallel 2
cmake --install "$task_work/curl-build"
cmake -S "$task_work/zstd-1.5.7/build/cmake" -B "$task_work/zstd-build" \
    -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$task_prefix" \
    -DCMAKE_INSTALL_LIBDIR=lib -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
    -DZSTD_BUILD_PROGRAMS=OFF -DZSTD_BUILD_TESTS=OFF \
    -DZSTD_BUILD_SHARED=OFF -DZSTD_BUILD_STATIC=ON
cmake --build "$task_work/zstd-build" --parallel 2
cmake --install "$task_work/zstd-build"
# Preserve exact versioned notices and build-time TLS/dependency selection.
# Product packages use the same pinned notices without searching Homebrew or a
# bare prefix/LICENSE that could belong to a different component.
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
cmake -DDEPENDENCY_WORK="$task_work" -P "$repo/cmake/notices/CaptureExternal.cmake"
printf 'Private dependency prefix: %s\n' "$task_prefix"
