# Accordare STAV Producer Intent

## Purpose

Close the durable audit circuit for the four ratified SAV Named Version mutations without giving qxctl, the SAV engine, or the Knowledge Session Coordinator arbitrary STAV append authority.

## Authority Boundary

`knowledge/stav/` owns the event vocabulary and STAV envelope. `knowledge/sav/` owns Named Version meaning. SSIAG owns permission decisions. The coordinator owns Named Version persistence. This module verifies their exact evidence, derives one safe candidate from a closed mapping, and submits it through the existing STAV authority. It owns none of those upstream truths.

## Operational Intent

- one separately authenticated producer process and private outbox per TOPS;
- Darwin/Linux kernel peer credentials in both IPC directions;
- exact SSIAG subject, target, capability, expiry, and caller-neutrality verification;
- no caller-selected event class, operation, target kind, outcome, reason, request ID, or correlation ID;
- durable outbox before append and deterministic retry identity;
- explicit committed or pending audit disposition;
- independent receipt-v2 installation and conservative per-TOPS enrollment;
- qxctl-owned installation grant administration with expected-state CAS and forward recovery;
- freezing-path operation only.
