# SACV Engine Skill

Read `knowledge/sacv/` before invoking this engine. Use `check` for canonical registry and registered-contract conformance, `diff` for file-bound compatibility evidence, `propose` for one caller-declared register or replace operation, and `project` for a disposable inventory.

The initial parser fully supports bounded OpenAPI 3.2.0 JSON entry documents. YAML remains a permitted canonical representation but this development version fails closed on `.yaml` rather than silently guessing, downgrading, or invoking an unpinned parser. Do not interpret parser unavailability as contract invalidity.

Never treat an engine result as ratification, canonical apply, SODV publication approval, SDK approval, or endpoint authorization.
