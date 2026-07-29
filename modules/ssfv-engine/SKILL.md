# SSFV Engine Skill

## Purpose

Use `symphony-ssfv` through qxctl to inspect and validate SSFV truth, compare a prior semantic snapshot with live canonical state, create a noncanonical caller-declared proposal, and build a disposable graph.

## Procedure

1. Read `knowledge/ssfv/` and identify the exact installed version and prefix.
2. Run `qxctl ssfv inspect` and confirm all authority and mutation surfaces remain false or disabled.
3. Run `qxctl ssfv check`; preserve its semantic snapshot when later freshness or diff evidence is required.
4. Use freshness `report` or `require` only with a prior snapshot.
5. Use `diff` for baseline-versus-live evidence, not for feature-worthiness acceptance.
6. Use `propose` for one namespace or feature operation and inspect every expected-state and desired-change digest.
7. Use `graph` only as a noncanonical, rebuildable JSON projection.

## Fail-Closed Conditions

Stop on stale expected digests, unsafe or symlinked paths, ambiguous managed markers, malformed records, duplicate identities, routing inconsistency, missing SKVI coverage, invalid hierarchy, cycles, missing relationship targets, excessive resources, response overflow, or expired deadlines.

## Boundaries

Treat every result as evidence. The engine does not authenticate, authorize, ratify, apply, create feature files, install hooks, persist graphs, contact external providers, or dock with Maestro.
