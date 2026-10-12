# SBV Go adapter usage

Read README.md and IMPLEMENTATION.md. Pin the qxctl executable hash and supported
native release, read exact schema/template resources, and supply caller-owned
requests. Preserve result statuses and source identities. Treat callback values
as provisional until completion plus a nil error; retain copies if needed.
Inspect possible write destinations after failure before a caller-selected retry.
External model/proposal declarations do not establish calibration or causality.
This library grants no live/provider/publication authority.
