# Optional SOV SSH carrier intent

Carry one explicit, authenticated controller-to-target SOV invocation through a
caller-selected SSH client and agent-held identity. SOV owns remote meaning;
qxctl owns complete selection and presentation; native SBV owns all calculation,
job-state, cancellation and result semantics. No bus or transport is universal.

See the exact [carrier contract](../../knowledge/sov/SSH-CARRIER.md). This
independent executable does not provision a host, manage credentials, start a
service, implement scientific work or infer remote termination from disconnection.
