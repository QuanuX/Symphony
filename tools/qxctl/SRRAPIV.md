# SRRAPIV administration

`qxctl sqv rithmic` controls the sovereign Symphony Rithmic RAPI Vector under SQV. Its semantic owner is the independently installed `srrapiv-runtime-cpp` C++ engine, `symphony-srrapiv`. Every command requires an exact `--prefix` and `--version`; the initial admitted engine version is `0.1.0-dev`. Engine and vendor SDK versions are separate identities. Selecting this engine never substitutes a newer SDK for an exact strategy binding.

| Command | Native operation | Effect |
| --- | --- | --- |
| `catalog inspect` | `catalog_inspect` | Inspect caller-supplied exact artifacts |
| `api describe` | `api_describe` | Describe the complete installed SDK dispatch surface |
| `binding validate` | `binding_validate` | Check an exact binding and compatibility assertions |
| `binding compare` | `binding_compare` | Compare two exact bindings |
| `coverage assess` | `coverage_assess` | Assess explicit method coverage evidence |
| `runtime start` | `runtime_start` | Start a resident local native owner |
| `runtime status` | `runtime_status` | Inspect the current owner and generation |
| `runtime invoke` | `runtime_invoke` | Invoke an exact SDK method through the native owner |
| `runtime events` | `runtime_events` | Read bounded native callback events |
| `runtime stop` | `runtime_stop` | Stop the selected owner generation |
| `runtime attempts` | `runtime_attempts` | Inspect retained invocation recovery evidence |
| `schema` / `template` | `--operation <exact native name>` | Read receipt-owned version-specific contracts |

Operation commands accept a bounded `--input` JSON file. Secrets never become command flags. The input protocol is `symphony.srrapiv.request.v1`; discovery templates are unanswered examples, not validated input or authorization. Use the installed schema and native SDK description for fields and exact method identities. SDK doubles use the native `f64_bits` object for exact floating-point values; integers outside the JSON safe-integer range use decimal strings. The transport rejects floating-point JSON numbers.

The Go client verifies the selected receipt, native executable, pinned owner interface, schema/template bytes, response identity and request/result digests. Native C++ owns API semantics, runtime generation, current-user permissions, credential handling, provider calls and recovery. Runtime start, invoke and stop are permission-backed mutations; provider and trading authority are explicit native request fields. This release does not claim SSIAG integration or live-provider qualification.

The client makes one attempt and does not retry uncertain mutations. After a lost start response, inspect `runtime status` without a generation to discover the authenticated local owner. Before repeating an invocation, inspect `runtime attempts` and reconcile its outcome with provider observations. A retained local attempt is not provider-side exactly-once execution evidence. A native domain refusal is printed as one complete validated response and exits nonzero.

A submitted SDK call may return `in_progress`; read `runtime attempts` with the same attempt ID for its retained outcome. One SDK worker runs at a time while status and event inspection remain available. A distinct concurrent invocation is refused. Stop may return `stopping`, or an explicit blocked result while an SDK invocation is active. Observe status and attempt evidence before further action; successful asynchronous admission is not completed provider activity.

Global `commands expected` and `commands manifest` expose all stable command identities, owner operation mappings, effect classifications and recovery routes. Existing module and installation controls manage independently installed engine versions. Installed schemas and descriptor identities fail closed when they differ from the selected client contract.
