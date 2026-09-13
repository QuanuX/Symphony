# Independent installation

Configure this module with CMake3.25+ and a C++26 compiler. It links the repository foundation by default; SYMPHONY_KVE_USE_INSTALLED uses the independently installed compatible foundation. No network dependency acquisition occurs. Build symphony-shv and install into an explicit prefix. Exact layout is libexec/symphony/shv-engine/0.1.0-dev, companion docs and receipt-owned shv.schema.json / shv.templates.json.

The install receipt is written last. Existing differing exact installations are not overwritten. uninstall-shv-engine uses the receipt-v2 guarded remover and touches only verified owned files; it does not remove caller catalogue/source data. Installing a new engine does not select it or activate a daemon. Platform support is the compiled receipt's actual OS/architecture, not an untested distribution promise.
