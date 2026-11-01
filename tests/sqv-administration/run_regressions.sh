#!/bin/sh
set -eu
umask 077

if [ "$#" -lt 1 ]; then
  echo "usage: $0 /absolute/fresh/private/work-directory [CMake configure arguments...]" >&2
  exit 2
fi
work=$1
shift
case "$work" in /*) ;; *) echo 'work directory must be absolute' >&2; exit 2 ;; esac
if [ -e "$work" ] || [ -L "$work" ]; then
  echo 'work directory must not already exist' >&2
  exit 2
fi
repository=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
mkdir -m 700 "$work"
work=$(CDPATH= cd -- "$work" && pwd -P)
mkdir "$work/logs" "$work/tmp" "$work/bin"

# Canonical, private temporary paths are part of the native store contract.
export TMPDIR="$work/tmp"
export SYMPHONY_TEST_TMPDIR="$work/tmp"
export GOWORK=off GOTOOLCHAIN=local GOENV=off
export GOMODCACHE="${GOMODCACHE:-$work/go/modules}"
export GOCACHE="${GOCACHE:-$work/go/build}"
export GOPATH="${GOPATH:-$work/go/path}"
export GOTMPDIR="$work/tmp"
mkdir -p "$GOMODCACHE" "$GOCACHE" "$GOPATH"
if [ "$(go env GOVERSION)" != go1.26.5 ]; then
  echo 'this qualification requires Go 1.26.5 on PATH' >&2
  exit 2
fi
jobs=${SQV_ADMIN_BUILD_JOBS:-2}
case "$jobs" in ''|*[!0-9]*|0) echo 'SQV_ADMIN_BUILD_JOBS must be a positive integer' >&2; exit 2 ;; esac

run_step() {
  label=$1
  shift
  printf '\nRunning %s\n' "$label"
  if "$@" > "$work/logs/$label.log" 2>&1; then
    cat "$work/logs/$label.log"
  else
    status=$?
    cat "$work/logs/$label.log" >&2
    printf 'FAILED %s (exit %s); artifacts retained in %s\n' "$label" "$status" "$work" >&2
    exit "$status"
  fi
}

host_info() {
  uname -srm
  if [ "$(uname -s)" = Darwin ]; then
    sw_vers
    echo 'The macOS native store gate below requires local APFS.'
  else
    stat -f -c 'Selected fixture filesystem: %T' "$work"
    if [ -r /etc/os-release ]; then cat /etc/os-release; fi
  fi
  df -h "$work"
  go version
  cmake --version
  "${CXX:-c++}" --version
  printf 'Source commit: '
  git -C "$repository" rev-parse HEAD
  git -C "$repository" status --short
}
run_step host-info host_info
run_step configure cmake -S "$repository/tests/sqv-administration" -B "$work/build" \
  -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=ON -DCMAKE_INSTALL_PREFIX="$work/prefix" "$@"
run_step build cmake --build "$work/build" --parallel "$jobs"
run_step native-tests ctest --test-dir "$work/build" --output-on-failure --no-tests=error \
  -R '^(metadata_validate|flow_validate|conversion_validate|read_only_store_boundary)$'
run_step install cmake --install "$work/build"
run_step writer-fixtures "$work/build/sqv-admin-fixtures" "$work/fixtures"

# These installed process tests are executable targets, not CTest registrations.
for binding in \
  sqmv-metadata-engine:metadata_validate \
  sqfv-flow-engine:flow_validate \
  sqtv-conversion-engine:conversion_validate \
  sqpv-inspection-engine:store \
  sqdv-checkpoint-engine:checkpoint \
  sqav-attempt-engine:attempts
do
  module=${binding%%:*}
  fixture=${binding#*:}
  case "$fixture" in
    *_validate) input="$repository/modules/$module/tests/fixtures/$fixture.json" ;;
    *) input="$work/fixtures/$fixture.json" ;;
  esac
  run_step "$module-process" "$work/build/$module/$module-process-test" \
    --prefix "$work/prefix" --input "$input"
done

cd "$repository/tools/qxctl"
run_step qxctl-build go build -mod=readonly -trimpath -o "$work/bin/qxctl" ./cmd/qxctl
export SQV_ADMIN_TEST_PREFIX="$work/prefix" SQV_ADMIN_FIXTURES="$work/fixtures"
export SQV_ADMIN_TEST_QXCTL="$work/bin/qxctl" SQV_ADMIN_REQUIRE_INSTALLED=1
run_step installed-go-tests go test -mod=readonly -race -count=1 -v \
  -run '^TestSQV(AdministrationInstalled|TransportShape)$' ./internal/knowledgeengine
run_step command-surface-tests go test -mod=readonly -race -count=1 -v \
  -run '^TestSQVCanonicalSurface$' ./cmd/qxctl
if grep -q -- '--- SKIP:' "$work/logs/installed-go-tests.log" "$work/logs/command-surface-tests.log"; then
  echo 'installed qualification must not skip selected tests' >&2
  exit 1
fi
printf '\nPASS installed SQV retained-state qualification; artifacts: %s\n' "$work"
