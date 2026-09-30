# SNV Skill

Use `--descriptor` or receipt-bound `qxctl snv schema|template --owner OWNER --operation OP --prefix PREFIX --version 0.1.0-dev --json`. Templates contain caller choices and unanswered fields; discovery does not establish valid evidence or permission.

The 15 leaves are identity/resources/clusters validation, names validation/resolution, inspect, evidence prepare/commit/status, state plan/apply/status/recover and shared schema/template. Typed transitions use the same validation leaves. There are no provider-specific trees or implicit collectors.

For a named view:

1. Validate exact supplied child inputs and assemble original-byte artifacts using the bundle schema. `snv inspect --input FILE` independently replays them.
2. Submit `snv evidence prepare --input FILE --state-root ROOT`, then `evidence commit --operation-id ID`. Keep the returned sealed acknowledgement. Status can inspect the exact attempt.
3. Obtain `state status --tops-id TOPS --view-id VIEW`. Populate the native state-plan input with that exact head/digest, the retained bundle, a fresh operation ID and explicit reason.
4. Run `state plan --input FILE` and save its entire public capsule unchanged. It writes no head or intent. Run `state apply --input CAPSULE` with the same exact installation and scope; SSIAG/STAV must be available and admit the precise action/resource.
5. For uncertain outcomes use `state status --operation-id ID`, then `state recover --operation-id ID`. Recovery replays the retained original intent and reauthorizes. Do not edit private files.
6. Selected `inspect --state-root ROOT --tops-id TOPS --view-id VIEW` captures the head and replay result. History/export modes and explicit diff baseline use the same leaf. Save returned export JSON and re-admit through an evidence-plan import in a distinct state root.

All commands require exact `--prefix` and `--version`; use `--json` for one machine-readable success or error. Roots are private, clean absolute paths. Name changes and material hardware rules are caller-selected typed data. Corrections retain causal records; unselect is an explicit tombstone with history. Removal of an executable is separate from unselection, record retirement and data erasure. See `SPEC.md` for independent capacity bounds, authority race and supported writer limits.
