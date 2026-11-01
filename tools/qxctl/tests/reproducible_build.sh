#!/bin/sh
# Local qualification only: creates two private builds and identity evidence.
set -eu
umask 077
[ "$#" -eq 1 ] || { echo "usage: $0 /absolute/new-work-directory" >&2; exit 2; }
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
command -v go >/dev/null 2>&1 || { echo 'Go 1.26.5 is required' >&2; exit 1; }
command -v cmake >/dev/null 2>&1 || { echo 'CMake 3.30+ is required for this qualification recipe' >&2; exit 1; }
export GOENV=off GOWORK=off GOTOOLCHAIN=local GO111MODULE=on
export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org
export GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOFLAGS= GOAUTH=off
export GOVCS='*:off' CGO_ENABLED=0
export GOAMD64=v1 GOARM64=v8.0 GOFIPS140=off
unset GOOS GOARCH GOEXPERIMENT GOCACHEPROG
exec cmake "-DQXCTL_MODULE=$script_dir/.." "-DQXCTL_WORK=$1" \
  "-DQXCTL_GO=$(command -v go)" -P "$script_dir/reproducible_build.cmake"
