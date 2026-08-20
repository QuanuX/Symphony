# SCSEV qxctl Command-Surface Evolution Profile

## Status

Architect-ratified first SEV profile. SCSEV has no separate engine or canonical command registry in v1.

## Purpose

SCSEV evaluates the complete repercussions of adding, changing, deprecating, replacing, or retiring one qxctl executable leaf. It reuses SSFV feature administration and reports an explicit gap when an independent module author omits the qxctl layer.

## Required Inputs

- exact source SAV CURRENT snapshot;
- SSFV semantic snapshot;
- `knowledge/FEATURE-ADMINISTRATION-PROFILE.json` evidence;
- canonical expected qxctl registry;
- optional observed qxctl registry;
- affected engine descriptor v2 objects;
- caller-declared command-change proposal;
- applicable owner contracts and invariant registry digests.

## Consequence Families

Every proposed leaf must account for:

1. stable `qxcmd:` identity and collision review;
2. exact grammar, aliases, visibility, and tombstone behavior;
3. SSFV feature and interaction binding;
4. every required `engop:` backend binding;
5. mutability, target scope, and SSIAG authority mode;
6. input, output, and result-validation protocols;
7. noninteractive and JSON behavior;
8. expected-state, apply-status, and recovery where mutation requires them;
9. Cobra construction and expected/observed manifest parity;
10. help-golden, client, producer, consumer, integration, and recovery tests;
11. invariant ownership and real-process evidence where applicable;
12. SKVI, SSFV, SCLV, root-summary, and documentation consequences;
13. thermal placement and forbidden-path noninterference;
14. compatibility, deprecation, replacement, and rollback consequences.

## States

The profile reuses feature-administration design, live, authorization, and module-integration states. A newly implemented module with omitted semantic or qxctl evidence is `semantic_registration_required` or `administration_unintegrated`, never silently complete.

## Remediation Boundary

SCSEV may return missing fields, allowed syntax, collision evidence, applicable identity families, required protocol classes, and test obligations. The engine MUST NOT invent the final feature name, command ID, grammar, exemption, authority decision, or canonical patch. Optional AI assistance remains an ordinary authorized caller.

## Non-Authorization Statement

Passing SCSEV does not implement a command, install qxctl, authenticate a subject, grant permission, make a module docking-ready by itself, or authorize canonical apply.
