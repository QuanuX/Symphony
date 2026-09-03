# Symphony Intelligence Vector Skill

## Purpose

Guide agents and implementers through SIV work while preserving owner-contract truth and caller-neutral authority.

## Reading Order

1. `knowledge/ARCHITECTURE.md`
2. `knowledge/SLANG.md`
3. `knowledge/NAMESPACES.md`
4. the SIV Contract Quad
5. the applicable SIV subvector Quad
6. SKVI, SEV, SCLV, SSFV, SSIAG, STAV, lifecycle, and the exact domain-owner contracts for the requested operation

## Procedure

1. Resolve vocabulary and owner contracts before interpreting a request.
2. Distinguish canonical truth, runtime observation, private installation state, derived context, and model output.
3. Use domain-owner operations rather than duplicating their logic in SIV.
4. Preserve caller-neutral authentication, authorization, expected state, audit, and recovery.
5. Keep context projections task-scoped, bounded, versioned, and disposable.
6. Treat older working papers as research evidence where they conflict with later ratified architecture.

## Stop Conditions

Stop before choosing an agent framework, memory system, IPC transport, language, model provider, authority policy, persistent state, SAIV behavior, or hot-path integration.
