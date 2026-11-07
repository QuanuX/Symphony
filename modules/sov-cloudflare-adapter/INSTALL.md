# Independent adapter installation

Build outside the source tree with CMake 3.30+ and a C++26 compiler:

```sh
cmake -S modules/sov-cloudflare-adapter -B /absolute/build -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/private-prefix
cmake --build /absolute/build
ctest --test-dir /absolute/build --output-on-failure
cmake --install /absolute/build
```

Receipt-v2 lives under share/symphony/receipts/sov-cloudflare-adapter/0.1.0-dev/. Select that exact prefix/version with qxctl's --adapter-prefix and --adapter-version flags. No global search path selection. The adapter receipt covers schemas, templates and its declaration. Toolchain bundles are imported separately by installation plan/apply. Their removal does not remove the adapter. Use `cmake --build /absolute/build --target uninstall-sov-cloudflare-adapter` for the common exact receipt-bound removal; it refuses changed files and leaves unrelated prefix content.

Example offline discovery input: {"store":null,"toolchain_id":null,"query":null}. Pass it via --input to `qxctl sov cloudflare cf capability list`. Use capability template with {"operation":"installation.plan"} for lifecycle inputs. Do not execute these examples under sudo.

The package contains AGPL-3.0 Symphony code and the nlohmann JSON MIT notice. External Node/cf/dependency licenses and provenance travel in their independently prepared bundle and C0 record. No downloads or npm lifecycle scripts run during CMake install.

The native authoring tools are build-tree tools, not runtime provider entrypoints:

```sh
/absolute/build/cloudflare-generate /absolute/source --check
/absolute/build/cloudflare-prepare_bundle --node /absolute/verified-node --cf /absolute/script-disabled-cf --output /absolute/new-bundle
```

The preparer never downloads or executes a dependency. Supply an already verified official Node distribution and the exact npm tree prepared with scripts disabled. It copies their complete file closure, removes unused npm .bin launch links, rejects remaining special files, and emits the content identity. Only bundles explicitly qualified in the owner declaration can execute offline cf commands; other integrity-valid local imports do not acquire that qualification.
