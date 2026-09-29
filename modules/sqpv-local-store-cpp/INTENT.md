# SQPV Local Store C++ Intent

Provide one selectable native C++26 retention library for an exact SQMV metadata
manifest and an exact SQFV batch stream. Callers can commit contiguous immutable
batches, obtain bounded commit evidence, retry an uncertain result, and read
retained batches back into their chosen SQFV context.

This first module selects local APFS on macOS, private caller-owned storage, one
exclusive writer handle and finite caller-supplied limits. Those are this module's
operating contract; other SQPV storage implementations remain possible. No
background service, provider collection, qxctl surface, data deletion, replication,
transformation, destination acknowledgement or universal exactly-once delivery is
created here.
