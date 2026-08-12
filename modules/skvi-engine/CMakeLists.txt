cmake_minimum_required(VERSION 3.25)
project(SymphonySkviEngine VERSION 0.1.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)

if(NOT CMAKE_INSTALL_LIBEXECDIR STREQUAL "libexec" OR
   NOT CMAKE_INSTALL_DATADIR STREQUAL "share")
    message(FATAL_ERROR
        "skvi-engine 0.1.0-dev requires the exact libexec/share receipt layout used by qxctl")
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

add_executable(symphony-skvi src/main.cpp src/skvi.cpp)
target_include_directories(symphony-skvi PRIVATE src)
target_link_libraries(symphony-skvi PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(symphony-skvi PROPERTIES
    CXX_STANDARD 26
    CXX_STANDARD_REQUIRED ON
    CXX_EXTENSIONS OFF
)
target_compile_options(symphony-skvi PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
)

if(BUILD_TESTING)
    add_executable(skvi-engine-tests tests/skvi_test.cpp src/skvi.cpp)
    target_include_directories(skvi-engine-tests PRIVATE src)
    target_link_libraries(skvi-engine-tests PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(skvi-engine-tests PROPERTIES
        CXX_STANDARD 26
        CXX_STANDARD_REQUIRED ON
        CXX_EXTENSIONS OFF
    )
    target_compile_options(skvi-engine-tests PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
    )
    add_test(NAME skvi-engine-tests COMMAND skvi-engine-tests ${SYMPHONY_REPOSITORY_ROOT})
    add_test(
        NAME skvi-engine-process-smoke
        COMMAND ${CMAKE_CURRENT_SOURCE_DIR}/tests/process_smoke.sh
                $<TARGET_FILE:symphony-skvi>
                ${SYMPHONY_REPOSITORY_ROOT}
    )
endif()

set(SYMPHONY_SKVI_LIBEXEC_DIR
    "${CMAKE_INSTALL_LIBEXECDIR}/symphony/skvi-engine/0.1.0-dev")
set(SYMPHONY_SKVI_DATA_DIR
    "${CMAKE_INSTALL_DATADIR}/doc/symphony/skvi-engine/0.1.0-dev")
set(SYMPHONY_SKVI_LICENSE_DIR
    "${CMAKE_INSTALL_DATADIR}/licenses/symphony-skvi-engine/0.1.0-dev")

configure_file(cmake/uninstall.cmake.in ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake @ONLY)

symphony_install_receipt_v2_preflight(
    RECEIPT_PATH share/symphony/receipts/skvi-engine/0.1.0-dev/install-receipt.json)

install(TARGETS symphony-skvi RUNTIME DESTINATION ${SYMPHONY_SKVI_LIBEXEC_DIR})
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md DESTINATION ${SYMPHONY_SKVI_DATA_DIR})
install(FILES ../../LICENSE DESTINATION ${SYMPHONY_SKVI_LICENSE_DIR} RENAME LICENSE-AGPL-3.0)
install(FILES ../../libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT
    DESTINATION ${SYMPHONY_SKVI_LICENSE_DIR}
    RENAME nlohmann-json-LICENSE.MIT)

symphony_install_receipt_v2(
    COMPONENT_ID skvi-engine COMPONENT_KIND vector_engine
    MODULE_ID skvi-engine VECTOR_ID skvi ENGINE_ID symphony-skvi
    PACKAGE_ID skvi-engine VERSION 0.1.0-dev
    RECEIPT_PATH share/symphony/receipts/skvi-engine/0.1.0-dev/install-receipt.json
    OWNED_FILES
        "libexec/symphony/skvi-engine/0.1.0-dev/symphony-skvi|executable"
        "share/doc/symphony/skvi-engine/0.1.0-dev/INSTALL.md|regular"
        "share/doc/symphony/skvi-engine/0.1.0-dev/INTENT.md|regular"
        "share/doc/symphony/skvi-engine/0.1.0-dev/MANIFEST.md|regular"
        "share/doc/symphony/skvi-engine/0.1.0-dev/SKILL.md|regular"
        "share/doc/symphony/skvi-engine/0.1.0-dev/SPEC.md|regular"
        "share/licenses/symphony-skvi-engine/0.1.0-dev/LICENSE-AGPL-3.0|regular"
        "share/licenses/symphony-skvi-engine/0.1.0-dev/nlohmann-json-LICENSE.MIT|regular"
    ENTRY_POINTS
        "symphony-skvi|executable|libexec/symphony/skvi-engine/0.1.0-dev/symphony-skvi|symphony.knowledge.engine-process.v1"
    COMPATIBLE_RECEPTORS symphony.maestro.knowledge-engine.v1)

add_custom_target(uninstall-skvi-engine
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
            -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
    COMMENT "Remove only skvi-engine receipt-owned files"
)
