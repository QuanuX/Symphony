#!/usr/bin/env bash
set -euo pipefail
if [[ ! -f /.dockerenv || $(id -u) != 0 || -n $(hostname -I | tr -d '[:space:]') ]]; then
    echo 'Requires a fresh disposable root container with --network none' >&2
    exit 2
fi
source_dir=$(cd "$(dirname "$0")/.." && pwd -P)
stage=/opt/symphony
build=$(mktemp -d)
chmod 0755 "$build"
export CXX=clang++-21
export CMAKE_GENERATOR=Ninja
cmake -S "$source_dir/../linux-secret-service" -B "$build/backend" -DBUILD_TESTING=OFF \
    -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$stage"
cmake --build "$build/backend" --parallel 2
cmake --install "$build/backend"
cmake -S "$source_dir" -B "$build/node" -DCMAKE_PREFIX_PATH="$stage" \
    -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$stage"
cmake --build "$build/node" --parallel 2
cmake --install "$build/node"
install -d -o ubuntu -g ubuntu "$build/node/Testing" "$build/node/Testing/Temporary"
runuser -u ubuntu -- ctest --test-dir "$build/node" --output-on-failure
cmake -S "$source_dir/tests/installed" -B "$build/consumer" -DCMAKE_PREFIX_PATH="$stage"
cmake --build "$build/consumer" --parallel 2
bin="$stage/libexec/symphony/ssiag-node-keyring/0.1.0-dev"
python3 -I -B "$source_dir/tests/activate-test.py" "$bin/symphony-node-keyring-activate"
python3 -I -B "$source_dir/tests/disposable-ssh-node.py" "$stage" "$build/consumer/node-keyring-installed-consumer"
python3 -I -B "$source_dir/headless/test_restart.py" "$bin/keyring_session.py" \
    "$bin/symphony-node-keyring-import" "$build/consumer/node-keyring-installed-consumer"
cmake --build "$build/node" --target uninstall-ssiag-node-keyring
cmake --build "$build/backend" --target uninstall-ssiag-linux-secret-service
