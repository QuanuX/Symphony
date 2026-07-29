# Symphony Semantic Feature Vector Namespace Registry

## Purpose

Allocate stable SSFV identity prefixes without coupling feature identity to repository paths, package registries, Git providers, hostnames, or graph databases.

## Namespace Entry Model

Each entry contains:

- `namespace`: lowercase namespace token;
- `id_prefix`: exact stable-ID prefix;
- `owner_contract`: canonical contract governing allocations;
- `scope`: bounded allocation purpose;
- `status`: `canonical`, `deprecated`, or `retired`;
- `evidence`: canonical ratification evidence;
- `notes`: safe context and non-claims.

## Canonical Namespace Entries

- namespace: `symphony`
- id_prefix: `ssfv:symphony:`
- owner_contract: `knowledge/ssfv/SPEC.md`
- scope: first-party application and platform feature identities owned within QuanuX/Symphony
- status: `canonical`
- evidence: `knowledge/ssfv/INTENT.md`
- notes: This allocation is an internal stable-ID namespace. It claims no URI scheme, trademark, package, repository, or network authority.

## Allocation Rules

New namespaces require explicit owner-contract ratification, collision review, and SKVI indexing. A namespace MUST NOT be inferred from an organization name, Git host, package coordinate, or caller identity.

Removing or reusing an allocated prefix is prohibited. Deprecated and retired namespaces remain recorded for identity lineage.

## Non-Authorization Statement

This registry allocates identifiers only. It does not declare any feature implemented, create a runtime principal, reserve an external name, or authorize engine installation.
