# Local observer v1

`snv-local-observer 0.1.0-dev` is an optional, independently installed C++26 collector. Its module receipt has null `vector_id` and `engine_id`. The finite process has the distinct target `symphony-snv-local-observer`; it is not a sixth SNV semantic owner. Exactly one `observe` operation is admitted with input `symphony.snv.local-observe-input.v1` and output `symphony.snv.local-observe.v1`.

## Selected Linux profile

`linux-proc-sysfs-v1` observes the invoking process's kernel/proc/sysfs exposure. The request supplies `observation_id`, nullable `node_ref` and one to four unique fields. A Node reference is caller-supplied association, never identification or attestation of the observed host. The executable cannot select another filesystem root, path, command, device, endpoint or source provider.

| Field | Fixed source | Meaning and representation |
| --- | --- | --- |
| `cpu_present` | `/sys/devices/system/cpu/present` | Kernel-present **logical CPU indices**, exact ordered list/ranges and count. This is distinct from online availability. |
| `cpu_online` | `/sys/devices/system/cpu/online` | Kernel-online logical CPU indices and count. This is not the invoking process's cgroup, affinity, allocation or entitlement. |
| `memory_total` | `/proc/meminfo`, `MemTotal` | Kernel-visible usable RAM view, exact reported `kB` multiplied by 1024 to decimal bytes. This is not full physical DIMM inventory. |
| `memory_available` | `/proc/meminfo`, `MemAvailable` | Kernel estimate of memory available for starting applications without swapping; decimal bytes. Missing older/filtered fields remain unavailable. This is not a reservation or process/cgroup allocation. |

The [kernel CPU hotplug documentation](https://docs.kernel.org/core-api/cpu_hotplug.html) separates possible, present and online masks. The [sysfs CPU ABI](https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-devices-system-cpu) describes those exported masks. The [proc documentation](https://docs.kernel.org/filesystems/proc.html) defines the memory view. These documents were reviewed September 30, 2026. Their mutable publication URLs describe the selected field semantics; they are not build dependencies or proof of target conformance.

The collector records architecture and kernel release from `uname`; it omits the nodename. It never reads machine-id, serial numbers, firmware/DMI, hostnames, process command lines/environments, addresses, provider metadata or device catalogues. Unselected sources are not read. No directories are enumerated.

## Capture and consistency

Before and after collection, the collector reads `/proc/sys/kernel/random/boot_id` and validates its lowercase UUID representation. The raw values are compared only in private process memory. Neither raw values nor their hashes appear in output, source digests, identifiers or errors. `boot_consistency` is `stable`, `changed` or `unavailable`; before/after read statuses remain explicit.

Each selected CPU source is checked before and after the memory read. Its first captured value is preserved, with separate `verification_status` and `consistency`; a changed mask yields `changed_during_capture`. Memory comes from one bounded meminfo read; its consistency is `not_checked`. A stable boot and stable CPU check only establish equality of checked endpoints over the capture interval. They establish no atomic whole inventory, unchanged intermediate history or stable physical identity. Wall-clock start/end and steady-clock elapsed time describe the interval; wall-clock time does not govern physical continuity.

`complete` means all selected fields were decoded and applicable checks succeeded. `partial` means some selected evidence or consistency check is unavailable. `changed_during_capture` takes precedence when boot or checked CPU masks differ. `unavailable` means the native collector is unavailable on this platform or no requested field was decoded. Per-field statuses distinguish unavailable, permission denied, unsupported source, too large, malformed and I/O errors. These states never become zero capacity, physical absence or destruction.

## Exact bounds and access

CPU files: 4096 bytes each, at most 2048 ordered nonoverlapping range segments, index maximum 1048575; ranges are not expanded. Boot file: 64 bytes. Meminfo: 16384 bytes. Selected quantities are canonical uint64 decimals with overflow rejected before multiplying by 1024. Duplicate selected meminfo labels and invalid units are malformed. Unrelated meminfo content is discarded and is never exposed or digested.

The maximum collection interval is five seconds or the earlier envelope deadline. Guards run before/after source reads and between read chunks, with a steady-clock interval limit. Expiry returns `deadline_exceeded` with process exit 3; it does not return a success labelled complete. The administering process additionally enforces its child-process deadline. Reads use no-follow traversal, nonblocking read-only descriptors, close-on-exec, regular-file admission and the expected proc/sysfs filesystem type. No permission escalation or command fallback occurs. Fixed kernel source reads are not a guarantee against a faulty kernel stalling inside a syscall.

## Independent use and authority

Each result states `acquisition_route`: `native_fixed_sources` for the executable/private native reader (including unavailable non-Linux platforms), or `sdk_supplied_reader` for explicit reader injection. Injected values never claim native acquisition merely because their declared platform is Linux. Finite-process consumers require the native route.

The SDK exposes an explicit `SourceReader` interface for deterministic synthetic testing and caller-authored integrations. It is dependency injection, not a configurable root for the public executable. The native finite process always uses its private Linux reader. Synthetic/macOS tests are separately identified from actual Linux VM/container evidence. The initial exact Linux target belongs to the follow-up execution record; no bare-metal conformance is implied.

Observation values may later be referenced as attributed evidence by an explicitly constructed SNIV/SNRV request. This collector emits no physical identity, materiality decision, inventory record, resource instance, name, incarnation, membership, selected head, authorization or hidden side effect. Evidence retention and protected selection remain separate qxctl/SNV operations.
