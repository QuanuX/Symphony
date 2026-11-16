#!/bin/sh
# Verify already-built candidates; never downloads, publishes, tags or activates.
set -eu
umask 077
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
export GOWORK=off GOENV=off GOTOOLCHAIN=local
[ "$(go env GOVERSION)" = go1.26.5 ] || { echo 'Go1.26.5 is required' >&2; exit 1; }
exec go run "$script_dir/go-candidate-closure.go" "$script_dir/go-candidate-closure-main.go" "$@"
