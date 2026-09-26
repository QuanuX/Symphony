# Symphony Quantitative Vector Intent

## Purpose

The Symphony Quantitative Vector (SQV) owns reusable Symphony framework contracts for quantitative research, trading-system construction, execution support, and later quantitative engines. It is a thin semantic parent: a child owns each admitted purpose and its exact interfaces.

## User Strategy Boundary

Strategy logic belongs to the user. SQV may supply reusable framework surfaces and exact compatibility evidence, but it does not determine a strategy's instruments, feeds, subscriptions, fields, depth, failure behavior, normalization, order logic, execution model, exhaust contents, or internal modular design.

## Scope

SQV is the home for ratified quantitative framework components. Symphony Orchestra Omega Vector (SOOV) remains its C++-only FIX child. Symphony Quantitative Metadata Vector (SQMV) and Symphony Quantitative Flow Vector (SQFV) are admitted research-data architecture owners. They describe data and govern its bounded movement, respectively; neither is a general data service or an installed runtime.

Research-data acquisition, transformation, persistence, and delivery are distinct intended purposes under the favored labels SQAV, SQTV, SQPV, and SQDV. Their labels and requirements are recorded in `knowledge/sqv/RESEARCH-DATA.md`; their child Quads and machine routes await a compatible SKVI capacity change. Naming a planned purpose does not admit an owner, operation, package, or namespace.

## Non-Scope

SQV does not make every Nest a strategy, require a broker or data API, make non-FIX semantics universal, or prescribe live strategy data handling. SOOV retains FIX scope. Backtesting, broker, indicator, and mathematics contracts remain with their respective future or separate owners. The research-data expansion does not turn its selected data path into a compulsory trading path.
