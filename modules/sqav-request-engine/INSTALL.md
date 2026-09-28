# SQAV request engine installation

Requires CMake 3.30+, C++26 and exact native dependencies. From the source repository configure `cmake -S modules/sqav-request-engine -B BUILD -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=PREFIX`, build, run `ctest --test-dir BUILD --output-on-failure`, and install. To use already installed dependencies set `SYMPHONY_SQAV_REQUEST_INSTALLED=ON` and `CMAKE_PREFIX_PATH` to their prefix. zstd 1.5.7 and its license must be available.

The executable lives under `libexec/symphony/sqav-request-engine/0.1.0-dev/`; receipt under `share/symphony/receipts/sqav-request-engine/0.1.0-dev/`. Installation starts no service. `uninstall-sqav-request-engine` removes only unchanged receipt-owned files using the existing receipt guard. No key material is part of the package.
