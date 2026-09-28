# Installing native-source-support-cpp

Exact release `0.1.0-dev`; CMake package `SymphonyNativeSourceSupport` version `0.1.0` and public target `Symphony::NativeSourceSupport`. CMake >=3.30, C++26, single-configuration generator. Dependencies: CURL 8.7.1 SDK/system library, knowledge-vector-engine-cpp 0.2.0-dev.

Configure from this module or set `SYMPHONY_NATIVESOURCESUPPORT_INSTALLED=ON` with an explicit CMake prefix for exact installed dependencies. Source dependency builds exclude their install rules.

```sh
cmake -S modules/native-source-support-cpp -B /tmp/native-source-support-cpp-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/native-source-support-cpp-build
ctest --test-dir /tmp/native-source-support-cpp-build --output-on-failure
cmake --install /tmp/native-source-support-cpp-build --prefix /tmp/native-source-support-cpp-prefix
cmake -DINSTALL_PREFIX=/tmp/native-source-support-cpp-prefix -P /tmp/native-source-support-cpp-build/uninstall.cmake
```

Receipt-v2 owns only this version’s archive, header(s), exact CMake exports, six owner documents and AGPL license. Guarded removal checks exact receipt/digest/path ownership. Installation starts no service or provider operation. Operational credentials, provider host applications and rights remain separate.
