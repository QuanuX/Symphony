cmake_minimum_required(VERSION 3.25)
project(symphony-validator CXX)

include(CTest)
include(GNUInstallDirs)

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")

if(NOT CMAKE_INSTALL_LIBEXECDIR STREQUAL "libexec" OR
   NOT CMAKE_INSTALL_DATADIR STREQUAL "share")
  message(FATAL_ERROR
    "symphony-validator 0.1.0-dev requires the exact libexec/share receipt layout used by qxctl")
endif()

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
if(NOT TARGET Symphony::KnowledgeVectorEngine)
  set(SYMPHONY_KVE_BUILD_TESTING OFF CACHE BOOL "Build embedded foundation tests" FORCE)
  add_subdirectory(
    "${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp"
    "${CMAKE_CURRENT_BINARY_DIR}/knowledge-vector-engine-cpp"
    EXCLUDE_FROM_ALL
  )
endif()

add_executable(symphony-validator
    src/main.cpp
    src/cli.cpp
    src/evidence.cpp
    src/paths.cpp
    src/skvi_index.cpp
    src/skvi_coverage.cpp
    src/skvi_paths.cpp
    src/sacv_registry.cpp
    src/sodv_releases.cpp
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
    src/projector.cpp
    src/feature_administration.cpp
    src/root_summary.cpp
)

target_include_directories(symphony-validator PRIVATE src)
target_link_libraries(symphony-validator PRIVATE Symphony::KnowledgeVectorEngine)

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

add_executable(sclv-cross-reference-tests
    tests/sclv_cross_reference_test.cpp
    src/cross_reference.cpp
    src/sclv_references.cpp
    src/sclv_skvi_references.cpp
    src/evidence.cpp
)

target_include_directories(sclv-cross-reference-tests PRIVATE src)

set_target_properties(sclv-cross-reference-tests PROPERTIES
  CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON
  CXX_EXTENSIONS OFF
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

add_executable(sodv-release-tests
    tests/sodv_release_test.cpp
    src/sodv_releases.cpp
    src/evidence.cpp
)

add_executable(feature-administration-tests
    tests/feature_administration_test.cpp
    src/feature_administration.cpp
    src/evidence.cpp
)

add_executable(root-summary-tests
    tests/root_summary_test.cpp
    src/root_summary.cpp
    src/feature_administration.cpp
    src/sodv_releases.cpp
    src/evidence.cpp
)

target_include_directories(root-summary-tests PRIVATE src)
target_link_libraries(root-summary-tests PRIVATE Symphony::KnowledgeVectorEngine)

set_target_properties(root-summary-tests PROPERTIES
  CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON
  CXX_EXTENSIONS OFF
)

target_include_directories(feature-administration-tests PRIVATE src)
target_link_libraries(feature-administration-tests PRIVATE Symphony::KnowledgeVectorEngine)

set_target_properties(feature-administration-tests PROPERTIES
  CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON
  CXX_EXTENSIONS OFF
)

target_include_directories(sodv-release-tests PRIVATE src)

set_target_properties(sodv-release-tests PROPERTIES
  CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON
  CXX_EXTENSIONS OFF
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
  add_test(NAME sclv-cross-reference-tests COMMAND sclv-cross-reference-tests)
  add_test(NAME caller-authority-tests COMMAND caller-authority-tests)
  get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
  add_test(NAME sodv-release-tests COMMAND sodv-release-tests ${SYMPHONY_REPOSITORY_ROOT})
  add_test(NAME feature-administration-tests COMMAND feature-administration-tests ${SYMPHONY_REPOSITORY_ROOT})
  add_test(NAME root-summary-tests COMMAND root-summary-tests ${SYMPHONY_REPOSITORY_ROOT})
endif()

set(SYMPHONY_VALIDATOR_LIBEXEC_DIR "${CMAKE_INSTALL_LIBEXECDIR}/symphony/symphony-validator/0.1.0-dev")
set(SYMPHONY_VALIDATOR_DATA_DIR "${CMAKE_INSTALL_DATADIR}/doc/symphony/symphony-validator/0.1.0-dev")
set(SYMPHONY_VALIDATOR_LICENSE_DIR "${CMAKE_INSTALL_DATADIR}/licenses/symphony-validator/0.1.0-dev")

configure_file(cmake/uninstall.cmake.in ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake @ONLY)

symphony_install_receipt_v2_preflight(
  RECEIPT_PATH share/symphony/receipts/symphony-validator/0.1.0-dev/install-receipt.json)

install(TARGETS symphony-validator RUNTIME DESTINATION ${SYMPHONY_VALIDATOR_LIBEXEC_DIR})
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md DESTINATION ${SYMPHONY_VALIDATOR_DATA_DIR})
install(FILES ../../LICENSE DESTINATION ${SYMPHONY_VALIDATOR_LICENSE_DIR} RENAME LICENSE-AGPL-3.0)
install(FILES ../../libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT
  DESTINATION ${SYMPHONY_VALIDATOR_LICENSE_DIR} RENAME nlohmann-json-LICENSE.MIT)

symphony_install_receipt_v2(
  COMPONENT_ID symphony-validator COMPONENT_KIND module MODULE_ID symphony-validator
  ENGINE_ID symphony-validator PACKAGE_ID symphony-validator VERSION 0.1.0-dev
  RECEIPT_PATH share/symphony/receipts/symphony-validator/0.1.0-dev/install-receipt.json
  OWNED_FILES
    "libexec/symphony/symphony-validator/0.1.0-dev/symphony-validator|executable"
    "share/doc/symphony/symphony-validator/0.1.0-dev/INSTALL.md|regular"
    "share/doc/symphony/symphony-validator/0.1.0-dev/INTENT.md|regular"
    "share/doc/symphony/symphony-validator/0.1.0-dev/MANIFEST.md|regular"
    "share/doc/symphony/symphony-validator/0.1.0-dev/SKILL.md|regular"
    "share/doc/symphony/symphony-validator/0.1.0-dev/SPEC.md|regular"
    "share/licenses/symphony-validator/0.1.0-dev/LICENSE-AGPL-3.0|regular"
    "share/licenses/symphony-validator/0.1.0-dev/nlohmann-json-LICENSE.MIT|regular"
  ENTRY_POINTS "symphony-validator|executable|libexec/symphony/symphony-validator/0.1.0-dev/symphony-validator|symphony.validation.result.v1")

add_custom_target(uninstall-symphony-validator
  COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
    -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
  COMMENT "Remove only symphony-validator receipt-owned files")
