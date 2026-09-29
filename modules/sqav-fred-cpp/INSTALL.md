# Installing sqav-fred-cpp

Exact release `0.1.0-dev`; CMake package `SymphonySqavFred` version `0.1.0` and public target `Symphony::SqavFred`. CMake >=3.30, C++26, single-configuration generator. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev.

Configure from this module or set `SYMPHONY_SQAVFRED_INSTALLED=ON` with an explicit CMake prefix for exact installed dependencies. Source dependency builds exclude their install rules.

```sh
cmake -S modules/sqav-fred-cpp -B /tmp/sqav-fred-cpp-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/sqav-fred-cpp-build
ctest --test-dir /tmp/sqav-fred-cpp-build --output-on-failure
cmake --install /tmp/sqav-fred-cpp-build --prefix /tmp/sqav-fred-cpp-prefix
cmake -DINSTALL_PREFIX=/tmp/sqav-fred-cpp-prefix -P /tmp/sqav-fred-cpp-build/uninstall.cmake
```

Receipt-v2 owns only this version’s archive, header(s), exact CMake exports, six owner documents and AGPL license. Guarded removal checks exact receipt/digest/path ownership. Installation starts no service or provider operation. Operational credentials, provider host applications and rights remain separate.
