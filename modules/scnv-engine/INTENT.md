# SCNV engine intent

`scnv-engine` implements Symphony Consolidated Naming Vector's supplied, attributed Node-family name records. Each recorded name is usable as a selector within its explicit scope, effective interval and evidence snapshot. Users supply names and subjects; the engine preserves their spelling and returns ambiguity or missing evidence explicitly.

The C++26 library and bounded process are independently buildable and installable. No parent SNV installation, registry service, provider access or secret is required to validate supplied records or resolve a supplied snapshot.
