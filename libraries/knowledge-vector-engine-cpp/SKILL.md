# Knowledge Vector Engine C++ Foundation Skill

## Purpose

Guide callers implementing bounded knowledge-vector process mechanics without moving vector semantics or authority into the shared library.

## Use For

- exact process-envelope parsing and response framing;
- deterministic digests and read-only snapshots;
- strict relative-path and no-follow file access;
- stable, safe protocol errors;
- independently versioned static linking.

## Required Checks

1. Apply input, depth, value-count, string, path, file, and response limits before domain work.
2. Reject duplicate JSON names, invalid UTF-8, trailing data, unknown envelope fields, floats, stale deadlines, and target-engine mismatches.
3. Treat repository paths and contents as untrusted.
4. Keep standard output to one bounded response in process mode.
5. Preserve read-only behavior unless a later canonical apply gate explicitly changes the contract.

## Do Not Use For

Do not place vector-specific decisions, host permission, caller classification, credentials, arbitrary commands, provider payloads, session mutation, canonical writes, release publication, or Maestro policy in this foundation.
