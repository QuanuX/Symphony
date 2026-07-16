# STAV Protocol Kernel Skill

## Agent Rules

Use the kernel only to implement the canonical contracts under `knowledge/stav/`. Never infer protocol truth from Go types when the knowledge vector differs.

Stop before adding network listeners, filesystem state, authorization, producer enrollment, configuration/status message content, committed-receipt emission, or ledger behavior. Those remain owner gates.

Any parser or serializer change must run the complete canonical and invalid fixture corpus and preserve all recorded digest vectors.
