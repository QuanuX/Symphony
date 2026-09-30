# SNV Install

CMake >=3.30, a CMake-recognized C++26 compiler and a single-configuration generator are required. Configure an explicit private installation prefix:

```sh
cmake -S modules/snv-engine -B build/snv -G Ninja -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/private/prefix
cmake --build build/snv
ctest --test-dir build/snv --output-on-failure
cmake --install build/snv
```

Source mode builds the exact declared foundation, shared mechanics and four owner SDKs. Installed-only mode adds `-DSYMPHONY_SNV_USE_INSTALLED_DEPENDENCIES=ON -DCMAKE_PREFIX_PATH=/absolute/dependency/prefix`; all six exact dependency packages and unchanged receipt-owned archives/headers/configuration must be present. No network fetch or substitute release occurs. macOS toolchain selection may require an explicit SDK and matching linker.

The SDK exports `SymphonySnv` version `0.1.0` and target `Symphony::Snv`. The executable is `libexec/symphony/snv-engine/0.1.0-dev/symphony-snv`. Headers, archive, CMake exports, schemas, owner declaration and Contract Quad documentation have exact versioned paths covered by the receipt-v2 package. Schemas and templates are inspected through that exact receipt, not a repository-relative fallback.

Each child also builds and installs independently with only the exact foundation/shared mechanics dependencies. Parent operation after installation does not require child executables; its embedded dependency manifest identifies the compiled reducers.

`cmake --build build/snv --target uninstall-snv-engine` uses the configured prefix and removes only unchanged receipt-owned files. A changed file, unsafe target or shared-root lifecycle ownership refuses removal. Dependencies and private user evidence stores remain. The existing qxctl lifecycle circuit owns managed shared-root reclamation. No package manager or uninstaller erases SNV journals.
