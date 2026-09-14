# Operating the SHV PDF adapter

Use `qxctl shv pdf schema` and `template` with explicit `--prefix` and `--version 0.1.0-dev`. The template is unanswered and must be populated with caller-selected source and trusted decoder identities. Run `extract --input request.json --json`; preserve its `result` with the original request. Run `verify --input verification.json --json` with exactly `{request,artifact}`, where artifact is that result. Reject failed replay; never silently replace source hashes or relabel AMD.OPN as Product ID Tray.
