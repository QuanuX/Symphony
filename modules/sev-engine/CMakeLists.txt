cmake_minimum_required(VERSION 3.25)
project(SymphonySevEngine VERSION 0.1.0 LANGUAGES CXX)
include(CTest)
include(GNUInstallDirs)
if(NOT CMAKE_INSTALL_LIBEXECDIR STREQUAL "libexec" OR NOT CMAKE_INSTALL_DATADIR STREQUAL "share")
    message(FATAL_ERROR "sev-engine requires the exact libexec/share receipt layout used by qxctl")
endif()
option(SYMPHONY_KVE_USE_INSTALLED "Use an installed Symphony knowledge-vector foundation" OFF)
get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")
if(SYMPHONY_KVE_USE_INSTALLED)
    find_package(SymphonyKnowledgeVectorEngine 0.1 REQUIRED CONFIG)
elseif(NOT TARGET Symphony::KnowledgeVectorEngine)
    set(SYMPHONY_KVE_BUILD_TESTING OFF CACHE BOOL "Build embedded foundation tests" FORCE)
    add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp"
        "${CMAKE_CURRENT_BINARY_DIR}/knowledge-vector-engine-cpp" EXCLUDE_FROM_ALL)
endif()
add_executable(symphony-sev src/main.cpp src/sev.cpp)
target_include_directories(symphony-sev PRIVATE src)
target_link_libraries(symphony-sev PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(symphony-sev PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
target_compile_options(symphony-sev PRIVATE $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
if(BUILD_TESTING)
    add_executable(sev-engine-tests tests/sev_test.cpp src/sev.cpp)
    target_include_directories(sev-engine-tests PRIVATE src)
    target_link_libraries(sev-engine-tests PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(sev-engine-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sev-engine-tests PRIVATE $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sev-engine-tests COMMAND sev-engine-tests)
    add_test(NAME sev-engine-process-smoke COMMAND ${CMAKE_CURRENT_SOURCE_DIR}/tests/process_smoke.sh $<TARGET_FILE:symphony-sev>)
endif()
set(SYMPHONY_SEV_LIBEXEC_DIR "${CMAKE_INSTALL_LIBEXECDIR}/symphony/sev-engine/0.1.0-dev")
set(SYMPHONY_SEV_DATA_DIR "${CMAKE_INSTALL_DATADIR}/doc/symphony/sev-engine/0.1.0-dev")
set(SYMPHONY_SEV_LICENSE_DIR "${CMAKE_INSTALL_DATADIR}/licenses/symphony-sev-engine/0.1.0-dev")
configure_file(cmake/uninstall.cmake.in ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake @ONLY)
symphony_install_receipt_v2_preflight(RECEIPT_PATH share/symphony/receipts/sev-engine/0.1.0-dev/install-receipt.json)
install(TARGETS symphony-sev RUNTIME DESTINATION ${SYMPHONY_SEV_LIBEXEC_DIR})
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md DESTINATION ${SYMPHONY_SEV_DATA_DIR})
install(FILES ../../LICENSE DESTINATION ${SYMPHONY_SEV_LICENSE_DIR} RENAME LICENSE-AGPL-3.0)
install(FILES ../../libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT DESTINATION ${SYMPHONY_SEV_LICENSE_DIR} RENAME nlohmann-json-LICENSE.MIT)
symphony_install_receipt_v2(
    COMPONENT_ID sev-engine COMPONENT_KIND vector_engine MODULE_ID sev-engine VECTOR_ID sev
    ENGINE_ID symphony-sev PACKAGE_ID sev-engine VERSION 0.1.0-dev
    RECEIPT_PATH share/symphony/receipts/sev-engine/0.1.0-dev/install-receipt.json
    OWNED_FILES
        "libexec/symphony/sev-engine/0.1.0-dev/symphony-sev|executable"
        "share/doc/symphony/sev-engine/0.1.0-dev/INSTALL.md|regular"
        "share/doc/symphony/sev-engine/0.1.0-dev/INTENT.md|regular"
        "share/doc/symphony/sev-engine/0.1.0-dev/MANIFEST.md|regular"
        "share/doc/symphony/sev-engine/0.1.0-dev/SKILL.md|regular"
        "share/doc/symphony/sev-engine/0.1.0-dev/SPEC.md|regular"
        "share/licenses/symphony-sev-engine/0.1.0-dev/LICENSE-AGPL-3.0|regular"
        "share/licenses/symphony-sev-engine/0.1.0-dev/nlohmann-json-LICENSE.MIT|regular"
    ENTRY_POINTS "symphony-sev|executable|libexec/symphony/sev-engine/0.1.0-dev/symphony-sev|symphony.knowledge.engine-process.v1"
    COMPATIBLE_RECEPTORS symphony.maestro.knowledge-engine.v1)
add_custom_target(uninstall-sev-engine COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
    -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake COMMENT "Remove only sev-engine receipt-owned files")
