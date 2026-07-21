# Knowledge Vector Engine C++ Foundation Specification

## Status

Implemented development foundation, version `0.1.0-dev`. It is not a published module release.

## Process Limits

| Limit | Value |
|---|---:|
| request | 1 MiB |
| response | 4 MiB |
| JSON depth | 64 |
| JSON values/events | 16,384 |
| JSON string or key | 65,536 bytes |
| request/correlation/engine token | 128 bytes |
| operation token | 64 bytes |
| relative path | 4,096 bytes |
| snapshot paths | 1,024 |
| one snapshot file | 4 MiB |
| deadline window | 300,000 ms |

JSON objects reject duplicate names. Process JSON rejects floating-point values, integers outside the interoperable range `[-9007199254740991, 9007199254740991]`, invalid UTF-8, unknown envelope fields, trailing bytes, excess nesting/count/size, unsupported protocol versions, expired or excessively distant deadlines, and target-engine mismatches.

Snapshot reads check the request deadline before and between file-read chunks. The future qxctl process client must also enforce that deadline on the child lifetime; the shared library does not claim that a cooperative check can cancel a blocked kernel/filesystem call.

## Request and Response

The exact canonical envelope schemas are `knowledge/schemas/v1/engine-process-request.schema.json` and `knowledge/schemas/v1/engine-process-response.schema.json`. The implementation reserves standard output for one compact response followed by one newline. Error diagnostics are stable codes plus bounded control-free text.

`response_digest` is `sha256:` plus the SHA-256 of the compact, key-sorted response object before the `response_digest` member is inserted.

## Path and Snapshot Contract

Portable paths are non-empty forward-slash relative paths with no absolute root, empty component, `.`, `..`, backslash, NUL, control byte, or component traversal. Reads open the root, every intermediate directory, and the final regular file through no-follow file-descriptor operations. Symlink and special-file reads fail closed.

Snapshot paths are unique and sorted. Each file records a tagged content digest and byte size. The snapshot digest covers a length-delimited canonical sequence of path, size, and content digest.

## Dependency Contract

nlohmann/json `v3.12.0` is pinned and vendored. Its header SHA-256 MUST remain `aaf127c04cb31c406e5b04a63f1ae89369fccde6d8fa7cdda1ed4f32dfc5de63`. Replacement or upgrade requires dependency review, conformance reruns, and SCLV/SODV evidence.

## Build Receipt Boundary

Version `0.1.0-dev` accepts single-configuration CMake generators only. This keeps the generated export filename deterministic and lets the installation receipt enumerate the exact owned file set. Multi-configuration packaging remains unsupported rather than producing an ambiguous or inaccurate uninstall boundary.

## Non-Authorization

The library has no semantic, authentication, ratification, mutation, publication, network, runtime-ledger, or docking authority.
