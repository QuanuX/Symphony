#!/usr/bin/env bash
# Disposable GitHub VM or WSL guest only. Uses /work, never a DGX OS installer.
set -euo pipefail
source /work/toolchain/paths.sh
version="$1"
arch="$2"
kind="$3"
"$PYTHON" /work/source/ci/qualify_local.py --source /work/source --out /work/result \
  --compiler "$COMPILER" --cmake "$CMAKE" --ctest "$CTEST" \
  --expected-os ubuntu --ubuntu-version "$version" --expected-arch "$arch" --host-kind "$kind" --jobs 2
mkdir -p /work/packages
"$PYTHON" /work/source/scripts/package_release.py \
  --binary /work/result/build/symphony-mcp --output "/work/packages/symphony-0.1.0-alpha.1-ubuntu-${version}-${arch}-candidate.zip"
if test "${4:-false}" = true; then
  "$PYTHON" /work/source/ci/qualify_local.py --source /work/source --out /work/sanitizers \
    --compiler "$COMPILER" --cmake "$CMAKE" --ctest "$CTEST" --expected-os ubuntu \
    --ubuntu-version "$version" --expected-arch "$arch" --host-kind "$kind" --jobs 2 --sanitizers
fi
