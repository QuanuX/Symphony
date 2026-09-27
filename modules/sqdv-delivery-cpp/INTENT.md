# SQDV Delivery C++ Intent

Deliver full immutable prepared batches to a selected trusted same-process
recipient under an exact view identity, with separate transfer, processing,
retention, and payload-release evidence. Provide disposable delivery and
retention-before-delivery as explicitly selected profiles.

SQFV continues to own actual movement, payload leases, and byte credits. SQMV
supplies the exact metadata binding. SQPV supplies actual retained commits and
reads. SQDV owns recipient identity, contiguous processing acknowledgement,
bounded outstanding processing obligations, and caller-persisted resume values.
The composition requires no service, universal queue, provider, converter,
background retention worker, external destination, or qxctl payload path.
