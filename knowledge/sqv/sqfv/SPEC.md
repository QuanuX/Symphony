# Symphony Quantitative Flow Vector Specification

## Movement Contract

SQFV owns bounded ports, immutable prepared-payload transfer, queue admission, partition and producer-generation context, per-consumer cursors and credits, release obligations, and selected in-process or transport adapters. Source-native position and internal transfer cursor are distinct; neither implies a total order across providers or event-time completeness. A finite frame and resource reservation are required even when a selected collection session is long-lived.

A producer may freeze a prepared payload and let compatible trusted readers hold explicit read leases. Publication ends mutation of that allocation; a transformation that changes bytes creates a distinct result. A reader or asynchronous device keeps its access obligation until actual release or a verified fencing procedure. A timeout or disconnected control channel alone does not make borrowed memory reusable.

## Resource and Failure Boundary

Each receiving edge has finite capacity and an independent cursor. Credits represent a stated resource unit and return at the defined release point; one blocked optional consumer must not silently consume another consumer's budget. Oversize frames, exhausted budgets, cancelled transfers, gaps, and stalled readers require explicit outcomes. A backlog cannot grow without a declared bound or be called lossless after data is discarded.

Control and release paths must have a bounded means to make progress when data queues are full. Global and local budgets, reserved control capacity, oversize and chunking behavior, lag policy, and live-to-retained catch-up require exact later admission. The planned SQPV persistence role would own retained replay and the durability boundary once admitted; SQFV may carry the corresponding cursor without claiming that storage was committed.

## Locality and Trust

An in-process lease is valid only under its reviewed trust and toolchain contract. A copied descriptor does not duplicate the payload or its release obligation. Cross-process and cross-Node movement require separately admitted handles, frame versions, integrity, access scope, generation, and reclamation rules. A raw pointer or C++ implementation-private container is not a portable process boundary. A read-only mapping cannot expose unrelated private bytes merely because a logical view hides them.

SQFV transports a resolved SQMV descriptor or reference without defining its dataset meaning. It does not select recipients or grant rights; recipient selection awaits the planned SQDV delivery contract and the applicable access owner. Finite SKV/qxctl administration may configure a selected flow but does not relay bulk records or become an implicit dependency of hot or warm strategy execution.

## Deferred Technical Contract

The exact production descriptor and frame, batch identity grammar, credit units, lifetime/fencing mechanism, target workload, supported localities, and transport adapters remain to be ratified. The offline prototype demonstrates only bounded in-process mechanics under its stated test environment; this Quad does not promote its development frame or one-megabyte payload bound to a released protocol.

## Non-Authorization Statement

SQFV establishes no provider, schema, access, durability, destination receipt, universal exactly-once, or performance claim. It imposes no required feed path, bus, cache service, or execution design on user-authored programs.
