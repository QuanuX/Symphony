# SCV DuckDB Graph Connector Intent

Supply an independently installable C++ connector that stores and indexes explicitly supplied SCV graph revisions in a caller-selected local DuckDB database and namespace. The connector preserves exact native graph identity and supports bounded retrieval, export and recoverable publication. qxctl is the operating interface for human and agentic callers.

Duncan selected C++ DuckDB as the supplied SQL database default. It remains a selected backend implementation, not a mandatory backend for every user composition. SCV retains provider, evidence, inference and composition semantics. Database rows are a rebuildable projection; storing or querying them grants no provider authority and selects no source, corpus or graph head.

The common storage and lifecycle boundary may inform future connectors and SHV integration. It does not allocate an SHV ontology, transfer hardware or Node ownership, require every vector to use this database, or insert database access into a user's trading path.
