# Installing scabv-ibkr-tws-cpp

Exact release `0.1.0-dev`; CMake package `SymphonyScabvIbkrTws` version `0.1.0` and public target `Symphony::ScabvIbkrTws`. CMake >=3.30, C++26, single-configuration generator. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev.

Configure from this module or set `SYMPHONY_SCABVIBKRTWS_INSTALLED=ON` with an explicit CMake prefix for exact installed dependencies. Source dependency builds exclude their install rules.

```sh
cmake -S modules/scabv-ibkr-tws-cpp -B /tmp/scabv-ibkr-tws-cpp-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/scabv-ibkr-tws-cpp-build
ctest --test-dir /tmp/scabv-ibkr-tws-cpp-build --output-on-failure
cmake --install /tmp/scabv-ibkr-tws-cpp-build --prefix /tmp/scabv-ibkr-tws-cpp-prefix
cmake -DINSTALL_PREFIX=/tmp/scabv-ibkr-tws-cpp-prefix -P /tmp/scabv-ibkr-tws-cpp-build/uninstall.cmake
```

Receipt-v2 owns only this version’s archive, header(s), exact CMake exports, six owner documents and AGPL license. Guarded removal checks exact receipt/digest/path ownership. Installation starts no service or provider operation. Operational credentials, provider host applications and rights remain separate.
