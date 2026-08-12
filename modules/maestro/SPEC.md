# Maestro Receptor Presence Specification

## Status

Original lifecycle Step 7 development slice, version `0.1.0-dev`.

## Process

The executable accepts the common bounded `symphony.knowledge.engine-process.v1` standard-input/output envelope with an empty child environment and hard caller deadline.

## Operations

- `inspect`: returns an exact receptor descriptor and creates no state;
- `inventory`: authenticates one TOPS-wide read, enumerates every protected receptor registry, and returns one sorted derived inventory without creating or repairing state;
- `status`: returns protected current presence without creating a missing stream;
- `dock`: commits an exact receipt/executable relationship using expected registry state;
- `undock`: commits the absence of that active relationship using expected registry state;
- `recover`: repairs only a unique valid digest-linked slot selection.

## Durability

Each TOPS/receptor stream uses a private no-follow lock, alternating registry slots, an atomic head, file and directory synchronization, content digests, linked generations, and explicit recovery. A missing or invalid predecessor slot after generation one is never silently retained: reads require recovery, and recovery publishes a new forward generation only from unique valid evidence. Unknown newer or critical state, non-linked generations, and ambiguous equal-generation evidence fail closed. A live component identity cannot be replaced by different receipt or executable evidence; it must be exactly undocked before a different identity can dock.

## Derived Inventory

`inventory` is governed separately by `symphony.maestro.receptor-inventory-command.v1` and `symphony.maestro.receptor-inventory-result.v1`. It discovers only receptor directories beneath the selected TOPS Maestro state boundary, rejects unexpected names and unsafe ownership, mode, link, or file conditions, obtains a nonblocking shared lock for every stream, and validates each head, slot, digest chain, TOPS identity, receptor identity, and directory-key binding. If any existing stream is busy, damaged, ambiguous, or incompatible, the complete inventory operation fails; it never returns a partial view as empty or complete.

Receptors and components are sorted by stable identity. `inventory_digest` covers only the stable receptor/component state, including exact registry, receipt, executable, and presence digests. `observed_at` is canonical whole-second UTC and appears only in the outer result; `observation_digest` covers the timestamped envelope. A later identical observation therefore preserves the stable inventory digest while producing new observation evidence. Inventory is `derived: true`, `read_only: true`, and `canonical: false`; it is not persisted as a second registry.

## Authorization

Inventory, status, dock, undock, and recovery validate fresh exact SSIAG decision and capability evidence. `inspect` is safe metadata. No caller class is an authorization input.

## Non-Authorization

Docking is persisted deployment presence only. It does not start an engine, authorize semantic work, change a package receipt, activate a service, mutate canonical knowledge, or create a hot/warm dependency.
