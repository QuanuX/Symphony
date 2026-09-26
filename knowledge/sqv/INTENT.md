# Symphony Quantitative Vector Intent

## Purpose

The Symphony Quantitative Vector (SQV) owns reusable Symphony framework contracts for quantitative research, trading-system construction, execution support, and later quantitative engines. It is a thin semantic parent: a child owns each admitted purpose and its exact interfaces.

## User Strategy Boundary

Strategy logic belongs to the user. SQV may supply reusable framework surfaces and exact compatibility evidence, but it does not determine a strategy's instruments, feeds, subscriptions, fields, depth, failure behavior, normalization, order logic, execution model, exhaust contents, or internal modular design.

## Scope

SQV is the home for ratified quantitative framework components. Its six admitted research-data architecture owners are Symphony Quantitative Acquisition Vector (SQAV), Metadata Vector (SQMV), Flow Vector (SQFV), Transformation Vector (SQTV), Persistence Vector (SQPV), and Delivery Vector (SQDV). Their purpose boundaries and cross-owner requirements are aligned in `knowledge/sqv/RESEARCH-DATA.md`. Architecture ownership establishes no installed data service, provider operation, package, or namespace.

Symphony Orchestra Omega Vector (SOOV) remains SQV's separate C++-only FIX child. Its scope is not absorbed by the six research-data owners.

## Non-Scope

SQV does not make every Nest a strategy, require a broker or data API, make non-FIX semantics universal, or prescribe live strategy data handling. SOOV retains FIX scope. Backtesting, broker, indicator, and mathematics contracts remain with their respective future or separate owners. The research-data expansion does not turn its selected data path into a compulsory trading path.
