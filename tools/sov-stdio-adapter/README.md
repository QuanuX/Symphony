# SOV local process adapter 0.1.0-dev

Optional, independently built Go executable for the qxctl SOV controller profile on macOS/Linux. Select exact adapter and target qxctl bytes through `qxctl sov remote invoke`. It forwards one request and response and never implements backtesting calculations or grants access.

From repository root, build to a caller-owned destination with `go build -o /chosen/prefix/bin/sov-stdio-adapter ./tools/sov-stdio-adapter`. Record its SHA-256 in the invocation and plan. Remove that selected executable to uninstall; no service, shared runtime state or SBV installation belongs to this adapter. Configuration lifecycle is `qxctl sov target configure|inspect|remove`. Removing a target configuration disables subsequent admissions; it does not cancel already admitted jobs. Use `job_cancel` explicitly.

The fixed process arguments are `--target`, `--target-sha256`, `--configuration`, and `--configuration-sha256`; stdin is the complete invocation, stdout the complete target result. The selected target executable is streamed into a private temporary file and hashed before execution. All scientific requests use exact installed native SBV through target qxctl. No shell, listener, provider request, provisioning, secret transfer or network transport is installed.

SIGINT/SIGTERM propagates cancellation; 250 ms bounds process/pipe cleanup after cancellation or exit, not calculation duration. There is no default calculation deadline. A lost response never proves native termination or licenses automatic retry. The controller inspects/reconciles the same durable job; trusted user-selected code may escape process groups or have external effects.

See [execution contract](../../knowledge/sov/PROCESS-EXECUTION.md). This local adapter is conformance evidence for the open protocol, not proof of a remote host or cluster.

For a selected authenticated remote carrier, optional `--self-sha256` checks this launcher against an exact expected executable digest before staging target qxctl. It detects installation drift within a trusted host; it is not hardware attestation. Existing four-argument local invocations remain compatible.
