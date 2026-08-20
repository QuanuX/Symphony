# SAV Engine Intent

## Status

Implemented development foundation.

## Purpose

`symphony-sav` is the independently installable C++26 freezing-path implementation of the SAV protocol. It validates Accord References, resolves coverage-qualified CURRENT snapshots from caller-supplied projections, evaluates the closed rule algebra, and emits deterministic read-only diffs, explanations, compatibility results, and disposable graphs.

## Boundaries

The engine performs no ambient discovery, canonical mutation, installation, selection, docking, authorization, network access, telemetry, or hot/warm-path work. qxctl is the preferred headless administrator, but direct bounded process IPC remains supported.
