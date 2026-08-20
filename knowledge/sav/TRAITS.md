# SAV Trait Contract

## Purpose

A SAV trait is a stable, machine-evaluable capability requirement or provision used by Accord References, Named Versions, Extension Capsules, and Installation Blueprints. A trait is not a marketing label, runtime permission, package presence claim, or substitute for an SSFV feature.

## Identity

Trait IDs use `savtrait:<namespace>:<stable.dotted-key>` and never derive authority from a repository, package, organization, caller, or hostname. The first-party namespace is `savtrait:symphony:`.

## Minimum Declaration

Every trait declares:

- stable ID and title;
- owner contract;
- semantic meaning;
- evidence protocol and evaluation rule;
- compatibility version;
- thermal restriction;
- applicable platforms;
- lifecycle state;
- record digest;
- explicit non-claims.

## Initial First-Party Traits

- `savtrait:symphony:headless-json-administration`
- `savtrait:symphony:caller-neutral-authority`
- `savtrait:symphony:receipt-v2-exact-installation`
- `savtrait:symphony:maestro-receptor-presence`
- `savtrait:symphony:freezing-path-only`
- `savtrait:symphony:two-way-version-selection`
- `savtrait:symphony:durable-evidence-recovery`

These names are allocated by the ratified Accordare contract. A component provides one only when exact evidence satisfies its owner rule.

## Non-Authorization Statement

Trait presence cannot grant permission, imply installation or docking, satisfy an undeclared feature interaction, override a prohibition, or make partial coverage complete.
