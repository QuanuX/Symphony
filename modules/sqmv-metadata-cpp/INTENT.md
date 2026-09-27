# SQMV Metadata C++ Intent

Provide independently usable C++26 immutable metadata manifests whose exact
references can bind a prepared SQFV batch to its caller-selected description.
Dataset lineage, revision, schema version, representation version, access scope,
and producer-attributed evidence remain distinguishable. Resolve the manifest
before movement, then retain its immutable value without a resident catalogue
or synchronous network lookup.

This first profile implements bounded structural and reference integrity. The
caller remains responsible for interpreting the selected schema, representation,
and evidence under their actual owner contracts. The module does not acquire
provider data, authenticate assertions, grant rights, infer time or completeness,
transform payloads, retain durable state, or publish a catalogue.
