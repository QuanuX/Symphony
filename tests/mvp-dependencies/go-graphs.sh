#!/bin/sh
set -eu
umask 077
if [ "$#" -ne 1 ]; then echo "usage: $0 /absolute/fresh-work" >&2; exit 2; fi
work=$1
case "$work" in /*) ;; *) echo 'work directory must be absolute' >&2; exit 2 ;; esac
if [ -e "$work" ] || [ -L "$work" ]; then echo 'work directory must not exist' >&2; exit 2; fi
harness=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repository=${MVP_REPOSITORY:-$(CDPATH= cd -- "$harness/../.." && pwd -P)}
mkdir -m 700 "$work"
work=$(CDPATH= cd -- "$work" && pwd -P)
mkdir "$work/logs" "$work/modules" "$work/build" "$work/path" "$work/tmp" "$work/telemetry"
printf 'off\n' > "$work/telemetry/mode"
export GOMODCACHE="$work/modules" GOCACHE="$work/build" GOPATH="$work/path" GOTMPDIR="$work/tmp"
export TEST_TELEMETRY_DIR="$work/telemetry"
export GOENV=off GOWORK=off GOTOOLCHAIN=local GO111MODULE=on
export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GOAUTH=off GOVCS='*:off'
export GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOFLAGS=
unset GOOS GOARCH GOEXPERIMENT GOCACHEPROG
if [ "$(go env GOVERSION)" != go1.26.5 ]; then echo 'Go1.26.5 is required on PATH' >&2; exit 2; fi
go version
cmake -DMVP_REPOSITORY="$repository" -DMVP_WORK="$work" -P "$harness/go-graphs.cmake"
