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

## First Native Slice

`modules/sqfv-batch-cpp/SPEC.md` defines the admitted `0.2.0-dev` library contract for one trusted address space. It exposes a native C++26 API with owning and borrowed standard-library types and move-only lifetime handles; a caller-resolved, immutable five-field metadata binding; a separate opaque source-native position; a partition/generation/sequence transfer cursor; one copied immutable payload with explicit batch and lease lifetimes; independent per-port byte credits and pending-entry limits; and an integrity-checked local `SQF1` frame. Its technical ceilings and numerical fixture belong to that module, not every SQFV implementation. The scope comparison is a same-process compatibility guard, not an access grant. Queue acceptance, read-lease release, destination processing, and durable commit remain separate facts.

The offline prototype's development frame and one-megabyte payload bound are not the module's protocol. The module's C++ interface and local frame do not admit IPC, networking, shared-memory reclamation, or a cross-Node transport.

## Deferred Technical Contract

Producer restart and generation evidence, retained replay, live-to-retained cutover, device fencing, access enforcement, cross-process handles, transport adapters, and a supported platform or throughput matrix remain separate gates. A production SQMV reference grammar and dataset interpretation also remain with SQMV.

## Non-Authorization Statement

SQFV establishes no provider, schema, access, durability, destination receipt, universal exactly-once, or performance claim. It imposes no required feed path, bus, cache service, or execution design on user-authored programs.
