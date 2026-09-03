# Symphony Quantitative Vector Intent

## Purpose

The Symphony Quantitative Vector (SQV) owns reusable Symphony framework contracts for quantitative research, trading-system construction, execution support, and later quantitative engines.

## User Strategy Boundary

Strategy logic belongs to the user. SQV may supply reusable framework surfaces and exact compatibility evidence, but it does not determine a strategy's instruments, feeds, subscriptions, fields, depth, failure behavior, normalization, order logic, execution model, exhaust contents, or internal modular design.

## Scope

SQV is the future home for ratified quantitative framework components. Symphony Orchestra Omega Vector (SOOV), the C++-only high-performance FIX architecture, is its first named subvector.

## Non-Scope

SQV does not make every Nest a strategy, require a broker or data API, make non-FIX semantics universal, or define deferred backtesting, broker, indicator, mathematics, or data-stream architecture before their respective contracts are ratified.
