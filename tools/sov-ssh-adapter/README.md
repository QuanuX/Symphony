# SOV SSH adapter 0.1.0-dev

Build from the repository workspace using the selected Go toolchain:

```sh
go build -o /chosen/prefix/bin/sov-ssh-adapter ./tools/sov-ssh-adapter
go build -o /chosen/target/bin/sov-stdio-adapter ./tools/sov-stdio-adapter
```

The second executable belongs on the selected target, together with target qxctl
and native SBV. Select exact executable digests; no latest-version discovery is
performed. The stdio launcher must support `--self-sha256`.

Inspect `qxctl sov remote template --operation ssh_endpoint --json`, author the
endpoint with the existing SSH host key, chosen public identity/agent socket and
runtime paths, and run `qxctl sov remote ssh-endpoint --input endpoint.json --json`.
Use its `endpoint_ref` in a `scoped_cli` remote selection with adapter ID
`symphony.sov.ssh-adapter`, version `0.1.0-dev`, and the exact adapter digest.
Compute the selection digest through `qxctl sov remote plan`. The
`ssh_invoke` template contains the full envelope and existing native invocation.
Run it with `qxctl sov remote ssh-invoke --input invocation.json --json`.

Templates contain synthetic references and public placeholder keys. They do not
discover credentials or authenticate a host. Load the chosen private key into the
user's agent through their existing credential workflow; no key is copied into
Symphony inputs. The target account and authorized key are owner-managed
prerequisites, not created by this adapter.

Remove the selected carrier executable to uninstall it. Its temporary public-key
and client copies are deleted after each completed call; no credential, target
configuration, native journal or result is removed. Target configuration remains
mutable through `qxctl sov target configure|inspect|remove` on the target.

The complete [contract](../../knowledge/sov/SSH-CARRIER.md) explains identity,
timeouts, cancellation, result framing, qxctl coverage and remaining qualification.
