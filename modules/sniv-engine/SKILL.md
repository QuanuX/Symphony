# SNIV engine procedure

1. Select the exact engine release and installed schema/template.
2. Supply the owner input with explicit source records, opaque subject identifiers, causal predecessor/generation and capacity profile.
3. Invoke `identity_validate` through the finite process protocol or native library, with a deadline.
4. Inspect `status`, `findings`, canonical input `source_digest` and `subject_ids`; conflicts and insufficient evidence remain visible.
5. For a typed transition, inspect the proposed records and retain/select them only through the applicable SNV evidence/state operations.

Do not infer authority, current hardware truth or performed mutation from a valid supplied-record result. Direct SDK calls perform no filesystem or network effects.
