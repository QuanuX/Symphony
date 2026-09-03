# Symphony Ops Vector Specification

## Status

Architect-ratified domain and boundary specification. Detailed operation schemas and executable behavior remain deferred.

## Operational Model

SOV operations are explicit, user-selected, exact-target transactions administered through qxctl. Configuration may differ by TOPS, TROG, cluster, Node, Habitat, Nest, provider, region, bus, or operation. A cluster-wide default must not erase a more specific user choice.

Every implemented mutation must eventually bind:

- the exact target and its applicable SNV identities;
- the requested operation and stable operation identity;
- the expected prior state and intended resulting state;
- exact component, Habitat, Nest, adapter, package, and version identities where applicable;
- SCV, SNV, SHV, or other evidence actually used for compatibility;
- authentication, authorization, audit, idempotency, interruption, retry, verification, and recovery behavior;
- the qxctl and independently installed component versions that can interpret the contract.

These fields are design obligations, not an authorization to invent their schemas before the corresponding operation is ratified.

## Node, Habitat, and Nest Conditioning

SOV treats a Node as target hardware, a Habitat as its exact OS/package conditioning, and a Nest as user-purpose code delivered into that Habitat. A Habitat must include only the software surface its declared work requires unless the user explicitly selects a broader general-purpose Habitat. Containers, Kubernetes, and implicit virtualization are not part of the target architecture.

Compatibility evidence may compare declared Nest requirements, Habitat contents, Node resources, provider constraints, bus availability, and exact dependency versions. It reports exact compatibility; it does not decide what the Nest should do.

## Provisioning

qxctl may eventually communicate ratified Terraform recipes to AWS, Azure, Google Cloud, DigitalOcean, and other supported providers, or perform the applicable supported operation for owned hardware. No provider is the baseline against which another provider is judged.

Hybrid clusters and provider/region restrictions are normal inputs. A supported plan must preserve the user's constraint rather than silently substituting a different provider, region, or product.

## Bus Administration

SOV owns the supported qxctl process for installing, configuring, connecting, changing, and removing a selected bus adapter. NATS JetStream, ZeroMQ, and other fabrics remain adapters. Multiple fabrics may coexist. No SOV rule requires hot-path messages, exhaust, control traffic, research traffic, or heavy transfers to share one fabric.

## Remote Operation

Remote operation is optional and explicitly enabled. Controller-side qxctl, target-side qxctl, and operation-staged qxctl are all valid arrangements to be made available when their exact contracts close. The cluster creator chooses among them for each setup.

IPC is the preferred remote command surface where the selected topology supports it; a scoped remote CLI or shell-mediated invocation may be a fallback. Neither wording authorizes a persistent listener, general-purpose shell exposure, or unbounded remote execution.

Bootstrap before an executable Habitat and command transport after one exists are separate research questions. The answer may vary by provider and owned-hardware setup without weakening performance doctrine.

## Thermal Boundary

qxctl and SOV operations may act upon a Node that also hosts hot-path work, but they are not continuous hot-path dependencies. Live changes are expected to be uncommon and explicitly invoked. No watcher, hidden loop, resident interpreter, or background package manager may be inferred.

## Compatibility and Failure

Version compatibility is exact. A working Nest bound to an older broker, data, compiler, package, Habitat, or adapter contract remains bound to that version until an explicit migration changes it. Current-version recency is not compatibility evidence.

An unavailable fact remains unresolved. Failure must not be converted into a guessed configuration, a substitute provider, a widened package surface, or a claim that an operation completed.

## Non-Authorization Statement

This specification defines architectural obligations only. It authorizes no executable qxctl leaf, provider credentials, Terraform apply, remote command execution, package mutation, bus traffic, Node reimage, or live Nest modification.
