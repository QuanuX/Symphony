#!/bin/sh
set -eu
umask 077
if [ "$#" -lt 3 ] || [ "$#" -gt 4 ]; then
  echo 'usage: run.sh GROUP /absolute/mvp-installed /absolute/fresh-work [/absolute/prior-campaign]' >&2; exit 2
fi
group=$1
case "$group" in bootstrap|knowledge|cloud|quantitative) ;; *) echo 'unsupported SDK distribution group' >&2; exit 2 ;; esac
for path in "$2" "$3" "${4:-/}"; do case "$path" in /*) ;; *) echo 'absolute paths required' >&2; exit 2 ;; esac; done
if [ -e "$3" ] || [ -L "$3" ]; then echo 'work must be fresh' >&2; exit 2; fi
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
installed=$(CDPATH= cd -- "$2" && pwd -P)
mkdir -m 700 "$3"
work=$(CDPATH= cd -- "$3" && pwd -P)
case "$work/" in "$repo/"*|"$installed/"*) echo 'disposable work must be outside source and selected installation' >&2; exit 2 ;; esac
mkdir "$work/logs" "$work/tmp" "$work/metadata"
export GOWORK=off GOTOOLCHAIN=local GOENV=off GOFLAGS= GOEXPERIMENT=
export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOAUTH=off GOVCS='*:off'
export GOMODCACHE="$work/go/modules" GOCACHE="$work/go/build" GOPATH="$work/go/path" TMPDIR="$work/tmp"
cmake -P "$repo/tests/release-qualification/require-go-toolchain.cmake"
{
  uname -srm
  if [ "$(uname -s)" = Darwin ]; then sw_vers; else cat /etc/os-release; fi
  df -h "$work"
  go version
  cmake --version
  "${CXX:-c++}" --version
  git -C "$repo" rev-parse HEAD 'HEAD^{tree}'
  git -C "$repo" status --short
} > "$work/logs/host-info.log"
go test -race -count=1 -v "$repo/tests/native-sdk-distribution/main.go" "$repo/tests/native-sdk-distribution/main_test.go" > "$work/logs/harness-tests.log" 2>&1
go run "$repo/tests/native-sdk-distribution/main.go" --repo "$repo" --group "$group" --installed "$installed" --work "$work" --campaign "${4:-}" > "$work/logs/qualification.log" 2>&1
cat "$work/logs/qualification.log"
