# Administration and authority

## Three distinct boundaries

User authorization states what work the agent may do. The bridge's operator configuration decides which command identities this integration may dispatch. The operation owner, including SSIAG or STAV where applicable, enforces native access and state requirements. None of these replaces the others.

Being an AI, a human, a service, or a workload does not itself grant or remove authority. Do not create extra universal restrictions on user-authored systems. Scope required checks to the selected command and installation.

## Prepare and apply

For supported non-read commands:

1. Discover and describe the exact command.
2. Resolve the target and required arguments from the user's request and current owner evidence.
3. Call `symphony_prepare` with `command_id`, `flags`, and `positionals`.
4. Review the returned plan. It must express the authorized operation, exact argument values, and intended target. Identify any actual destructive or externally consequential effect before proceeding.
5. Call `symphony_apply` with that plan's `plan_id` when the user's authorization covers the operation.
6. Inspect native outcome and retained state where the command's contract requires it.

The plan is an ephemeral bridge transport preview. It does not execute native validation, resolve a selected deployment, obtain credentials, reserve resources, certify input files, or grant permission. It is not evidence that the operation will succeed. A plan ID is not a native policy grant, operation ID, expected-state digest, or install receipt.

Plans expire after five minutes and are consumed on an application attempt. Direct existing regular-file flag values can be digest-bound by the preview; that does not snapshot positional file arguments, nested dependencies, or mutable native state. If a plan expires or an input changes before dispatch, inspect current evidence and prepare the intended operation again. After uncertain dispatch, reconcile native state before preparing another mutation.

If the native workflow requires a proposal file followed by native apply, bridge preparation/application is used for each non-read command in that workflow. Do not skip the native planning stage because a bridge plan exists. Conversely, do not force a native proposal stage onto a command whose actual contract does not have one.

## Exact installation and lifecycle changes

Use registry discovery to locate the relevant installation, binding, lifecycle, and removal operations. Confirm the selected prefix, module role, version, and receipt from current evidence. Keep side-by-side installations distinct. Neither a listed module nor a source checkout proves the corresponding engine is installed.

Use the owner's expected-state precondition and recovery mechanism. On a stale digest or concurrent change, inspect current state and reassess the user's intended operation. Do not replace the digest automatically to force a retry. Do not choose a different engine/version simply because the requested one is missing.

The package's command enablement is an integration access boundary. A disabled command may require an operator to edit the bridge configuration and restart it. Do not add commands automatically while executing an unrelated user request. An explicit request to configure access is separate authorized work and should identify the exact new capabilities being enabled.

## Identity and audit

Preserve TOPS identity, user/system scope, subject and provider identity, exact grants, expected state, and required audit evidence. Native SSIAG/STAV communication belongs to those owners. The bridge neither owns raw audit append authority nor substitutes an MCP plan ID for native authentication.

A source policy proposal is not a grant. A successful local peer check is not a remote principal grant. A recorded audit event is not proof that every intended downstream effect completed. Do not export credentials, signed material, raw private provider state, or authentication storage as part of ordinary diagnostics.

## Effects outside the installation

For provider requests, publication, deployment, paid acquisition, or trading-related actions, preserve the user's explicitly selected target and effects. Check the actual installed command's capabilities and authorization requirements. A data contract, backtest result, or disconnected plan does not imply a live provider or broker operation exists.

Do not introduce an approval ritual where the user already authorized the operation. If a material destination, cost, destructive effect, or authority is missing, make the proposed action concrete and obtain that missing information before applying it.

## Extension

Use registered commands and owner schemas for new domains. Avoid hard-coded synonyms and guessed command grammar. Extending this bridge must preserve its argv boundary, configuration restrictions, protocol semantics, and native owner authority; the underlying quantOS remains open to independently designed tools and modules.
