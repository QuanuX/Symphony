# Observer usage

1. Inspect the installed exact descriptor/schema/template before invoking.
2. Supply an observation ID, nullable caller Node association, exact `linux-proc-sysfs-v1` profile, and one to four unique admitted field IDs. No arbitrary root, path, command, provider or endpoint is admitted.
3. Preserve the acquisition route (`native_fixed_sources` or `sdk_supplied_reader`) and each field status, value, source path, capture interval and consistency result. `complete` covers only the selected profile; `changed_during_capture` preserves the separately captured values and prevents a simultaneous-inventory claim.
4. A caller Node reference is unverified association. Logical CPU masks are neither physical core inventory nor cgroup allocation. MemAvailable is a kernel estimate.
5. Raw boot IDs are compared only in private process memory and never exported or converted into identifiers. The SDK reader injection is explicit dependency injection for tests/custom integrations; the public executable always selects its native fixed-source reader.
