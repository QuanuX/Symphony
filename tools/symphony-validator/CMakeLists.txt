cmake_minimum_required(VERSION 3.10)
project(symphony-validator CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED True)

add_executable(symphony-validator
    src/main.cpp
    src/cli.cpp
    src/evidence.cpp
    src/paths.cpp
    src/skvi_index.cpp
    src/sclv_changelog.cpp
    src/cross_reference.cpp
    src/vocabulary.cpp
)

target_include_directories(symphony-validator PRIVATE src)
