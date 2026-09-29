# Databento public test fixtures

`../public_fixture.hpp` embeds two public fixtures from
[databento/databento-cpp v0.68.0](https://github.com/databento/databento-cpp/tree/v0.68.0/tests/data).
The upstream fixture bytes are unchanged; this repository represents them as
hexadecimal strings inside a Symphony C++ test helper.

| Function | Upstream file | Bytes | Upstream Git blob |
| --- | --- | ---: | --- |
| `fixture()` | `tests/data/test_data.mbo.v3.dbn` | 472 | `de0f31d1073e5259a29b12da7a1a9e001263dbb4` |
| `fixture_v1()` | `tests/data/test_data.mbo.v1.dbn` | 318 | `476dd0fea15faa7513eadb472c77718315be6e00` |

The upstream project distributes these files under the Apache License 2.0.
[LICENSE.Apache-2.0](LICENSE.Apache-2.0) is an unchanged copy of its
[v0.68.0 LICENSE](https://github.com/databento/databento-cpp/blob/v0.68.0/LICENSE),
Git blob `91e18a62b67551a4d427e2eaee0d33dcab94e141`.
The upstream tree at that version contains no separate NOTICE file.

These are test-only upstream materials, attributed to the Databento project
and its contributors. They are not installed with the native library. The
root Symphony license continues to describe Symphony's own source. The paid
ES/AAPL samples collected during local development are separate private
verification inputs and are not included in this source distribution.
