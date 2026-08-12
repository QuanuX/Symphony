cmake_minimum_required(VERSION 3.25)
project(SymphonySodvEngine VERSION 0.1.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)

if(NOT CMAKE_INSTALL_LIBEXECDIR STREQUAL "libexec" OR
   NOT CMAKE_INSTALL_DATADIR STREQUAL "share")
    message(FATAL_ERROR
        "sodv-engine 0.1.0-dev requires the exact libexec/share receipt layout used by qxctl")
endif()

option(SYMPHONY_KVE_USE_INSTALLED "Use an installed Symphony knowledge-vector foundation" OFF)
get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")
if(SYMPHONY_KVE_USE_INSTALLED)
    find_package(SymphonyKnowledgeVectorEngine 0.1 REQUIRED CONFIG)
elseif(NOT TARGET Symphony::KnowledgeVectorEngine)
    set(SYMPHONY_KVE_BUILD_TESTING OFF CACHE BOOL "Build embedded foundation tests" FORCE)
    add_subdirectory(
        "${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp"
        "${CMAKE_CURRENT_BINARY_DIR}/knowledge-vector-engine-cpp"
        EXCLUDE_FROM_ALL
    )
endif()

add_executable(symphony-sodv src/main.cpp src/sodv.cpp)
target_include_directories(symphony-sodv PRIVATE src)
target_link_libraries(symphony-sodv PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(symphony-sodv PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
target_compile_options(symphony-sodv PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

if(BUILD_TESTING)
    add_executable(sodv-engine-tests tests/sodv_test.cpp src/sodv.cpp)
    target_include_directories(sodv-engine-tests PRIVATE src)
    target_link_libraries(sodv-engine-tests PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(sodv-engine-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sodv-engine-tests PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sodv-engine-tests COMMAND sodv-engine-tests ${SYMPHONY_REPOSITORY_ROOT})
    add_test(NAME sodv-engine-process-smoke
        COMMAND ${CMAKE_CURRENT_SOURCE_DIR}/tests/process_smoke.sh
                $<TARGET_FILE:symphony-sodv> ${SYMPHONY_REPOSITORY_ROOT})
endif()

set(SYMPHONY_SODV_LIBEXEC_DIR "${CMAKE_INSTALL_LIBEXECDIR}/symphony/sodv-engine/0.1.0-dev")
set(SYMPHONY_SODV_DATA_DIR "${CMAKE_INSTALL_DATADIR}/doc/symphony/sodv-engine/0.1.0-dev")
set(SYMPHONY_SODV_LICENSE_DIR "${CMAKE_INSTALL_DATADIR}/licenses/symphony-sodv-engine/0.1.0-dev")

configure_file(cmake/uninstall.cmake.in ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake @ONLY)

symphony_install_receipt_v2_preflight(
    RECEIPT_PATH share/symphony/receipts/sodv-engine/0.1.0-dev/install-receipt.json)

install(TARGETS symphony-sodv RUNTIME DESTINATION ${SYMPHONY_SODV_LIBEXEC_DIR})
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md DESTINATION ${SYMPHONY_SODV_DATA_DIR})
install(FILES ../../LICENSE DESTINATION ${SYMPHONY_SODV_LICENSE_DIR} RENAME LICENSE-AGPL-3.0)
install(FILES ../../libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT
    DESTINATION ${SYMPHONY_SODV_LICENSE_DIR} RENAME nlohmann-json-LICENSE.MIT)

symphony_install_receipt_v2(
    COMPONENT_ID sodv-engine COMPONENT_KIND vector_engine MODULE_ID sodv-engine
    VECTOR_ID sodv ENGINE_ID symphony-sodv PACKAGE_ID sodv-engine VERSION 0.1.0-dev
    RECEIPT_PATH share/symphony/receipts/sodv-engine/0.1.0-dev/install-receipt.json
    OWNED_FILES
        "libexec/symphony/sodv-engine/0.1.0-dev/symphony-sodv|executable"
        "share/doc/symphony/sodv-engine/0.1.0-dev/INSTALL.md|regular"
        "share/doc/symphony/sodv-engine/0.1.0-dev/INTENT.md|regular"
        "share/doc/symphony/sodv-engine/0.1.0-dev/MANIFEST.md|regular"
        "share/doc/symphony/sodv-engine/0.1.0-dev/SKILL.md|regular"
        "share/doc/symphony/sodv-engine/0.1.0-dev/SPEC.md|regular"
        "share/licenses/symphony-sodv-engine/0.1.0-dev/LICENSE-AGPL-3.0|regular"
        "share/licenses/symphony-sodv-engine/0.1.0-dev/nlohmann-json-LICENSE.MIT|regular"
    ENTRY_POINTS "symphony-sodv|executable|libexec/symphony/sodv-engine/0.1.0-dev/symphony-sodv|symphony.knowledge.engine-process.v1"
    COMPATIBLE_RECEPTORS symphony.maestro.knowledge-engine.v1)

add_custom_target(uninstall-sodv-engine
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
            -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
    COMMENT "Remove only sodv-engine receipt-owned files")
