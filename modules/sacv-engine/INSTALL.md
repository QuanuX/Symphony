# SACV Engine Installation

```bash
cmake -S modules/sacv-engine -B /tmp/sacv-build \
  -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/chosen/prefix
cmake --build /tmp/sacv-build
ctest --test-dir /tmp/sacv-build --output-on-failure
cmake --install /tmp/sacv-build
```

The install is `installed_undocked`, creates no global command alias, selects no active version, and contacts no service. Invoke the exact installed version through `qxctl sacv ... --prefix /chosen/prefix`.

```bash
cmake --build /tmp/sacv-build --target uninstall-sacv-engine
```

The uninstall target removes only the nine paths named in the exact receipt. A custom `libexec` or `share` directory name is rejected because qxctl deliberately resolves a single receipt layout.

The receipt is immutable for one exact module/version path. A repeated install fails before any owned-file install rule; install another version side by side or run the receipt-verified uninstaller first. Uninstall validates the configured ownership set and every remaining file's recorded size and SHA-256, removes the receipt last, and treats already-missing owned files only as idempotent retry evidence.
