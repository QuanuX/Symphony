# R API+ binding regeneration

The C++ generator consumes the reviewed `vendor_schema_13_7.json` interoperability inventory and the operator's separately licensed `RApiPlus.h`. It checks the exact header SHA-256 before emitting any files. No vendor header, library, implementation or documentation belongs in the repository.

From the repository root:

```sh
c++ -std=c++2c -Wall -Wextra -Wpedantic -Werror \
  -Ilibraries/knowledge-vector-engine-cpp/third_party \
  -Ilibraries/knowledge-vector-engine-cpp/include \
  modules/srrapiv-runtime-cpp/tools/generate_vendor_bridge.cpp \
  libraries/knowledge-vector-engine-cpp/src/digest.cpp \
  -o /tmp/srrapiv-generator
/tmp/srrapiv-generator \
  --header /path/to/licensed/13.7.0.0/include/RApiPlus.h \
  --schema modules/srrapiv-runtime-cpp/tools/vendor_schema_13_7.json \
  --output modules/srrapiv-runtime-cpp/src
```

Generated files contain native interface names, type/field/overload schemas and Symphony-owned typed adapter code. The schema records native parameters separately from control adaptations: runtime-owned callbacks/timers, bounded lock scopes, opaque generation-bound handles and protected password output. Metadata is emitted in bounded literal chunks. The checked-in output makes builds reproducible without distributing the SDK.

For a new release, create a separate reviewed schema/version binding; do not edit the selected release's digest to bypass the guard. Independently compare the new header's public declarations and class inheritance with a compiler AST, classify every pointer/count and lifetime relationship, review version notes, regenerate, and compile against the exact new SDK. Execute offline ownership/type/callback tests and separately qualify provider behavior. The 13.7.0.0 inventory was independently checked against Apple clang's AST: 118 engine signatures, 91 RCallbacks, 3 AdmCallbacks, and 1,231 direct public fields. The timer is the additional callback.

`tests/vendor_test.cpp` includes the bridge implementation as a test seam. Compile that test by itself, not together with a separately compiled vendor.cpp. Mark vendor includes as system headers while retaining strict warnings on Symphony code. The tests link the real SDK, invoke its local version/object utilities, and call every callback adapter with native fixture objects. They neither construct REngine nor contact a provider. Full dispatch compilation proves native type compatibility; it does not prove remote behavior, entitlements, recovery or live trading acceptance.

Public input fields are deliberately initialized to zero/null after native construction and before decoding caller values. This is the bridge's deterministic input profile, not a claim about vendor constructor defaults; the SDK constructors retain authority over their private state. Initialization buffers, including envp, live until the native engine is destroyed. Supplied presence flags and PnL value-state fields are preserved; unavailable associated values are tagged without reading their storage. The reviewed schema records those conditions. Callback payloads are copied as supplied by the SDK and are never reinitialized.

All environment values are redacted by default. Private `safe_environment_keys` may expose named non-credential values. Private `protected_environment_outputs` maps a variable name to a `secret_outputs` reference; that path receives exact bytes through owned0600, single-link, no-symlink directory traversal. This gives authorized access to every environment field without writing credentials to ordinary JSON or journals. `getPassword` uses the same protected output mechanism. Native strings retain byte lengths and pointer presence, distinguishing a null value from a non-null empty value.
