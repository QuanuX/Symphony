# Maestro Manifest

## Identity

- module: `maestro`
- executable: `symphony-maestro`
- development version: `0.1.0-dev`
- language: C++26
- thermal path: freezing

## Contract

Implements `symphony.maestro.knowledge-engine-docking.v1` plus `symphony.maestro.receptor-inventory-command.v1` using the common bounded local process envelope. One exact receptor owns a protected digest-linked presence registry per TOPS; the inventory operation derives one deterministic sorted TOPS-wide view without becoming writable state.

## Installation

The module is independently buildable, installable, rollbackable, and uninstallable. Its immutable receipt describes nine owned files. Uninstall preserves operational presence state.

## Boundaries

Maestro is the sole supported writer of Maestro docking presence. Its inventory reader fails closed if any discovered receptor stream is unsafe, busy, ambiguous, or unreadable, so an incomplete view cannot masquerade as complete. qxctl administers the process; the knowledge-session coordinator prepares and verifies lifecycle actions but never writes Maestro state. The module does not invoke engines, execute receipt entry points, supervise services, mutate packages, append STAV, or own semantic truth.
