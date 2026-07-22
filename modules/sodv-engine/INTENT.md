# SODV Engine Intent

## Purpose

Implement bounded, provider-neutral SODV release-ledger validation, observed-state comparison, forward proposal, recovery reconciliation, and disposable projection without publishing or mutating canonical truth.

## Authority Boundary

`knowledge/sodv/` owns the protocol and release truth. The engine is a freezing-path implementation that computes deterministic facts from canonical files and caller-supplied observations. It neither discovers authority nor converts evidence into ratification.

## Operational Outcome

The module gives every permission-bearing caller the same independently installable local process for detecting release-ledger drift, distinguishing unpublished/waiting/ready/completed/mismatched state, preparing one forward v2 record proposal, resuming an interrupted local transaction safely, and building a disposable release inventory.

## Non-Goals

The engine is not a Git forge client, package-provider client, publisher, tag writer, ledger writer, public-documentation generator, Mintlify integration, NotebookLM automation, session authority, or Maestro receptor.
