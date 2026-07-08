cmake_minimum_required(VERSION 3.25)
project(symphony-validator CXX)

add_executable(symphony-validator
    src/main.cpp
    src/cli.cpp
    src/evidence.cpp
    src/paths.cpp
    src/skvi_index.cpp
    src/skvi_coverage.cpp
    src/skvi_paths.cpp
    src/sclv_changelog.cpp
    src/sclv_shape.cpp
    src/sclv_ledger.cpp
    src/sclv_references.cpp
    src/cross_reference.cpp
    src/vocabulary.cpp
    src/artifacts.cpp
    src/canonical_surfaces.cpp
    src/validator_contracts.cpp
    src/runtime_contracts.cpp
    src/knowledge_contracts.cpp
    src/root_contracts.cpp
    src/doctrine_vocab.cpp
)

target_include_directories(symphony-validator PRIVATE src)

set_target_properties(symphony-validator PROPERTIES
  CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON
  CXX_EXTENSIONS OFF
)
