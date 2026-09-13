# Focused connector process tests

`connector_test.py` covers native storage and projection boundaries. `installed_integration.py` covers the normal receipt-backed qxctl workflow. `interrupted_commit.py` adds eight deterministic interruption cases in the connector's own prepare/commit translation unit, followed by recovery through the exact installed production connector and SCV owner.

Build the separate fault executable explicitly:

```sh
cmake --build /absolute/build --target scv-graph-duckdb-commit-fault-test
python3 modules/scv-graph-duckdb-connector/tests/interrupted_commit.py --qxctl /absolute/qxctl --connector-prefix /absolute/connector-prefix --owner-prefix /absolute/scv-owner-prefix --graph /absolute/retained-graph.json --fault-engine /absolute/build/scv-graph-duckdb-commit-fault-test --out /absolute/new-evidence-directory
```

The selected packages are connector `0.1.0-dev` and SCV owner `0.10.0-dev`; the graph must be valid under that owner at the harness's explicit simulated time, `2026-09-13T05:00:00Z`. The harness makes no network request or source/graph-head selection. The output directory must be new. It contains private test databases and evidence; it is not a production index.

`SYMPHONY_SCV_GRAPH_INDEX_TEST_BARRIERS` is defined only for `scv-graph-duckdb-commit-fault-test`. The normal executable never links `commit_barrier.cpp`, has no runtime fault-control interface, and is the only connector target installed by CMake. The test executable uses a separate inherited pipe to announce the selected stage, then stops. The parent observes SIGSTOP, sends SIGKILL and reaps the child with a bounded timeout. No destructor rollback or graceful close runs in the killed writer.

The writer is an instrumented test build of the actual connector implementation, not the receipt-owned production executable. It uses a caller-supplied production Installation value in the test input; native mechanical validation does not authenticate that value. The installed qxctl recovery independently validates the exact production package and semantic owner. Evidence must retain both binary identities and never present the test writer as an authenticated production installation.

| Barrier | Expected durable state after process death |
| --- | --- |
| `prepare.before_commit` | Target operation absent; existing snapshot preserved; recover rejects absent intent and explicit import can retry. |
| `prepare.after_commit` | Exact prepared intent retained, target snapshot absent. |
| `commit.after_snapshot` | Prepared intent retained, uncommitted snapshot absent. |
| `commit.after_row` | Prepared intent retained, partially inserted row inventory not published. |
| `commit.before_commit` | Prepared intent retained despite in-transaction complete rows and intent update. |
| `commit.after_commit` | Fully committed snapshot retained despite missing response. |

The last two barriers also run when another operation has already published the exact same snapshot. Every case verifies an earlier sentinel snapshot, full native/Go export validation, exact owner replay, idempotent recovery and expected-intent mismatch refusal. No elapsed delay is used to guess a transaction boundary.

These are process-interruption checks at explicit boundaries surrounding DuckDB COMMIT. They do not interrupt inside DuckDB's COMMIT implementation, simulate power loss, prove filesystem/hardware durability, establish load performance or constitute the full SCV milestone suite. Keep actual results and hashes in the increment evidence rather than inferring execution from this coverage description.
