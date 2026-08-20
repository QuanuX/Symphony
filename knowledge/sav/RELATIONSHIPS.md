# SAV Relationship Vocabulary

## Purpose

This file owns the closed v1 relationship vocabulary evaluated by SAV. A relationship is meaningful only when an Accord Reference identifies its owner contract, subject, object, applicability, evidence protocols, and evaluation rule.

## Relationship Types

| Type | Meaning |
|---|---|
| `depends_on` | The subject requires the object before the declared capability can be valid or ready. |
| `implements` | The subject supplies behavior governed by the object contract. |
| `administers` | The subject exposes a reviewed administration interaction for the object. |
| `invokes` | The subject calls the exact object operation under a declared protocol. |
| `validates` | The subject verifies a closed object result or invariant. |
| `compatible_with` | The subject and object satisfy the declared compatibility profile. |
| `selected_as` | Protected state selects the exact subject for the declared role. |
| `docked_at` | Maestro persists exact subject presence at the object receptor. |
| `requires_trait` | The subject requires one registered SAV trait. |
| `provides_trait` | The subject provides one registered SAV trait with evidence. |
| `supersedes` | The subject is the declared successor of the object without erasing it. |
| `evidenced_by` | The subject assertion is supported by the exact object evidence. |
| `prohibited_from` | The subject is forbidden from the object behavior or thermal surface. |

## Evaluation Result

Each applicable relationship evaluates independently to `in_accord`, `out_of_accord`, `unknown`, or `not_applicable`. Missing or incompatible evidence is `unknown`; it is not a definite contradiction. A relationship rule never authenticates, grants permission, activates a package, or changes a binding.

## Extension

New relationship types require a compatible schema revision or new major protocol, explicit owner semantics, deterministic evidence rules, SSFV administration review, SKVI routing, and Architect ratification. Unknown types fail closed.
