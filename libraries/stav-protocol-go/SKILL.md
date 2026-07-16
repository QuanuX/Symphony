# STAV Protocol Kernel Skill

## Agent Rules

Use the kernel only to implement the canonical contracts under `knowledge/stav/`. Never infer protocol truth from Go types when the knowledge vector differs.

Stop before adding listeners, filesystem state, authorization, producer enrollment, operational receipt emission, or ledger behavior. Configuration/status/local-envelope content is implemented here only as canonical authority-free data mechanics.

Any parser or serializer change must run the complete canonical and invalid fixture corpus and preserve all recorded digest vectors.
