# SSIAG macOS Keychain Provider Intent

## Purpose

Provide the optional macOS Apple Keychain boundary for Symphony Secure Identity and Access Governance as an independently built, installed, upgraded, and removed Swift executable.

## Process Boundary

This adapter is never linked into the Go SSIAG foundation. The foundation invokes it through a protected, versioned local IPC contract. Apple frameworks and native provider behavior remain entirely inside this process.

## Current Readiness Foundation

The executable preserves the exact Phase 9 metadata protocol and adds a separate Phase 10B readiness operation. The Apple Security framework is used only to validate the complete code-signature structure, compile and evaluate a receipt-owned native code requirement, and observe bounded security-session capability flags. It does not read or write Keychain items, accept credential material, request broad certificate/profile/entitlement payloads, open the synthetic one-shot descriptor, or advertise operational access.

Production packaging is a complete Developer ID-signed, hardened, securely timestamped, notarized app-like bundle with exact receipt-v2 ownership. The Go foundation reconstructs every owned file in private staging before invocation. Structural validity, native protected-policy match, and operational eligibility are distinct. Eligibility and every operation flag remain disabled. The future operational architecture targets the per-user data-protection Keychain, prefers non-exportable operations, and reserves a separate one-shot protected channel for any explicitly exportable bytes. Exact item, interaction, memory, and operational-channel details remain later gates.

## Non-Scope

The adapter is not qxctl, a general secret store, a plaintext fallback, a daemon, a network service, STAV, or an authority for SSIAG protocol truth.
