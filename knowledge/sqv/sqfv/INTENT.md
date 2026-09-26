# Symphony Quantitative Flow Vector Intent

## Purpose

The Symphony Quantitative Flow Vector (SQFV) is SQV's owner of bounded research-data movement. It defines how prepared payloads pass through selected ports and transports with explicit resource budgets, ordering context, consumer credits, and safe release obligations.

## Composition Boundary

Each producer-consumer edge follows an exact selected contract. Compatible readers may share an immutable payload only while ownership and access conditions permit it. A direct connection may satisfy SQFV without a central broker, common bus, or compulsory resident dispatcher. Users select their topology and resource policies within the capabilities actually installed.

## Separation of Concerns

SQFV moves bytes and carries references to their meaning. SQMV owns dataset/schema/provenance interpretation; acquisition owns source observations; persistence owns durable commits; delivery owns the selected recipient view and its rights. SOOV retains its FIX framework scope.

## Non-Scope

SQFV does not choose providers, infer financial meaning, grant recipient access, decide retention, prescribe a strategy's hot path, or claim that a transport receipt proves processing or durability.
