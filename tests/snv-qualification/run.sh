#!/bin/sh
set -eu
umask 077
if [ "$#" -ne 3 ]; then
  echo 'usage: run.sh /absolute/installed-prefix /absolute/fresh-work /absolute/qxctl' >&2
  exit 2
fi
prefix=$1
work=$2
qxctl=$3
for path in "$prefix" "$work" "$qxctl"; do
  case "$path" in /*) ;; *) echo 'all selected paths must be absolute' >&2; exit 2 ;; esac
done
if [ ! -d "$prefix" ] || [ ! -x "$qxctl" ] || [ -e "$work" ] || [ -L "$work" ]; then
  echo 'require an installed prefix, executable qxctl and absent work directory' >&2
  exit 2
fi
harness=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo=$(CDPATH= cd -- "$harness/../.." && pwd -P)
prefix=$(CDPATH= cd -- "$prefix" && pwd -P)
mkdir -m 700 "$work"
work=$(CDPATH= cd -- "$work" && pwd -P)
mkdir "$work/logs" "$work/outputs" "$work/tmp" "$work/metadata"
export TMPDIR="$work/tmp" SYMPHONY_TEST_TMPDIR="$work/tmp"
export GOWORK=off GOTOOLCHAIN=local GOENV=off GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org
export GOMODCACHE="$work/go/modules" GOCACHE="$work/go/build" GOPATH="$work/go/path"
export GOAUTH=off GOVCS='*:off' GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOFLAGS=
{
  uname -srm
  if [ "$(uname -s)" = Linux ]; then
    cat /etc/os-release
    for source in /proc /sys /proc/meminfo /sys/devices/system/cpu/present /sys/devices/system/cpu/online; do
      stat -f -c '%n filesystem=%T' "$source" || echo "$source filesystem observation unavailable"
    done
    if command -v systemd-detect-virt >/dev/null 2>&1; then
      systemd-detect-virt || [ "$?" -eq 1 ]
    else
      echo 'virtualization classification unavailable'
    fi
    if [ -e /.dockerenv ] || [ -e /run/.containerenv ]; then echo 'container marker present'; else echo 'container marker absent; absence does not prove bare metal'; fi
  else
    sw_vers
    echo 'native observer unavailable on macOS; SDK injected-reader tests are separate'
  fi
  df -h "$work"
  cmake --version
  "${CXX:-c++}" --version
  go version
  git -C "$repo" rev-parse HEAD
  git -C "$repo" status --short
} > "$work/logs/host-info.log"
cmake -P "$repo/tests/release-qualification/require-go-toolchain.cmake"
cmake -DMVP_REPOSITORY="$repo" -DMVP_WORK="$work" -DMVP_SELECTED_PREFIX="$prefix" \
  -DSNV_QXCTL="$qxctl" -P "$harness/qualify.cmake"
