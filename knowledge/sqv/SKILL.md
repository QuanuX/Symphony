# Symphony Quantitative Vector Skill

## Purpose

Keep Symphony framework design separate from third-party strategy decisions.

## Reading Order

1. `knowledge/ARCHITECTURE.md`
2. `knowledge/SLANG.md`
3. the SQV Contract Quad
4. the applicable SQV subvector Quad, including SOOV for FIX work
5. exact SOV, SNV, SCV, SHV, SACV, SSIAG, STAV, and SODV contracts implicated by the work

## Procedure

1. Identify the reusable framework concern without describing the user's strategy for them.
2. Separate FIX and non-FIX assumptions.
3. Preserve exact API, compiler, Habitat, hardware, and adapter versions.
4. Keep thermal and resource consequences visible without turning guidance into prohibition.
5. Ask the Architect for missing engine purpose, ownership, and naming decisions.

## Stop Conditions

Stop before inventing strategy requirements, feed declarations, subscription rules, broker semantics, FIX architecture, backtesting behavior, indicator mathematics, or public engine names.
