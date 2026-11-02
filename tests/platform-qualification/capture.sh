#!/bin/sh
set -eu
umask 077
if [ "$#" -ne 2 ]; then echo "usage: $0 PROFILE_ID /absolute/work-directory" >&2; exit 2; fi
profile=$1
work=$2
case "$work" in /*) ;; *) echo 'work directory must be absolute' >&2; exit 2 ;; esac
repository=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
mkdir -p "$work/raw"
uname -s > "$work/raw/os.txt"
uname -m > "$work/raw/architecture.txt"
uname -r > "$work/raw/kernel.txt"
"${CXX:-c++}" --version > "$work/raw/compiler.txt"
cmake --version > "$work/raw/cmake.txt"
if command -v go >/dev/null 2>&1; then GOTOOLCHAIN=local GOENV=off go version > "$work/raw/go.txt"; else printf 'unavailable; no Go qualification claimed by this native gate\n' > "$work/raw/go.txt"; fi
git -C "$repository" rev-parse HEAD > "$work/raw/commit.txt"
git -C "$repository" rev-parse HEAD^{tree} > "$work/raw/tree.txt"
git -C "$repository" status --porcelain > "$work/raw/source-state.txt"
if [ "$(uname -s)" = Darwin ]; then
  sw_vers -productVersion > "$work/raw/release.txt"
  sw_vers > "$work/raw/os-release.txt"
  "${CC:-cc}" "$repository/tests/platform-qualification/filesystem.c" -o "$work/filesystem-probe"
  "$work/filesystem-probe" "$work" > "$work/raw/filesystem.txt"
  rm -f "$work/filesystem-probe"
  sysctl -n hw.model > "$work/raw/hardware.txt" 2> "$work/raw/hardware-error.txt" || printf 'unavailable\n' > "$work/raw/hardware.txt"
  xcrun --show-sdk-version > "$work/raw/sdk.txt"
  otool -L /usr/lib/libSystem.B.dylib > "$work/raw/c-runtime.txt" 2>&1 || true
  printf 'libc++.1.dylib; dyld shared cache; exact loaded library proof is per executable\n' > "$work/raw/cxx-runtime.txt"
  printf 'not assessed\n' > "$work/raw/virtualization.txt"
else
  . /etc/os-release
  printf '%s\n' "$VERSION_ID" > "$work/raw/release.txt"
  printf '%s\n' "$ID" > "$work/raw/distribution.txt"
  cat /etc/os-release > "$work/raw/os-release.txt"
  getconf GNU_LIBC_VERSION > "$work/raw/c-runtime.txt"
  dpkg-query -W -f='${Package} ${Version}\n' libc6 libstdc++6 > "$work/raw/cxx-runtime.txt"
  stat -f -c '%T' "$work" > "$work/raw/filesystem.txt"
  findmnt -T "$work" -o FSTYPE,OPTIONS -n > "$work/raw/mount.txt"
  systemd-detect-virt > "$work/raw/virtualization.txt" || true
  if [ -r /sys/firmware/devicetree/base/model ]; then tr '\000' '\n' < /sys/firmware/devicetree/base/model > "$work/raw/hardware.txt";
  elif [ -r /sys/class/dmi/id/product_name ]; then cat /sys/class/dmi/id/product_name > "$work/raw/hardware.txt";
  else printf 'unavailable\n' > "$work/raw/hardware.txt"; fi
  if [ "$profile" = dgx-spark-dgxos7-arm64 ]; then
    test -r /etc/dgx-release
    cat /etc/dgx-release > "$work/raw/dgx-release.txt"
    nvidia-smi --query-gpu=name,driver_version --format=csv,noheader > "$work/raw/gpu.txt"
    nvidia-smi > "$work/raw/nvidia-smi.txt"
    if command -v nvcc >/dev/null 2>&1; then nvcc --version > "$work/raw/cuda.txt"; else printf 'nvcc unavailable; driver CUDA compatibility is in nvidia-smi.txt\n' > "$work/raw/cuda.txt"; fi
  fi
fi
cmake -DPLATFORM_REPOSITORY="$repository" -DPLATFORM_PROFILE="$profile" -DPLATFORM_WORK="$work" \
  -P "$repository/tests/platform-qualification/capture.cmake"
