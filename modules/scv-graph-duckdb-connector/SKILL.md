# SCV DuckDB Graph Connector Working Guidance

Read `knowledge/scv/GRAPH-INDEX.md`, this module's specification, its installed schema and the exact selected executable descriptor. Supply the caller's database and namespace explicitly. Preserve graph, source, capture, owner and connector identities without relabeling any of them.

Use only the connector's finite operations. Keep raw SQL, database schema changes and extension loading outside its supported payloads. Resume an unfinished import using its retained exact identity and original inputs; do not reuse its operation identity for another graph or namespace. Report prepared and committed publication separately.

Treat index matches as stored graph structure, not newly validated provider facts or fresh evidence. Native semantic evaluation remains with the exact SCV owner and caller-selected query time and policy. Keep unsupported queries and backend limitations visible. No index operation selects a protected source or graph head.

Run focused checks for changed connector code and affected producer/consumer boundaries. Report actual test execution separately from declared test names. Preserve original graph artifacts, installed receipts and databases not selected for the current task.
