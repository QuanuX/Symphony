#!/bin/sh
# Read-only CLI first-run smoke; deliberately not final-download acceptance.
set -eu
umask 077
[ "$#" -eq 3 ] || { echo 'usage: source-smoke.sh ABSOLUTE_QXCTL ABSOLUTE_REPO FRESH_EVIDENCE' >&2; exit 2; }
for path in "$@"; do case "$path" in /*) ;; *) echo 'absolute paths required' >&2; exit 2;; esac; done
[ -x "$1" ] && [ -d "$2/knowledge" ] || exit 2
[ ! -e "$3" ] && [ ! -L "$3" ] || { echo 'evidence path must be new' >&2; exit 2; }
mkdir -m 700 "$3"
"$1" version --json > "$3/version.json"
"$1" knowledge invariant status --repo "$2" --json > "$3/invariant-status.json"
if "$1" knowledge invariant status --repo "$3/absent-repository" --json > "$3/refusal.stdout.txt" 2> "$3/refusal.stderr"; then
 echo 'missing repository was accepted' >&2; exit 1
else
 result=$?
 [ "$result" -eq 1 ] || { echo 'unexpected refusal exit status' >&2; exit 1; }
fi
grep -Fq 'repository root must identify a directory rather than a symlink' "$3/refusal.stdout.txt" || { echo 'expected repository admission diagnostic absent' >&2; exit 1; }
[ ! -e "$3/absent-repository" ] || { echo 'read-only refusal created repository' >&2; exit 1; }
printf '%s\n' 'PASS: source CLI identity, bounded registry read and missing-repository refusal; no release acceptance claim' > "$3/result.txt"
cat "$3/result.txt"
