# Symphony Quantitative Transformation Vector Specification

## Operation Contract

Each admitted operation declares supported input and output schemas and representations, exact operation release and parameters, required source evidence, numerical and timestamp rules, bounded workspace, and explicit failure results. Unsupported or ambiguous conversion fails without silent clipping, rounding, coercion, or field loss.

## Meaning and Loss

Lossless representation conversion preserves all meaning required by its declared domain and records the mapping. Information-reducing conversion identifies removed or narrowed meaning and requires an explicit user-selected allowance with attributable loss evidence. A research transformation, such as event aggregation or a join, creates a separately identified derived dataset with its input and operation lineage; it is not described as mere serialization.

Null, missing, sentinel, numeric scale, overflow, units, timestamp precision and calendar, character encoding, symbol identity, and nested structure are distinct where the selected source provides them. No operation is a universal converter merely because its format names match.

## Preparation and Recovery

Compatible consumers may share an immutable prepared result only for the same exact input snapshot, operation release and parameters, output representation, and permitted access scope. A different scope or changed input requires a distinct result. SQFV owns published buffer lifetime and release; SQTV owns the semantics of preparation and any attributable transformation cache.

Stateful windows and joins require separate partition, lateness, checkpoint, input-position, and restart contracts before support is claimed. No stateful operation or performance guarantee is admitted by this architecture Quad alone.

## Deferred Technical Contract

The first converter pair, analytic format, numeric rules, stateful-operation extent, target platform, public ABI, and resource thresholds require separate admission. The offline SQV prototype does not settle any conversion semantics. This Quad creates no installed capability or namespace family.

## Non-Authorization Statement

SQTV does not authorize access to source records, information loss, persistent storage, destination delivery, strategy execution, or reinterpretation of an earlier dataset revision.
