#!/bin/sh
# Read-only installed evidence plus private notice copies; no network/publication.
set -eu
umask 077
[ "$#" -eq 4 ] || { echo 'usage: run.sh PREFIX FRESH_WORK OPTIONAL_EXTERNAL OPTIONAL_GO_ARTIFACT' >&2; exit 2; }
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
prefix=$(CDPATH= cd -- "$1" && pwd -P)
case "$2" in /*) ;; *) echo 'absolute work required' >&2; exit 2;; esac
[ ! -e "$2" ] || { echo 'fresh work required' >&2; exit 2; }
parent=$(CDPATH= cd -- "$(dirname -- "$2")" && pwd -P)
work="$parent/$(basename -- "$2")"
case "$work/" in "$prefix/"*|"$repo/"*) echo 'evidence work must be outside installation/source' >&2; exit 2;; esac
mkdir -p "$work/logs" "$work/notices"
export GOWORK=off GOTOOLCHAIN=local GOENV=off GOFLAGS= GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOAUTH=off GOPROXY=off GOSUMDB=off
[ "$(go env GOVERSION)" = "go$(sed -n 's/^go //p' "$repo/tools/qxctl/go.mod")" ] || { echo 'exact Go toolchain required' >&2; exit 2; }
cmake -DNOTICE_TEST_WORK="$work/regressions" -P "$repo/tests/dependency-notices/regressions.cmake" > "$work/logs/cmake-regressions.log" 2>&1
go test -count=1 "$repo/tests/dependency-notices/inventory.go" "$repo/tests/dependency-notices/inventory_test.go" > "$work/logs/go-regressions.log" 2>&1
if [ -f "$4" ]; then
 (cd "$repo/tools/qxctl" && go mod verify) > "$work/logs/go-module-verification.log" 2>&1
fi
go run "$repo/tests/dependency-notices/inventory.go" "$repo" "$prefix" "$work" "$3" "$4" "$(go env GOMODCACHE)" > "$work/logs/inventory.log" 2>&1
cat "$work/logs/inventory.log"
