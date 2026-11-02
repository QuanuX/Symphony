#!/bin/sh
# Exercise the real CLI receipt verifier before connector lifecycle removal.
set -eu
umask 077
[ "$#" -eq 2 ] || { echo 'usage: qxctl-admission.sh /absolute/prefix /absolute/fresh-work' >&2; exit 2; }
case "$1:$2" in /*:/*) ;; *) echo 'absolute paths required' >&2; exit 2;; esac
[ ! -e "$2" ] && [ ! -L "$2" ] || { echo 'fresh work required' >&2; exit 2; }
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
prefix=$(CDPATH= cd -- "$1" && pwd -P)
mkdir -m 700 "$2"
work=$(CDPATH= cd -- "$2" && pwd -P)
export GOWORK=off GOTOOLCHAIN=local GOENV=off GOFLAGS= GOEXPERIMENT=
export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org
export GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOAUTH=off GOVCS='*:off'
export GOCACHE="$work/build-cache" GOMODCACHE="$work/module-cache" GOPATH="$work/go-path"
cmake -P "$repo/tests/release-qualification/require-go-toolchain.cmake"
cp -R "$repo/tools/qxctl" "$work/qxctl-source"
go env -json GOVERSION GOOS GOARCH GOWORK GOPROXY GOSUMDB GOFLAGS GOMODCACHE GOCACHE > "$work/environment.json"
(cd "$work/qxctl-source" && go build -mod=readonly -trimpath -o "$work/qxctl" ./cmd/qxctl && go mod verify) > "$work/build.log" 2>&1
cmp "$repo/tools/qxctl/go.mod" "$work/qxctl-source/go.mod"
cmp "$repo/tools/qxctl/go.sum" "$work/qxctl-source/go.sum"
"$work/qxctl" shv graph store inspect --connector-prefix "$prefix" --connector-version 0.4.0-dev --json > "$work/shv.json"
"$work/qxctl" scv graph-index inspect --connector-prefix "$prefix" --connector-version 0.2.0-dev --json > "$work/scv.json"
cmake "-DADMISSION_WORK=$work" "-DADMISSION_PREFIX=$prefix" "-DADMISSION_REPOSITORY=$repo" -P "$repo/tests/duckdb-connectors/qxctl-admission.cmake"
