cmake_minimum_required(VERSION 3.25)
project(symphony-validator CXX)

include(CTest)

add_executable(symphony-validator
    src/main.cpp
    src/cli.cpp
    src/evidence.cpp
    src/paths.cpp
    src/skvi_index.cpp
    src/skvi_coverage.cpp
    src/skvi_paths.cpp
    src/sacv_registry.cpp
    src/sclv_changelog.cpp
    src/sclv_shape.cpp
    src/sclv_ledger.cpp
    src/sclv_references.cpp
    src/sclv_skvi_references.cpp
    src/cross_reference.cpp
    src/vocabulary.cpp
    src/artifacts.cpp
    src/canonical_surfaces.cpp
    src/validator_contracts.cpp
    src/runtime_contracts.cpp
    src/knowledge_contracts.cpp
    src/root_contracts.cpp
    src/doctrine_vocab.cpp
    src/caller_authority.cpp
    src/validator_build.cpp
)

target_include_directories(symphony-validator PRIVATE src)

set_target_properties(symphony-validator PROPERTIES
  CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON
  CXX_EXTENSIONS OFF
)

add_executable(sclv-temporal-tests
    tests/sclv_temporal_test.cpp
    src/evidence.cpp
    src/sclv_changelog.cpp
    src/sclv_ledger.cpp
    src/sclv_shape.cpp
)

target_include_directories(sclv-temporal-tests PRIVATE src)

set_target_properties(sclv-temporal-tests PROPERTIES
  CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON
  CXX_EXTENSIONS OFF
)

add_executable(caller-authority-tests
    tests/caller_authority_test.cpp
    src/caller_authority.cpp
    src/evidence.cpp
)

target_include_directories(caller-authority-tests PRIVATE src)

target_compile_definitions(caller-authority-tests PRIVATE
  SYMPHONY_VALIDATOR_TESTING=1
)

set_target_properties(caller-authority-tests PROPERTIES
  CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON
  CXX_EXTENSIONS OFF
)

if(BUILD_TESTING)
  add_test(NAME sclv-temporal-tests COMMAND sclv-temporal-tests)
  add_test(NAME caller-authority-tests COMMAND caller-authority-tests)
endif()
