#!/bin/sh
set -eu
[ "$#" -eq 2 ] || { echo 'usage: run.sh ABSOLUTE_KNOWLEDGE_PREFIX FRESH_PRIVATE_WORK' >&2; exit 2; }
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../../../.." && pwd -P)
exec sh "$repo/tests/shv-release-qualification/run.sh" "$1" "$2"
