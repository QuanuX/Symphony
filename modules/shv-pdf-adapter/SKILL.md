# Operating the SHV PDF adapter

Use `qxctl shv pdf schema` and `template` with explicit `--prefix` and `--version 0.2.0-dev`. The template is unanswered and must be populated with caller-selected source and trusted decoder identities. Run `extract --input request.json --json`; preserve its `result` with the original request. Run `verify --input verification.json --json` with exactly `{request,artifact}`, where artifact is that result. Reject failed replay; never silently replace source hashes or relabel AMD.OPN as Product ID Tray.

Version 0.2.0 adds native `graph_project` and `graph_validate`, administered by `qxctl shv pdf graph project|validate|roundtrip`. Project takes an extraction request; validate and roundtrip take `{request,graph}`. Roundtrip additionally selects `--adapter-prefix` and `--adapter-version`. See PDF-ADAPTER.md for exact provenance semantics. Retained 0.1.0 installations remain explicitly selectable for their original operations.
