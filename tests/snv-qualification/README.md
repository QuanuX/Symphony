# Installed SNV qualification

Run after the knowledge group has installed its exact dependency closure and
built qxctl, with CMake 3.30+, a C++26 compiler and Go 1.26.5 on PATH:

```sh
sh tests/snv-qualification/run.sh \
  /absolute/installed-prefix /absolute/new-private-work /absolute/qxctl
```

The existing `mvp-dependencies` knowledge jobs supply fresh prefixes on Ubuntu
24.04 and 26.04 amd64, Ubuntu 24.04 arm64, and macOS 15 arm64. This harness does
not install another package set. An explicit pre-existing prefix is also useful
for a local reproduction; its receipt digests and the harness source revision
are recorded separately. Passing against an older prefix does not certify that
its binaries were rebuilt from the current checkout.

The fail-closed sequence verifies eight exact receipts (foundation, neutral
common SDK, five semantic owners, and optional observer), compiles a copied
consumer using only their imported SDKs, replays all five owners, and runs actual
qxctl supplied-record commands. It checks private evidence prepare/commit/status,
exact repeated commit, a nonmutating state proposal, and absent-root status.
The existing installed Go suite checks interface tampering, five-engine consumers,
adversarial process transport, and the observer consumer, with race checking and
no accepted skips. Receipt-owned bytes are verified again after use.

Observer qualification is separate. The installed SDK consumer injects a denied
field and a changed reading, proving that unseen remains null and a refreshed
observation preserves the old record. Actual `snv observe` calls use the native
fixed sources twice and then select only one field. On Linux, the full captures
must contain some actual observed proc/sysfs evidence; partial/changed results
retain their statuses. A fully unavailable capture is retained but does not pass
Linux runtime qualification. macOS must explicitly report the native observer
unavailable; its supplied-record and SDK qualification still runs.

`logs/host-info.log` records kernel, architecture, OS release, selected kernel
filesystem types, virtualization classification when available, and container
marker presence/absence. Marker absence does not prove bare metal. No raw boot
identity, machine ID, hostname or cgroup allocation is collected. A native result
is limited to that process's exposure and the selected fields. It establishes
neither physical Node identity, complete inventory, nor cgroup entitlement.

`evidence.json` exists with `complete: true` only after every gate succeeds. It
binds receipt bytes, qxctl bytes, command outputs, source revision, host details,
and completed gate logs. `evidence.pending.json` is incomplete evidence only.
Logs and public synthetic/native outputs may be retained as CI artifacts; the
private store, compiler output and module caches are not needed as artifacts.
The signed-release/publisher and installed-native-suite evidence remain owned
by their separate qualification jobs.

The harness never applies a selected view, requests SSIAG authority, starts a
service, provisions infrastructure, assigns names, or invokes a provider. Its
retention mutations are limited to the fresh private fixture directory. Protected
selection/recovery and different-account deployment require their own admitted
qualification; read-only or synthetic results do not substitute for them.
