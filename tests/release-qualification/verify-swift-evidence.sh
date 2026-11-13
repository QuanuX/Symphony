#!/bin/sh
# SwiftPM can emit results-swift-testing.xml separately from XCTest results.xml.
# Never convert a missing XML path (grep exit 2) into a no-skips success.
set -eu
[ "$#" -eq 1 ] || { echo 'usage: verify-swift-evidence.sh /absolute/log-directory' >&2; exit 2; }
case "$1" in /*) ;; *) echo 'absolute log directory required' >&2; exit 2;; esac
logs=$1
fail() { echo 'Swift qualification requires complete passing execution log and XML evidence' >&2; exit 1; }
[ -f "$logs/tests.log" ] && [ ! -L "$logs/tests.log" ] && [ -s "$logs/tests.log" ] || fail
grep -Eq 'Test run with [1-9][0-9]* tests?( in [1-9][0-9]* suites?)? passed' "$logs/tests.log" || fail
if grep -Eq 'Test .* skipped' "$logs/tests.log"; then fail; fi
command -v xmllint >/dev/null 2>&1 || fail
total=0
reports=0
for xml in "$logs/results.xml" "$logs/results-swift-testing.xml"; do
  if [ ! -e "$xml" ] && [ ! -L "$xml" ]; then continue; fi
  [ -f "$xml" ] && [ ! -L "$xml" ] && [ -s "$xml" ] || fail
  [ "$(wc -c < "$xml" | tr -d ' ')" -le 16777216 ] || fail
  # --nonet disables external network retrieval; do not enable entity expansion.
  root=$(xmllint --nonet --xpath 'name(/*)' "$xml" 2>/dev/null) || fail
  case "$root" in testsuites|testsuite) ;; *) fail;; esac
  bad=$(xmllint --nonet --xpath 'count(//failure | //error | //skipped | //testcase[not(ancestor::testsuite)] | //testsuite[not(@tests) or number(@tests) != count(.//testcase) or (@failures and number(@failures) != 0) or (@errors and number(@errors) != 0) or (@skipped and number(@skipped) != 0)])' "$xml" 2>/dev/null) || fail
  [ "$bad" = 0 ] || fail
  count=$(xmllint --nonet --xpath 'count(//testcase)' "$xml" 2>/dev/null) || fail
  [ "$count" -ge 0 ] || fail
  total=$((total + count))
  reports=$((reports + 1))
done
[ "$reports" -gt 0 ] && [ "$total" -gt 0 ] || fail
printf 'Verified Swift XML evidence: %s reports, %s test cases\n' "$reports" "$total"
