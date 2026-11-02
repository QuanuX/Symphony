#!/bin/sh
set -eu
umask 077
if [ "$#" -ne 1 ]; then echo "usage: $0 /absolute/fresh-work" >&2; exit 2; fi
work=$1
case "$work" in /*) ;; *) echo 'absolute work path required' >&2; exit 2 ;; esac
if [ -e "$work" ] || [ -L "$work" ]; then echo 'work path must be fresh' >&2; exit 2; fi
here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo=$(CDPATH= cd -- "$here/../.." && pwd -P)
mkdir -m 700 "$work"
work=$(CDPATH= cd -- "$work" && pwd -P)
mkdir "$work/modules" "$work/build" "$work/path" "$work/tmp" "$work/telemetry"
printf 'off\n' > "$work/telemetry/mode"
export GOMODCACHE="$work/modules" GOCACHE="$work/build" GOPATH="$work/path" GOTMPDIR="$work/tmp"
export TMPDIR="$work/tmp" TEST_TELEMETRY_DIR="$work/telemetry"
export GOENV=off GOWORK=off GOTOOLCHAIN=local GO111MODULE=on CGO_ENABLED=1
export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GOAUTH=off GOVCS='*:off'
export GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOFLAGS=
unset GOOS GOARCH GOEXPERIMENT GOCACHEPROG
[ "$(go env GOVERSION)" = go1.26.5 ] || { echo 'exact Go1.26.5 required' >&2; exit 2; }
{
  uname -srm
  go version
  git -C "$repo" rev-parse HEAD
  git -C "$repo" rev-parse 'HEAD^{tree}'
} > "$work/host-info.log"
cmake -DMVP_REPOSITORY="$repo" -P "$here/verify.cmake"
go test "$here/go-runner.go" "$here/go-runner_test.go"
go run "$here/go-runner.go" --repo "$repo" --manifest "$here/manifest.json" \
  --work "$work/logs" --mode full
git -C "$repo" diff --exit-code
