# Accordare STAV Producer Intent

## Purpose

Close the durable audit circuit for the four ratified SAV Named Version mutations without giving qxctl, the SAV engine, or the Knowledge Session Coordinator arbitrary STAV append authority.

## Authority Boundary

`knowledge/stav/` owns the event vocabulary and STAV envelope. `knowledge/sav/` owns Named Version meaning. SSIAG owns permission decisions. The coordinator owns Named Version persistence. This module verifies their exact evidence, derives one safe candidate from a closed mapping, and submits it through the existing STAV authority. It owns none of those upstream truths.

## Operational Intent

- one separately authenticated producer process with private intent and candidate stores per TOPS;
- Darwin/Linux kernel peer credentials in both IPC directions;
- exact SSIAG subject, target, capability, expiry, and caller-neutrality verification;
- no caller-selected event class, operation, target kind, outcome, reason, request ID, or correlation ID;
- durable intent before coordinator mutation, followed by a durable candidate before append;
- deterministic exact-command retry with a fresh authorization proof and no guessed outcome;
- explicit intent-pending, append-pending, or committed audit disposition;
- independent receipt-v2 installation and conservative per-TOPS enrollment;
- qxctl-owned installation grant administration with expected-state CAS and forward recovery;
- independent native supervision administered through exact receipt-v2 qxctl routes;
- freezing-path operation only.
