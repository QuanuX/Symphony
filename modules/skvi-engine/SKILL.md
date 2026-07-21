# SKVI Engine Skill

## Direct Diagnostics

```bash
symphony-skvi --help
symphony-skvi --version
symphony-skvi --descriptor
```

Process mode accepts one bounded common request. `inspect` accepts `{}`. `check` accepts only `expected_index_digest`. `project` accepts only `format: "json"`. `propose` accepts the exact caller-declared payload governed by `knowledge/skvi/schemas/v1/operation-payload.schema.json`.

## Safe Procedure

1. Run `check` and inspect its deterministic summary.
2. Treat any structural, duplicate, unsafe, missing, or expected-state finding as drift evidence.
3. Use `project` only when check state is valid; retain its input and projection digests with any disposable copy.
4. Use `propose` only for an explicit caller-declared add, replace, or remove operation.
5. Review proposal expected state, read set, write set, operation digest, validation, and expiry before any separately authorized workflow consumes it.

Proposal fields are nonsecret administrative metadata. Never place credentials, cryptographic proofs, raw tokens, provider payloads, environment data, or executable instructions in an entry or repository reference.

## Stop Conditions

Stop for a separately reviewed increment before adding canonical writes, inferred membership, general search, file-backed projection output, session mutation, qxctl lifecycle activation, SSIAG/STAV calls, a network listener, or Maestro docking.
