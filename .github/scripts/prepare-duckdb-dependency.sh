#!/bin/sh
set -eu
umask 077
if [ "$#" -ne 1 ]; then echo "usage: $0 /absolute/fresh-dependency-directory" >&2; exit 2; fi
case "$1" in /*) ;; *) echo 'dependency directory must be absolute' >&2; exit 2 ;; esac
case "$1" in *'['*|*']'*|*';'*) echo 'unsupported dependency path metacharacter' >&2; exit 2 ;; esac
clean_path=$(printf '%s' "$1" | LC_ALL=C tr -d '[:cntrl:]')
if [ "$clean_path" != "$1" ]; then echo 'dependency path contains control characters' >&2; exit 2; fi
if [ -e "$1" ] || [ -L "$1" ]; then echo 'dependency directory must not exist' >&2; exit 2; fi
repository=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
case "$(uname -s)" in Darwin) os=macos ;; Linux) os=linux ;; *) echo 'unsupported OS' >&2; exit 2 ;; esac
case "$(uname -m)" in x86_64|amd64) architecture=amd64 ;; arm64|aarch64) architecture=arm64 ;; *) echo 'unsupported architecture' >&2; exit 2 ;; esac
mkdir -m 700 "$1"
cmake -DDUCKDB_PREPARE_ROOT="$1" -DDUCKDB_PLATFORM="$os-$architecture" \
  -DDUCKDB_PROVENANCE="$repository/modules/scv-graph-duckdb-connector/DUCKDB-PROVENANCE.json" \
  -P "$repository/cmake/SymphonyPrepareDuckDB.cmake"
