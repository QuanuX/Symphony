# Installing sqav-databento-reference-cpp

Exact release `0.1.0-dev`; CMake package `SymphonySqavDatabentoReference` version `0.1.0` and public target `Symphony::SqavDatabentoReference`. CMake >=3.30, C++26, single-configuration generator. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev, sqav-databento-dbn-cpp 0.5.0-dev, zstd 1.5.7 static library.

Configure from this module or set `SYMPHONY_SQAVDATABENTOREFERENCE_INSTALLED=ON` with an explicit CMake prefix for exact installed dependencies. Source dependency builds exclude their install rules.

```sh
cmake -S modules/sqav-databento-reference-cpp -B /tmp/sqav-databento-reference-cpp-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/sqav-databento-reference-cpp-build
ctest --test-dir /tmp/sqav-databento-reference-cpp-build --output-on-failure
cmake --install /tmp/sqav-databento-reference-cpp-build --prefix /tmp/sqav-databento-reference-cpp-prefix
cmake -DINSTALL_PREFIX=/tmp/sqav-databento-reference-cpp-prefix -P /tmp/sqav-databento-reference-cpp-build/uninstall.cmake
```

Receipt-v2 owns only this version’s archive, header(s), exact CMake exports, six owner documents and AGPL license. Guarded removal checks exact receipt/digest/path ownership. Installation starts no service or provider operation. Operational credentials, provider host applications and rights remain separate.
