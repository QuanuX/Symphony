# SSIAG macOS Keychain Provider Intent

## Purpose

Provide the optional macOS Apple Keychain boundary for Symphony Secure Identity and Access Governance as an independently built, installed, upgraded, and removed Swift executable.

## Process Boundary

This adapter is never linked into the Go SSIAG foundation. The foundation invokes it through a protected, versioned local IPC contract. Apple frameworks and native provider behavior remain entirely inside this process.

## Current Scaffold

The Phase 9 executable implements metadata discovery over one fail-closed request/response process only. It does not import the Security framework, read or write Keychain items, accept credential material, open the synthetic one-shot descriptor, or advertise operational access.

The operational architecture is ratified as per-user and session-aware, with mutual executable trust, non-exportable operations preferred, and a separate one-shot protected channel for any explicitly exportable bytes. The current canonical channel descriptor is synthetic and cannot deliver bytes. Exact item, signing, entitlement, interaction, provisioning, memory, and operational channel details remain later implementation gates.

## Non-Scope

The adapter is not qxctl, a general secret store, a plaintext fallback, a daemon, a network service, STAV, or an authority for SSIAG protocol truth.
