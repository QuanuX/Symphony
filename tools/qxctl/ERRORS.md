# qxctl SCV failure output and help

Status: implemented local CLI contract. Owner: qxctl. Protocol:
`symphony.qxctl.error.v1`. This contract covers root failures from the `scv`
command family when JSON output is requested; it does not change the success
payloads of exact installed engine versions or the evidence protocols returned
by other qxctl command families.

## Failure envelope

An unsuccessful SCV invocation with `--json` emits one JSON object followed by
a newline on standard output. Cobra error and usage printing are suppressed.
No additional human diagnostic accompanies the envelope. The exit status is
nonzero and equals `exit_code`.

```json
{
  "protocol": "symphony.qxctl.error.v1",
  "outcome": "error",
  "command_id": "qxcmd:symphony:scv.provider.interpret",
  "error": {
    "code": "engine_rejected",
    "message": "The selected engine rejected the request.",
    "engine_code": "interpretation.invalid"
  },
  "exit_code": 1
}
```

The object and its nested `error` object have exactly the fields shown. The
protocol and outcome are fixed. `command_id` is the trusted registered command
identity selected by the command tree, or `null` when no executable command was
selected. Caller-provided command text is never copied into this field.
`exit_code` is an integer from 1 through 125. Normal SCV failures retain exit 1;
an engine's own process exit status is not adopted as the CLI status.

| `error.code` | Exact `message` | Meaning |
| --- | --- | --- |
| `invalid_arguments` | Select a command and supply valid arguments; use its --help for accepted flags. | A command-selection, positional-argument, or flag-parsing failure. |
| `engine_rejected` | The selected engine rejected the request. | The established engine client returned a typed `ProcessError`. |
| `command_failed` | The command could not complete; no successful result is available. | Other local failures, including unreadable/malformed input, absent or invalid installation, authority rejection, or result-validation failure. |

`command_failed` is deliberately conservative. Untyped error prose is not
parsed to invent a narrower diagnosis. A missing input flag can therefore
produce this code when the operation's input reader detects it after Cobra
parsing. It is not evidence that a protected operation had no partial local
effect: retained intents and recovery contracts remain authoritative.

`engine_code` is `null` except for `engine_rejected` with a recognized code from
the finite allowlist in [cli_errors.go](cmd/qxctl/cli_errors.go). That allowlist
preserves existing codes such as `interpretation.invalid`, `corpus.invalid`,
`knowledge.invalid`, and the explicit `scv.*` source-validation codes. Unknown
codes remain `null`; syntactically valid arbitrary identifiers are insufficient.
The fixed envelope never includes raw error messages, source text, input
content, file paths, engine stderr, credentials, or caller-supplied flag values.

Identical selected command, failure class, allowed engine code and exit status
produce identical envelope bytes. There is no timestamp, generated identity,
digest or authorization assertion. This is a failure transport protocol, not a
successful SCV evidence artifact. Command result-validation protocols continue
to describe the operation's successful validated output.

## Output intent and compatibility

For the normal `qxctl scv ...` grammar, `--json`, `--json=true`, and other true
boolean forms request the envelope even when an earlier flag fails to parse.
`--json=false` and other valid false forms select the existing human failure
path. The last explicit JSON flag before the `--` terminator determines output
intent. An invalid explicit boolean such as `--json=invalid` requests a machine
argument error. A value following `--json` without `=` is positional, as in the
existing Cobra boolean-flag grammar.

The output-intent reader uses the current SCV flag definitions to skip consumed
values. Thus `--prefix --json` supplies the literal string `--json` as the prefix;
it does not request JSON. Likewise `--prefix=--json` is a string value, and a
`--json` token after `--` is positional. This reader only chooses error rendering;
it does not validate misplaced flags or alter the accepted command grammar.

Successful payloads remain byte-for-byte those printed by their existing
handlers, without an outer success wrapper. An already-emitted validation
outcome or exact evidence exit sentinel emits no second envelope. Existing
specialized validated exit statuses in the range 1–125 remain unchanged, with
the existing fallback to 1 outside that range. Other command families retain
their established error behavior. Successful `--help` requests remain human help.

## Human discovery

Root and namespace help enumerate every visible registered executable leaf
beneath the selected command, sorted by command path. Hidden and retired
commands are excluded. Leaf help displays its actual flag definitions and exact
version defaults. Both views are derived from the same current Cobra tree that
owns the `CommandSpec` records; there is no separate static command list or
manually synchronized help fixture. `qxctl commands manifest --json` remains the
machine discovery surface.

## Verification

The CLI tests cover missing and malformed input, absent installation, unknown
and incomplete flags, explicit false and repeated JSON flags, consumed flag
values and terminators, secret-marker exclusion, deterministic single-document
output, allowlisted engine codes, preserved evidence/status behavior, and help
coverage of every public registry leaf. The installed-process test invokes an
exact 0.3.0-dev engine to verify `interpretation.invalid` and an unchanged
successful source-status payload. The exact-version fixture is enabled by
`SYMPHONY_SCV_INTERPRETATION_PREFIX`; it never substitutes a newer installation.
