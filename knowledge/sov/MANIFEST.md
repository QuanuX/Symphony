# Symphony Ops Vector Manifest

The optional scoped network carrier and its complete qxctl selection are defined
by `SSH-CARRIER.md` and `tools/sov-ssh-adapter/MANIFEST.md`. It does not provision
the selected endpoint or qualify a deployment by its presence alone.

## Canonical Target

`knowledge/sov/`

## Declared Contract Truth Role

SOV is the semantic owner of supported provisioning, conditioning, delivery, remote-administration, and bus-adapter operations exposed through qxctl. qxctl implements and presents those operations but does not own their domain meaning.

## Canonical Surfaces

- `knowledge/sov/INTENT.md`
- `knowledge/sov/MANIFEST.md`
- `knowledge/sov/SPEC.md`
- `knowledge/sov/SKILL.md`
- `knowledge/sov/PRIMA-PARTE.md`
- `knowledge/sov/REMOTE-QXCTL.md`
- `knowledge/sov/REMOTE-HANDOFF.md`
- `knowledge/sov/PROCESS-EXECUTION.md`

## Implementation Status

Canonical architecture contract with exact `qxctl sov remote plan|assess|schema|template|invoke` and `qxctl sov target configure|inspect|remove|invoke` under `REMOTE-HANDOFF.md` and `PROCESS-EXECUTION.md`. The independently packaged `tools/sov-stdio-adapter` implements the selected local process carrier. Real remote-host deployment and authenticated network identity remain unqualified. No SOV engine, provider mutation, Terraform execution, Habitat builder, or Nest deployment command is claimed.

## Language and Packaging

qxctl remains Go. Performance-sensitive or Node-local C++ components remain separately installed executables behind ratified contracts. Provider and transport adapters must not be absorbed into qxctl merely for convenience.

## Installability

Future executable SOV components must remain independently installable, exact-version selectable, rollbackable, and removable under the common SKV lifecycle. A pure SOV contract does not itself create an installed component.

## Non-Authorization Statement

This manifest authorizes no cloud account access, provider action, remote shell, package installation, bus connection, Node mutation, Nest interpretation, live-trading intervention, or canonical apply operation.

## Native Cloudflare local foundation

The subordinate owner at `knowledge/sov/cloudflare/MANIFEST.md` assigns C0/I1 local CLI lifecycle and offline discovery to the independently installed C++26 `modules/sov-cloudflare-adapter`. qxctl implements exact Go presentation and dispatch. Provider credentials/resources and optional SSH profiles remain unavailable pending I2–I6. This addition does not broaden the SBV receiver or direct SSH v1 carrier.

## Protected transport profiles (I2)

`knowledge/sov/transport-profiles/MANIFEST.md` assigns protected local administration to `tools/sov-transport-profile-admin`. This later increment supersedes the I1 statement that all profile administration is pending. qxctl reads/plans as the user and invokes the exact sudo helper only for apply/recover. SSIAG/STAV authenticate local root audit; I3/I5 transport execution/session qualification remain unavailable.
