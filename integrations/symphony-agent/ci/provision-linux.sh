#!/usr/bin/env bash
# Run only in a disposable Ubuntu build container/VM. Never on a DGX appliance.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
. /etc/os-release
test "$ID" = ubuntu
apt-get update
apt-get install -y --no-install-recommends ca-certificates curl gnupg make g++ libssl-dev binutils python3 python3-venv xz-utils
case "$VERSION_ID" in
  26.04)
    apt-get install -y --no-install-recommends clang-21 libclang-rt-21-dev cmake
    compiler=/usr/bin/clang++-21
    python=/usr/bin/python3
    cmake=/usr/bin/cmake
    ;;
  24.04)
    apt-get install -y --no-install-recommends clang-18 libclang-rt-18-dev
    compiler=/usr/bin/clang++-18
    python=/usr/bin/python3
    ;;
  22.04)
    curl --fail --location --retry 3 https://apt.llvm.org/llvm-snapshot.gpg.key -o /tmp/llvm.asc
    gpg --batch --show-keys --with-colons /tmp/llvm.asc | awk -F: '$1=="fpr" {print $10}' > /tmp/llvm-fingerprints
    grep -qx 6084F3CF814B57C1CF12EFD515CF4D18AF4F7421 /tmp/llvm-fingerprints
    gpg --batch --dearmor -o /usr/share/keyrings/symphony-llvm.gpg /tmp/llvm.asc
    echo 'deb [signed-by=/usr/share/keyrings/symphony-llvm.gpg] https://apt.llvm.org/jammy/ llvm-toolchain-jammy-20 main' > /etc/apt/sources.list.d/symphony-llvm.list
    apt-get update
    apt-get install -y --no-install-recommends clang-20 libclang-rt-20-dev zlib1g-dev libffi-dev libbz2-dev liblzma-dev libsqlite3-dev
    compiler=/usr/bin/clang++-20
    curl --fail --location --retry 3 https://www.python.org/ftp/python/3.12.15/Python-3.12.15.tar.xz -o /tmp/Python.tar.xz
    echo 'c2c4321961fab0fb999d66e0cecf521c2ab3994c7992873ea99e306c1094fd5a  /tmp/Python.tar.xz' | sha256sum -c -
    tar -xJf /tmp/Python.tar.xz -C /tmp
    (cd /tmp/Python-3.12.15; ./configure --prefix=/opt/symphony-python --with-ensurepip=install; make -j2; make install)
    python=/opt/symphony-python/bin/python3.12
    ;;
  *) echo 'Unsupported Ubuntu version' >&2; exit 1;;
esac
if test "$VERSION_ID" != 26.04; then
  "$python" -m venv /opt/symphony-build-tools
  /opt/symphony-build-tools/bin/pip install --only-binary=:all: --report /tmp/cmake-install.json cmake==3.31.6
  cmake=/opt/symphony-build-tools/bin/cmake
fi
mkdir -p /work/toolchain
printf 'COMPILER=%q\nPYTHON=%q\nCMAKE=%q\nCTEST=%q\n' "$compiler" "$python" "$cmake" "${cmake%/*}/ctest" > /work/toolchain/paths.sh
dpkg-query -W > /work/toolchain/packages.tsv
cp /etc/os-release /work/toolchain/os-release
test ! -f /tmp/cmake-install.json || cp /tmp/cmake-install.json /work/toolchain/
"$compiler" --version > /work/toolchain/compiler.txt
"$python" --version > /work/toolchain/python.txt
"$cmake" --version > /work/toolchain/cmake.txt
getconf PAGESIZE > /work/toolchain/page-size.txt
if ! id symphonybuild >/dev/null 2>&1; then useradd --create-home symphonybuild; fi
chown -R symphonybuild:symphonybuild /work
