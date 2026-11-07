# Build and install SRRAPIV

A source/control build needs CMake 3.30+, a C++26 compiler and the repository foundation. It provides exact contracts and local orchestration while vendor invocation remains unavailable.

```sh
cmake -S modules/srrapiv-runtime-cpp -B build/srrapiv -DCMAKE_BUILD_TYPE=Release
cmake --build build/srrapiv
ctest --test-dir build/srrapiv --output-on-failure
cmake --install build/srrapiv --prefix /absolute/selected/prefix
```

For an independent installed build, select `-DSYMPHONY_SRRAPIV_INSTALLED=ON` and `-DCMAKE_PREFIX_PATH=/absolute/foundation/prefix`. This requires the exact installed SymphonyKnowledgeVectorEngine 0.2.0 SDK and refuses source fallback. Shared qualification uses this profile with `SYMPHONY_SRRAPIV_WITH_RAPI=OFF`.

For the actual C++ bridge, obtain the SDK under your applicable vendor agreement, extract it outside the source tree and explicitly select its platform lane.

```sh
cmake -S modules/srrapiv-runtime-cpp -B build/srrapiv-sdk   -DCMAKE_BUILD_TYPE=Release -DSYMPHONY_SRRAPIV_WITH_RAPI=ON   -DRAPI_ROOT=/absolute/private/13.7.0.0 -DRAPI_PLATFORM=darwin-10
cmake --build build/srrapiv-sdk
```

The other supplied lanes are darwin-20.6-arm64, linux-gnu-3.10.0-x86_64 and linux-gnu-4.18-x86_64. Select only the actual host/compiler/ABI combination; listing a lane is not qualification. This mapping refuses a different header digest. Vendor dependency order and static archive digests are explicit. Do not redistribute SDK-linked artifacts without the applicable vendor and dependency review.

Install into an exact selected prefix with the versioned receipt. Keep an existing prefix intact when preparing a different build. Use `qxctl sqv rithmic api describe --prefix /absolute/prefix --version 0.1.0-dev --input /absolute/request.json --json` with request `{"protocol":"symphony.srrapiv.request.v1"}` to inspect the native map. The schema and template leaves accept the exact native operation via --operation.

Runtime files are deliberately outside package receipts. Create a mode-0700 directory and mode-0600 private configuration, with no symlink components. Select catalog/binding records as documented by BINDINGS.md. Keep allow_provider_calls and allow_trading false until explicitly intending those operations. Native initialization, login and order operations are independent actions. Package removal must never recursively delete runtime state, credentials or retained attempts.
