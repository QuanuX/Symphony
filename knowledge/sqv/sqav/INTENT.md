# Symphony Quantitative Acquisition Vector Intent

## Purpose

The Symphony Quantitative Acquisition Vector (SQAV) owns research-data collection from user-selected, exact source interfaces beneath SQV.

## Scope

SQAV defines source and collection bindings, bounded requests and subscriptions, acquisition positions, reconnect and supported backfill behavior, and attributable observations and gaps produced by collection. First-party connectors and parsers for this data path are native C++.

The initial source programme includes FRED/ALFRED, a news provider still to be selected, Databento, and selected read-only IBKR research surfaces through a future SCABV-owned non-FIX adapter. These are intended integration families, not implemented provider claims or selected exact operations.

## User Choice and Neighbor Boundaries

The user selects sources, datasets, instruments where relevant, collection mode, interval, and permitted resource limits. SQAV preserves provider-native meaning and exact interface, adapter, and dependency versions. It consumes broker-specific identity, authentication, symbol, and protocol contracts from their owner; the SCABV design remains a draft until admitted.

SQAV does not own FIX sessions (SOOV), broker account truth, user strategy logic, general data movement (SQFV), dataset catalogue meaning (SQMV), conversion (SQTV), retention (SQPV), or consumer delivery (SQDV). A source access grant and collection authority do not imply order-entry or redistribution authority.
