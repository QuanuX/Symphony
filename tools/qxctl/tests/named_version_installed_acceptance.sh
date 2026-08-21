#!/bin/sh
set -eu

REPO=${1:-$(pwd)}
case "$REPO" in
  /*) ;;
  *) echo "repository path must be absolute" >&2; exit 2 ;;
esac

WORK=$(mktemp -d "${TMPDIR:-/tmp}/symphony-named-version-acceptance.XXXXXX")
trap 'rm -rf "$WORK"' EXIT HUP INT TERM
PREFIX="$WORK/prefix"

cmake -S "$REPO/modules/sav-engine" -B "$WORK/sav-build" -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$PREFIX"
cmake --build "$WORK/sav-build" --target symphony-sav
cmake --install "$WORK/sav-build"

cmake -S "$REPO/modules/knowledge-session-coordinator" -B "$WORK/coordinator-build" -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$PREFIX"
cmake --build "$WORK/coordinator-build" --target symphony-knowledge-session
cmake --install "$WORK/coordinator-build"

SYMPHONY_NAMED_VERSION_ACCEPTANCE_PREFIX="$PREFIX" \
SYMPHONY_NAMED_VERSION_ACCEPTANCE_REPOSITORY="$REPO" \
go test "$REPO/tools/qxctl/cmd/qxctl" -run '^TestInstalledNamedVersionAcceptance$' -count=1

echo "installed Named Version acceptance passed"
