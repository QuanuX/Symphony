#!/bin/sh
# Installed native execution only. No live provider or credential requests.
set -eu
umask 077
[ "$#" -eq 2 ] || { echo 'usage: run.sh /absolute/installed-prefix /absolute/fresh-work' >&2; exit 2; }
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
prefix=$(CDPATH= cd -- "$1" && pwd -P)
case "$2" in /*) ;; *) echo 'absolute fresh work required' >&2; exit 2;; esac
[ ! -e "$2" ] || { echo 'work must not exist' >&2; exit 2; }
mkdir -p "$2"
work=$(CDPATH= cd -- "$2" && pwd -P)
case "$work/" in "$repo/"*) echo 'qualification work must be outside source checkout' >&2; exit 2;; esac
mkdir -p "$work/logs" "$work/outputs" "$work/bin" "$work/source" "$work/tmp" "$work/go-build" "$work/go-modules"
printf '%s\n' '{"format_version":1,"status":"pending","qualification":"sbv-mvp-installed-process","live_data":false,"network_transport":false}' > "$work/evidence.pending.json"
export GOFLAGS= GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOAUTH=off
export GOWORK=off GOTOOLCHAIN=local GOENV=off GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org
export GOCACHE="$work/go-build" GOMODCACHE="$work/go-modules" TMPDIR="$work/tmp"
# Explicit optional cache seed is copied, never changes the caller's cache.
if [ -n "${SBV_QUALIFICATION_MODULE_SEED:-}" ]; then cp -R "$SBV_QUALIFICATION_MODULE_SEED/." "$GOMODCACHE/"; fi
expected=$(sed -n 's/^go //p' "$repo/tools/qxctl/go.mod")
[ "$(go env GOVERSION)" = "go$expected" ] || { echo 'exact qxctl Go toolchain required' >&2; exit 2; }
cp "$repo/tests/sbv-launch-boundary/"*.go "$work/source/"
cp "$repo/tests/sbv-launch-boundary/"*.cpp "$repo/tests/sbv-launch-boundary/CMakeLists.txt" "$work/source/"
cp "$repo/modules/sbv-engine/tests/feedback_fixture.cpp" "$work/source/"
cp "$repo/modules/sqav-databento-dbn-cpp/tests/public_fixture.hpp" "$work/source/"
cp "$repo/modules/sqav-databento-dbn-cpp/tests/third_party/README.md" "$work/outputs/fixture-attribution.md"
cp "$repo/modules/sqav-databento-dbn-cpp/tests/third_party/LICENSE.Apache-2.0" "$work/outputs/fixture-LICENSE.Apache-2.0"
# Copy entire Go modules outside checkout so neither build can borrow go.work.
cp -R "$repo/tools/qxctl" "$work/qxctl-source"
cp -R "$repo/tools/sov-stdio-adapter" "$work/adapter-source"
go env -json GOVERSION GOWORK GOTOOLCHAIN GOENV GOAUTH GOFLAGS GOPROXY GOSUMDB GOPRIVATE GONOPROXY GONOSUMDB GOINSECURE GOMODCACHE GOCACHE > "$work/outputs/build-environment.json"
(cd "$work/qxctl-source" && go mod verify) >"$work/logs/verify-qxctl-before.log" 2>&1
(cd "$work/adapter-source" && go mod verify) >"$work/logs/verify-adapter-before.log" 2>&1
(cd "$work/qxctl-source" && go build -mod=readonly -trimpath -o "$work/bin/qxctl" ./cmd/qxctl) >"$work/logs/build-qxctl.log" 2>&1
(cd "$work/adapter-source" && go build -mod=readonly -trimpath -o "$work/bin/sov-stdio-adapter" .) >"$work/logs/build-adapter.log" 2>&1
(cd "$work/qxctl-source" && go mod verify) >"$work/logs/verify-qxctl-after.log" 2>&1
(cd "$work/adapter-source" && go mod verify) >"$work/logs/verify-adapter-after.log" 2>&1
cmp "$repo/tools/qxctl/go.mod" "$work/qxctl-source/go.mod"
cmp "$repo/tools/qxctl/go.sum" "$work/qxctl-source/go.sum"
cmp "$repo/tools/sov-stdio-adapter/go.mod" "$work/adapter-source/go.mod"
"$work/bin/qxctl" --version > "$work/outputs/qxctl-version.txt"
"$work/bin/sov-stdio-adapter" --version > "$work/outputs/adapter-version.txt"
[ "$(cat "$work/outputs/qxctl-version.txt")" = 'qxctl version 0.1.0-dev' ]
[ "$(cat "$work/outputs/adapter-version.txt")" = 'sov-stdio-adapter version 0.1.0-dev' ]
(cd "$work/source" && go test -count=1 main.go main_test.go) >"$work/logs/harness-tests.log" 2>&1
(cd "$work/source" && go build -trimpath -o "$work/bin/qualify" main.go) >"$work/logs/build-driver.log" 2>&1
cmake -S "$work/source" -B "$work/fixtures" -DSBV_PREFIX="$prefix" >"$work/logs/configure-fixtures.log" 2>&1
cmake --build "$work/fixtures" --parallel 2 >"$work/logs/build-fixtures.log" 2>&1
"$work/fixtures/launch-data" "$work/outputs/fixture.dbn"
git -C "$repo" rev-parse HEAD > "$work/outputs/source-commit.txt"
git -C "$repo" diff --stat > "$work/outputs/source-working-diff.txt"
uname -a > "$work/outputs/host.txt"
go version > "$work/outputs/go-version.txt"
cmake --version > "$work/outputs/cmake-version.txt"
"${CXX:-c++}" --version > "$work/outputs/compiler-version.txt"
if [ "$(uname -s)" = Darwin ]; then sw_vers > "$work/outputs/os-release.txt"; else
  cat /etc/os-release > "$work/outputs/os-release.txt"
  getconf GNU_LIBC_VERSION > "$work/outputs/libc-version.txt"
fi
"$work/bin/qualify" "$prefix" "$work" >"$work/logs/qualification.log" 2>&1
cat "$work/logs/qualification.log"
