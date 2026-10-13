# SOV SSH carrier manifest

- Module: `github.com/QuanuX/Symphony/tools/sov-ssh-adapter`
- Release: `0.1.0-dev`
- Go baseline: `1.26.5`; standard library only
- Stock platforms: Darwin and Linux; exact host qualification is recorded separately
- Owner: `knowledge/sov/SSH-CARRIER.md`
- Runtime: selected OpenSSH client, selected agent socket/key, existing target SSH endpoint and selected target stdio launcher
- Control: `qxctl sov remote ssh-endpoint|ssh-invoke`; complete templates/schemas in `qxctl sov remote template|schema`
- Public protocol: `contract/contract.go`; qxctl shares these wire types at build time
- Implementation: `main.go`; tests: `main_test.go`, `contract/contract_test.go`
- Installation/removal: `README.md`; no owned service, credentials or shared runtime state

Target-side native ownership and optional SSIAG admission remain unchanged.
The launcher self-check detects trusted-installation drift; it is not attestation.
