# Symphony Quantitative Vector Skill

## Purpose

Keep Symphony framework design separate from third-party strategy decisions.

## Reading Order

1. `knowledge/ARCHITECTURE.md`
2. `knowledge/SLANG.md`
3. the SQV Contract Quad and `knowledge/sqv/RESEARCH-DATA.md` for research-data work
4. the applicable admitted child Quad: SQMV for metadata, SQFV for flow, or SOOV for FIX
5. exact SOV, SNV, SCV, SHV, SACV, SSIAG, STAV, and SODV contracts implicated by the work

## Procedure

1. Identify the reusable framework concern without describing the user's strategy for them.
2. Route research-data requirements to their purpose owner; keep planned labels distinct from admitted child Quads and operational modules.
3. Separate FIX, non-FIX broker, and independent-source assumptions.
4. Preserve exact API, compiler, Habitat, hardware, and adapter versions.
5. Keep thermal, memory, credit, retention, and privacy consequences visible without turning guidance into prohibition.
6. Record a gate decision where a missing identity, guarantee, workload, or authority affects correctness or compatibility.

## Stop Conditions

Stop before inventing strategy requirements, feed declarations, subscription rules, broker semantics, FIX architecture, backtesting behavior, indicator mathematics, public engine names, colon namespaces, qxctl commands, or operational support. An offline prototype and a favored child label do not supply those admissions.
