# Third-Party Source Record

## nlohmann/json

- upstream: `https://github.com/nlohmann/json`
- release: `v3.12.0`
- release date: `2025-04-11`
- asset: `json.hpp`
- asset URL: `https://github.com/nlohmann/json/releases/download/v3.12.0/json.hpp`
- verified SHA-256: `aaf127c04cb31c406e5b04a63f1ae89369fccde6d8fa7cdda1ed4f32dfc5de63`
- license: MIT, copied as `nlohmann/LICENSE.MIT`
- use: bounded JSON parsing and deterministic compact serialization inside the shared C++ foundation
- linkage: header-only source compiled into static consumers; no runtime shared-library or download dependency

The checked-in header was obtained from the official release asset and matched the checksum published with that release. The dependency is isolated from `tools/symphony-validator/` and may not be discovered or updated at runtime.
