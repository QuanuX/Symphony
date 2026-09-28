# Installing sqav-news-api-cpp

Exact release `0.1.0-dev`; CMake package `SymphonySqavNewsApi` version `0.1.0` and public target `Symphony::SqavNewsApi`. CMake >=3.30, C++26, single-configuration generator. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev.

Configure from this module or set `SYMPHONY_SQAVNEWSAPI_INSTALLED=ON` with an explicit CMake prefix for exact installed dependencies. Source dependency builds exclude their install rules.

```sh
cmake -S modules/sqav-news-api-cpp -B /tmp/sqav-news-api-cpp-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/sqav-news-api-cpp-build
ctest --test-dir /tmp/sqav-news-api-cpp-build --output-on-failure
cmake --install /tmp/sqav-news-api-cpp-build --prefix /tmp/sqav-news-api-cpp-prefix
cmake -DINSTALL_PREFIX=/tmp/sqav-news-api-cpp-prefix -P /tmp/sqav-news-api-cpp-build/uninstall.cmake
```

Receipt-v2 owns only this version’s archive, header(s), exact CMake exports, six owner documents and AGPL license. Guarded removal checks exact receipt/digest/path ownership. Installation starts no service or provider operation. Operational credentials, provider host applications and rights remain separate.
