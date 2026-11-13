#!/bin/sh
set -eu
umask 077
here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
work=$(mktemp -d "${TMPDIR:-/tmp}/symphony-swift-evidence.XXXXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
pass=0
fixture() {
  mkdir "$work/$1"
  printf '%s\n' 'Test run with 1 test in 1 suite passed after 0.001 seconds.' > "$work/$1/tests.log"
}
check() {
  expected=$2
  if sh "$here/verify-swift-evidence.sh" "$work/$1" > "$work/$1/result.log" 2>&1; then actual=pass; else actual=refused; fi
  [ "$actual" = "$expected" ] || { echo "Unexpected Swift evidence outcome: $1" >&2; cat "$work/$1/result.log" >&2; exit 1; }
  pass=$((pass + 1))
}
valid='<testsuites><testsuite tests="1" failures="0" errors="0" skipped="0"><testcase name="one"/></testsuite></testsuites>'
fixture suffixed; printf '%s\n' "$valid" > "$work/suffixed/results-swift-testing.xml"; check suffixed pass
fixture unsuffixed; printf '%s\n' "$valid" > "$work/unsuffixed/results.xml"; check unsuffixed pass
fixture mixed; printf '%s\n' "$valid" > "$work/mixed/results-swift-testing.xml"; printf '%s\n' '<testsuite tests="0" failures="0"/>' > "$work/mixed/results.xml"; check mixed pass
fixture missing; check missing refused
fixture empty; : > "$work/empty/results.xml"; check empty refused
fixture zero; printf '%s\n' '<testsuite tests="0"/>' > "$work/zero/results.xml"; check zero refused
fixture malformed; printf '%s\n' '<testsuite' > "$work/malformed/results.xml"; check malformed refused
fixture skipped; printf '%s\n' '<testsuite tests="1" skipped="1"><testcase><skipped/></testcase></testsuite>' > "$work/skipped/results-swift-testing.xml"; check skipped refused
fixture failed; printf '%s\n' '<testsuite tests="1" failures="1"><testcase><failure/></testcase></testsuite>' > "$work/failed/results-swift-testing.xml"; check failed refused
fixture error; printf '%s\n' '<testsuite tests="1" errors="1"><testcase><error/></testcase></testsuite>' > "$work/error/results.xml"; check error refused
fixture count; printf '%s\n' '<testsuite tests="2"><testcase/></testsuite>' > "$work/count/results.xml"; check count refused
fixture lie; printf '%s\n' '<testsuite tests="1" failures="0"><testcase><failure/></testcase></testsuite>' > "$work/lie/results.xml"; check lie refused
fixture attributes; printf '%s\n' '<testsuite tests="1" errors="NaN"><testcase/></testsuite>' > "$work/attributes/results.xml"; check attributes refused
fixture root; printf '%s\n' '<unexpected><testcase/></unexpected>' > "$work/root/results.xml"; check root refused
fixture link; ln -s "$work/suffixed/results-swift-testing.xml" "$work/link/results.xml"; check link refused
fixture hidden; printf '%s\n' "$valid" > "$work/hidden/results-swift-testing.xml"; printf '%s\n' '<testsuite tests="1" skipped="1"><testcase/></testsuite>' > "$work/hidden/results.xml"; check hidden refused
fixture no_suite; printf '%s\n' '<testsuites><testcase/></testsuites>' > "$work/no_suite/results.xml"; check no_suite refused
fixture log; printf '%s\n' "$valid" > "$work/log/results.xml"; printf '%s\n' 'No Swift tests ran' > "$work/log/tests.log"; check log refused
printf 'Swift evidence regression cases passed: %s\n' "$pass"
