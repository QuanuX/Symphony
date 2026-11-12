# Independent adapter installation

Build outside the source tree with CMake 3.30+ and a C++26 compiler:

```sh
cmake -S modules/sov-cloudflare-adapter -B /absolute/build -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/private-prefix
cmake --build /absolute/build
ctest --test-dir /absolute/build --output-on-failure
cmake --install /absolute/build
```

To consume an independently installed foundation, configure a fresh build with
`-DSYMPHONY_CLOUDFLARE_INSTALLED=ON -DCMAKE_PREFIX_PATH=/absolute/foundation-prefix`.
This requires `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` and never falls back to
the sibling source tree. The default source build remains available. Shared
installed-SDK qualification uses the installed option and verifies the selected
headers and library belong to the supplied prefix.

Receipt-v2 lives under share/symphony/receipts/sov-cloudflare-adapter/0.1.0-dev/. Select that exact prefix/version with qxctl's --adapter-prefix and --adapter-version flags. No global search path selection. The adapter receipt covers schemas, templates and its declaration. Toolchain bundles are imported separately by installation plan/apply. Their removal does not remove the adapter. Use `cmake --build /absolute/build --target uninstall-sov-cloudflare-adapter` for the common exact receipt-bound removal; it refuses changed files and leaves unrelated prefix content.

Example offline discovery input: {"store":null,"toolchain_id":null,"query":null}. Pass it via --input to `qxctl sov cloudflare cf capability list`. Use capability template with {"operation":"installation.plan"} for lifecycle inputs. Do not execute these examples under sudo.

The package contains AGPL-3.0 Symphony code and the nlohmann JSON MIT notice. External Node/cf/dependency licenses and provenance travel in their independently prepared bundle and C0 record. No downloads or npm lifecycle scripts run during CMake install.

The native authoring tools are build-tree tools, not runtime provider entrypoints:

```sh
/absolute/build/cloudflare-generate /absolute/source --check
/absolute/build/cloudflare-prepare_bundle --node /absolute/verified-node --cf /absolute/script-disabled-cf --output /absolute/new-bundle
```

The preparer never downloads or executes a dependency. Supply an already verified official Node distribution and the exact npm tree prepared with scripts disabled. It copies their complete file closure, removes unused npm .bin launch links, rejects remaining special files, and emits the content identity. Only bundles explicitly qualified in the owner declaration can execute offline cf commands; other integrity-valid local imports do not acquire that qualification.


I4 installed-only builds also require the independently installed `SymphonySSIAGPrivateStore 0.1.0 EXACT` SDK. Install `modules/ssiag-provider-private-store` after the foundation and before this adapter. Provider installation does not create or select credentials. Native backend bootstrap is explicit; see its INSTALL.md/SPEC.md.

## Optional I5 Access session package

The default receipt includes `symphony-cloudflare-access` and `access-sessions.json`.
It supports offline `--describe` but refuses session/connection execution until a
protected runtime is explicitly selected. This does not install host policy or
change SSH v1.

For an approved protected helper package, supply both absolute CMake inputs
`SYMPHONY_ACCESS_CLOUDFLARED` and `SYMPHONY_ACCESS_RUNTIME_CONFIG`. They install as
receipt-owned sibling files `cloudflared` and `access-runtime.json`. The latter is:

```json
{"protocol":"symphony.sov.cloudflare-access-runtime.v1","cloudflared_version":"2026.10.7","cloudflared_digest":"sha256:5c55e77da7012cebca19170357e196176c48c9647cc2ead404702fa125ddbd86","carrier_digest":"sha256:<actual protected carrier digest>","ssh_digest":"sha256:<actual protected OpenSSH digest>"}
```

Placeholders are deliberately invalid. Current runtime admission is macOS amd64
only, matching the exact C0 artifact. Linux fixtures do not authorize a Linux vendor
binary. Use the I2 receipt/package installation contract for the helper's `proxy`
role, selecting receipt entry point `symphony-cloudflare-access` and preserving its
sibling layout. Select exact carrier and SSH packages in the profile. All runtime
files and their ancestors must be root-owned, nonsymlink and not group/world
writable; no user prefix, environment override or PATH discovery grants trust.
Profile create/enable/policy changes use the existing sudo-gated administrator and
its SSIAG/STAV admission. Do not replace a protected executable in place.

As the admitted ordinary user, create a private session directory, select the
Access application and independent SSH endpoint through I3, then use
`qxctl sov transport session template --json` and
`qxctl sov transport session schema --json`. Replace the expired template deadline
and synthetic endpoint. Inspect, plan explicit authentication with audience/issuer/
subject, and authenticate using its exact plan digest. Browser completion may be
required. See `knowledge/sov/CLOUDFLARE-ACCESS-SESSIONS.md` for renewal, clearing,
uncertain-outcome recovery and the pending live qualification gate.
