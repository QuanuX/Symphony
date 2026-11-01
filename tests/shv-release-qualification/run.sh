#!/bin/sh
# Required installed-owner qualification. Missing inputs and skips fail the gate.
set -eu
umask 077
if [ "$#" -ne 2 ]; then echo "usage: $0 /absolute/installed-prefix /absolute/fresh-work" >&2; exit 2; fi
case "$1:$2" in /*:/*) ;; *) echo 'absolute paths required' >&2; exit 2 ;; esac
if [ -e "$2" ] || [ -L "$2" ]; then echo 'qualification work must be fresh' >&2; exit 2; fi
repository=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
prefix=$(CDPATH= cd -- "$1" && pwd -P)
mkdir -m 700 "$2"
work=$(CDPATH= cd -- "$2" && pwd -P)
mkdir "$work/logs" "$work/bin"
export GOTOOLCHAIN=local GOENV=off GOFLAGS= GOEXPERIMENT=
export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org
export GOPRIVATE= GONOPROXY=none GONOSUMDB=none GOINSECURE= GOVCS='*:off' GOAUTH=off
if [ "$(go env GOVERSION)" != 'go1.26.5' ]; then echo 'Go 1.26.5 is required' >&2; exit 2; fi
# This campaign deliberately tests current composed source, including private
# SSIAG/STAV owners. Independent public module closure is a separate L05 gate.
export GOWORK="$repository/go.work"
export GOMODCACHE="${GOMODCACHE:-$work/go-modules}" GOCACHE="${GOCACHE:-$work/go-build}"
go env -json GOVERSION GOOS GOARCH GOPROXY GOSUMDB GOWORK GOMODCACHE GOCACHE GOFLAGS GOEXPERIMENT GOPRIVATE GONOPROXY GONOSUMDB GOINSECURE GOVCS GOAUTH > "$work/go-environment.json"
git -C "$repository" rev-parse HEAD 'HEAD^{tree}' > "$work/source.txt"
git -C "$repository" status --porcelain=v1 > "$work/source-status.txt"
if [ -n "${SHV_RELEASE_PDF_REQUEST:-}" ]; then
  echo "The required gate prepares its own pinned request; use SHV_RELEASE_SOURCE and SHV_RELEASE_ARCHIVE for retained exact inputs" >&2; exit 2
fi
  set -- -work "$work/assets" -lock "$repository/tests/shv-release-qualification/dependencies.json"
  if [ -n "${SHV_RELEASE_SOURCE:-}" ]; then set -- "$@" -source "$SHV_RELEASE_SOURCE"; fi
  if [ -n "${SHV_RELEASE_ARCHIVE:-}" ]; then set -- "$@" -archive "$SHV_RELEASE_ARCHIVE"; fi
  go run "$repository/tests/shv-release-qualification/prepare.go" "$@" > "$work/logs/prepare.log" 2>&1
  SHV_RELEASE_PDF_REQUEST="$work/assets/request.json"
export SHV_RELEASE_PDF_REQUEST
if [ -z "${SHV_RELEASE_STORE_PREFIX:-}" ]; then
  sh "$repository/.github/scripts/prepare-duckdb-dependency.sh" "$work/duckdb" > "$work/logs/duckdb.log" 2>&1
  cmake -S "$repository/modules/shv-graph-duckdb-connector" -B "$work/store-build" \
    -C "$work/duckdb/dependency.cmake" -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=OFF \
    -DSYMPHONY_KVE_USE_INSTALLED=ON "-DCMAKE_PREFIX_PATH=$prefix" \
    "-DCMAKE_INSTALL_PREFIX=$work/store-prefix" > "$work/logs/store-configure.log" 2>&1
  cmake --build "$work/store-build" --parallel 2 > "$work/logs/store-build.log" 2>&1
  cmake --install "$work/store-build" > "$work/logs/store-install.log" 2>&1
  SHV_RELEASE_STORE_PREFIX="$work/store-prefix"
fi
export SHV_RELEASE_STORE_PREFIX
# Build only the current copied-input installed replay fixture; the selected
# product installation is checked by qxctl and receipt-v2, never source fallback.
cmake -S "$repository/modules/shv-pdf-adapter" -B "$work/pdf-test-build" \
  -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=ON -DSYMPHONY_KVE_USE_INSTALLED=ON \
  "-DCMAKE_PREFIX_PATH=$prefix" > "$work/logs/pdf-test-configure.log" 2>&1
cmake --build "$work/pdf-test-build" --target shv-pdf-adapter-installed-integration --parallel 2 > "$work/logs/pdf-test-build.log" 2>&1
(cd "$repository" && go build -o "$work/bin/qxctl" ./tools/qxctl/cmd/qxctl && \
 go build -o "$work/bin/ssiag" ./modules/secure-identity-access-governance/cmd/symphony-ssiag && \
 go build -o "$work/bin/stav" ./modules/stav-append-authority/cmd/symphony-stav-append-authority) > "$work/logs/go-build.log" 2>&1
for module in tools/qxctl modules/secure-identity-access-governance modules/stav-append-authority; do
  (cd "$repository/$module" && go mod verify) >> "$work/logs/go-mod-verify.log" 2>&1
done
"$work/pdf-test-build/shv-pdf-adapter-installed-integration" \
  --qxctl "$work/bin/qxctl" --prefix "$prefix" --version 0.3.0-dev \
  --request "$SHV_RELEASE_PDF_REQUEST" --output "$work/pdf-replay" > "$work/logs/pdf-replay.log" 2>&1
export SHV_RELEASE_PREFIX="$prefix" SHV_RELEASE_EVIDENCE="$work/catalogue"
export SHV_RELEASE_SSIAG="$work/bin/ssiag" SHV_RELEASE_STAV="$work/bin/stav"
(cd "$repository/tools/qxctl" && go test -race -count=1 -json ./cmd/qxctl \
  -run '^TestInstalledSHVBoundedCatalogueWorkflow$' -timeout 5m) > "$work/logs/catalogue.jsonl" 2>&1
# Validate the actual event stream, not just the test process exit status.
go run "$repository/tests/shv-release-qualification/verify.go" \
  -work "$work" -prefix "$prefix" -store-prefix "$SHV_RELEASE_STORE_PREFIX" \
  -request "$SHV_RELEASE_PDF_REQUEST" -repository "$repository" > "$work/logs/verify.log" 2>&1
cat "$work/logs/verify.log"
