#!/bin/sh
# Build only the qxctl module, outside its checkout, using public dependencies.
set -eu
umask 077

fail() { printf '%s\n' "$*" >&2; exit 1; }

if [ "$#" -gt 1 ]; then
  fail "usage: $0 [path-to-qxctl-module]"
fi
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
source_module=$(CDPATH= cd -- "${1:-$script_dir/..}" && pwd -P)
[ -f "$source_module/go.mod" ] && [ -f "$source_module/go.sum" ] ||
  fail "qxctl go.mod and go.sum are required"
[ "$(awk '$1 == "module" { print $2 }' "$source_module/go.mod")" = \
  'github.com/QuanuX/Symphony/tools/qxctl' ] || fail "expected the qxctl module"
required_go=$(awk '$1 == "go" { print $2 }' "$source_module/go.mod")
[ -n "$required_go" ] || fail "go.mod must declare the exact Go baseline"
command -v go >/dev/null 2>&1 || fail "install Go $required_go and put it on PATH"
if command -v sha256sum >/dev/null 2>&1; then
  hash_program=sha256sum
elif command -v shasum >/dev/null 2>&1; then
  hash_program=shasum
else
  fail "sha256sum or shasum is required"
fi
module_hashes() (
  CDPATH= cd -- "$1" || exit 1
  if [ "$hash_program" = sha256sum ]; then
    sha256sum go.mod go.sum
  else
    shasum -a 256 go.mod go.sum
  fi
)
original_hashes=$(module_hashes "$source_module")
source_checkout=$(git -C "$source_module" rev-parse --show-toplevel 2>/dev/null || true)
if [ -n "$source_checkout" ]; then
  source_checkout=$(CDPATH= cd -- "$source_checkout" && pwd -P)
fi
temp_base=$(CDPATH= cd -- "${TMPDIR:-/tmp}" && pwd -P)
work_dir=$(mktemp -d "$temp_base/symphony-qxctl-standalone.XXXXXX")
module_copy="$work_dir/qxctl"
copy_complete=false
cleanup() {
  status=$?
  trap - 0 HUP INT TERM
  if ! current_hashes=$(module_hashes "$source_module") || [ "$current_hashes" != "$original_hashes" ]; then
    printf '%s\n' 'ERROR: source go.mod or go.sum changed during the standalone check' >&2
    status=1
  fi
  if [ "$copy_complete" = true ]; then
    if ! current_hashes=$(module_hashes "$module_copy") || [ "$current_hashes" != "$original_hashes" ]; then
      printf '%s\n' 'ERROR: standalone check changed copied go.mod or go.sum' >&2
      status=1
    fi
  fi
  # Go's module cache contains read-only directories. Touch only our mktemp tree.
  case "$work_dir" in
    "$temp_base"/symphony-qxctl-standalone.*)
      chmod -R u+w "$work_dir" || status=1
      rm -rf -- "$work_dir" || status=1
      ;;
    *) printf '%s\n' 'ERROR: refusing cleanup outside the generated temporary tree' >&2; status=1 ;;
  esac
  exit "$status"
}
trap cleanup 0
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
case "$work_dir/" in
  "$source_module/"*) fail "TMPDIR must be outside the qxctl source module" ;;
esac
if [ -n "$source_checkout" ]; then
  case "$work_dir/" in
    "$source_checkout/"*) fail "TMPDIR must be outside the source checkout" ;;
  esac
fi
mkdir -p "$module_copy" "$work_dir/module-cache" "$work_dir/build-cache" \
  "$work_dir/gopath" "$work_dir/go-tmp" "$work_dir/telemetry" "$work_dir/bin" "$work_dir/run"
cp -R "$source_module/." "$module_copy/"
copy_complete=true
# A source symlink could silently reconnect the isolated copy to the checkout.
if [ -n "$(find "$module_copy" -type l -print)" ]; then
  fail "the standalone qxctl source copy must not contain symlinks"
fi

export GOENV=off GOWORK=off GOTOOLCHAIN=local GO111MODULE=on
export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org
export GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOFLAGS= GOAUTH=off
export GOVCS='*:off'
export GOMODCACHE="$work_dir/module-cache" GOCACHE="$work_dir/build-cache"
export GOPATH="$work_dir/gopath" GOTMPDIR="$work_dir/go-tmp"
# The pinned Go toolchain supports this test-only telemetry-directory override.
# Keep its counters/configuration inside the same disposable tree.
export TEST_TELEMETRY_DIR="$work_dir/telemetry"
printf 'off\n' > "$TEST_TELEMETRY_DIR/mode"
unset GOOS GOARCH GOEXPERIMENT GOCACHEPROG
CDPATH= cd -- "$module_copy"
actual_go=$(go env GOVERSION)
[ "$actual_go" = "go$required_go" ] ||
  fail "Go $required_go is required; found $actual_go (automatic toolchain downloads are disabled)"
printf 'Standalone qxctl check: %s on %s/%s\n' "$actual_go" "$(go env GOHOSTOS)" "$(go env GOHOSTARCH)"

# -mod=readonly makes missing/stale committed dependency metadata a failure.
# Check all selected modules, including replacements with remote targets.
go list -mod=readonly -m -f '{{if .Replace}}{{.Path}} => {{.Replace.Path}} {{.Replace.Version}}{{end}}' all > "$work_dir/replacements.txt"
if grep -q '[^[:space:]]' "$work_dir/replacements.txt"; then
  cat "$work_dir/replacements.txt" >&2
  fail "standalone builds must not select module replacements"
fi
go build -mod=readonly -trimpath -o "$work_dir/bin/qxctl" ./cmd/qxctl
go mod verify
CDPATH= cd -- "$work_dir/run"
"$work_dir/bin/qxctl" --version > "$work_dir/version.txt"
"$work_dir/bin/qxctl" --help > "$work_dir/help.txt"
grep -q '^qxctl version ' "$work_dir/version.txt" || fail "unexpected qxctl --version output"
[ -s "$work_dir/help.txt" ] || fail "qxctl --help produced no output"
cat "$work_dir/version.txt"
printf '%s\n' 'Standalone qxctl build and smoke checks passed; module hashes are checked again on exit.'
