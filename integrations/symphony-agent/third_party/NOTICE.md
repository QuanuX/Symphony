# Third-party source

nlohmann/json v3.12.0, header-only, MIT. Upstream: https://github.com/nlohmann/json/releases/tag/v3.12.0.

`nlohmann/json.hpp` SHA-256: `aaf127c04cb31c406e5b04a63f1ae89369fccde6d8fa7cdda1ed4f32dfc5de63`.

Copied without modification from Symphony's already-vendored source at commit `53d6a5a3bbdfac209b5885a690a8847f0144759e`; its MIT notice is in `nlohmann/LICENSE.MIT`. Builds do not download dependencies. SHA-256 uses CommonCrypto on macOS or OpenSSL 3 Crypto on Linux. OpenSSL is a system build dependency and is not bundled.

The adapter source is distributed under the included GNU AGPL v3 license, consistent with the existing Symphony project. This does not relabel third-party code.
