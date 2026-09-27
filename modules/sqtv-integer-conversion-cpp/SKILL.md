# SQTV Integer Conversion C++ Skill

Read the SQTV Quad, this SPEC and the SQFV/SQMV contracts before integration.

1. Establish that the input really is a dense non-null integer array under the
   exact schema and layout declared here. Metadata assertions do not parse or
   validate a provider schema.
2. Supply the actual immutable SQFV batch and its exact SQMV manifest.
3. Choose output representation, finite per-call limits and independent output
   transfer position. Preserve input evidence and permitted access scope.
4. Handle overflow and every other non-success status without publishing output.
5. Keep Result alive while borrowing batch, manifest or reference views; retain
   owner handles when independent lifetime is needed.
6. Compose retention and delivery through their actual owner contracts.

No provider access, information-loss allowance, cache or background work is
inferred from calling or installing this library.
