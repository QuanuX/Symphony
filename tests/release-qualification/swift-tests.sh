#!/bin/sh
set -eu
umask 077
if [ "$#" -ne 1 ]; then echo "usage: $0 /absolute/fresh-work" >&2; exit 2; fi
[ "$(uname -s)" = Darwin ] || { echo 'Swift provider qualification requires macOS' >&2; exit 2; }
work=$1
case "$work" in /*) ;; *) echo 'absolute work path required' >&2; exit 2 ;; esac
if [ -e "$work" ] || [ -L "$work" ]; then echo 'work path must be fresh' >&2; exit 2; fi
here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo=$(CDPATH= cd -- "$here/../.." && pwd -P)
package="$repo/modules/ssiag-provider-macos-keychain"
mkdir -m 700 "$work"
work=$(CDPATH= cd -- "$work" && pwd -P)
mkdir "$work/logs" "$work/tmp"
export TMPDIR="$work/tmp"
{
  uname -srm
  sw_vers
  swift --version
  git -C "$repo" rev-parse HEAD
  git -C "$repo" rev-parse 'HEAD^{tree}'
} > "$work/logs/host-info.log"
swift build --package-path "$package" --scratch-path "$work/build" -c debug \
  --product symphony-ssiag-provider-macos-keychain > "$work/logs/build.log" 2>&1
binary_directory=$(swift build --package-path "$package" --scratch-path "$work/build" -c debug --show-bin-path)
export SYMPHONY_SSIAG_TEST_EXECUTABLE="$binary_directory/symphony-ssiag-provider-macos-keychain"
test -x "$SYMPHONY_SSIAG_TEST_EXECUTABLE"
swift test --package-path "$package" --scratch-path "$work/build" -c debug \
  --xunit-output "$work/logs/results.xml" > "$work/logs/tests.log" 2>&1
cat "$work/logs/tests.log"
grep -Eq 'Test run with [1-9][0-9]* tests?( in [1-9][0-9]* suites?)? passed' "$work/logs/tests.log" || {
  echo 'Swift Testing did not report executed passing tests' >&2; exit 1;
}
if grep -Eq 'Test .* skipped|<skipped[ >]' "$work/logs/tests.log" "$work/logs/results.xml"; then
  echo 'Selected Swift tests skipped' >&2; exit 1
fi
git -C "$repo" diff --exit-code
