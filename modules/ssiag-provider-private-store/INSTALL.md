# Install the native private credential provider

Build this C++26 component independently with CMake 3.30+:

```
cmake -S modules/ssiag-provider-private-store -B /absolute/provider-build -DCMAKE_INSTALL_PREFIX=/absolute/prefix
cmake --build /absolute/provider-build
ctest --test-dir /absolute/provider-build --output-on-failure
cmake --install /absolute/provider-build
```

For an installed-only build, first install the exact foundation SDK, then configure with `-DSYMPHONY_SSIAG_PRIVATE_INSTALLED=ON -DCMAKE_PREFIX_PATH=/absolute/prefix`. There is no source fallback in that mode. The SOV Cloudflare adapter's installed-only build requires this SDK in addition to the foundation. Exact package removal uses `uninstall-ssiag-provider-private-store`; remove active consumer use first. Credential vaults are operator-owned state and are never deleted by package removal.

Installation alone creates no vault, configuration, credential, SSIAG permission or network session. Explicitly configure the actual native consumer digest, SSIAG service digest/socket/UID, TOPS and toolchain. Import material through FD 3 from an operator-controlled native secret source; never put it in argv, environment of qxctl, shell history, JSON or retained logs. Metadata examples use fictional account IDs and opaque references. See SPEC.md for closed bootstrap frames and expiry/generation rules.


The selected STAV producer grant must admit `symphony.sov.cloudflare.use.intent` / `symphony.sov.cloudflare.use.prepare` and `symphony.sov.cloudflare.use.outcome` / `symphony.sov.cloudflare.use.observe`. SSIAG requires the exact profile permission described in its SPEC.md. Intent and outcome retain only safe target and digest metadata. No installer edits a running permission policy or producer grant automatically.
