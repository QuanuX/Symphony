cmake_minimum_required(VERSION 3.25)
project(SymphonySclvEngine VERSION 0.1.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)

if(NOT CMAKE_INSTALL_LIBEXECDIR STREQUAL "libexec" OR
   NOT CMAKE_INSTALL_DATADIR STREQUAL "share")
    message(FATAL_ERROR
        "sclv-engine 0.1.0-dev requires the exact libexec/share receipt layout used by qxctl")
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

function(symphony_sclv_target target)
    target_include_directories(${target} PRIVATE src)
    target_link_libraries(${target} PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(${target} PROPERTIES
        CXX_STANDARD 26
        CXX_STANDARD_REQUIRED ON
        CXX_EXTENSIONS OFF
    )
    target_compile_options(${target} PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
    )
endfunction()

add_executable(symphony-sclv src/main.cpp src/sclv.cpp src/provider.cpp)
symphony_sclv_target(symphony-sclv)

add_executable(symphony-sclv-evidence-local-git
    src/local_git_main.cpp src/local_git.cpp src/provider.cpp)
symphony_sclv_target(symphony-sclv-evidence-local-git)

add_executable(symphony-sclv-evidence-airgap
    src/airgap_main.cpp src/provider.cpp)
symphony_sclv_target(symphony-sclv-evidence-airgap)

if(BUILD_TESTING)
    add_executable(sclv-engine-tests tests/sclv_test.cpp src/sclv.cpp src/provider.cpp)
    symphony_sclv_target(sclv-engine-tests)
    add_test(NAME sclv-engine-tests COMMAND sclv-engine-tests ${SYMPHONY_REPOSITORY_ROOT})
    add_test(
        NAME sclv-engine-process-smoke
        COMMAND ${CMAKE_CURRENT_SOURCE_DIR}/tests/process_smoke.sh
                $<TARGET_FILE:symphony-sclv>
                $<TARGET_FILE:symphony-sclv-evidence-local-git>
                $<TARGET_FILE:symphony-sclv-evidence-airgap>
                ${SYMPHONY_REPOSITORY_ROOT}
    )
endif()

set(SYMPHONY_SCLV_LIBEXEC_DIR
    "${CMAKE_INSTALL_LIBEXECDIR}/symphony/sclv-engine/0.1.0-dev")
set(SYMPHONY_SCLV_DATA_DIR
    "${CMAKE_INSTALL_DATADIR}/doc/symphony/sclv-engine/0.1.0-dev")
set(SYMPHONY_SCLV_LICENSE_DIR
    "${CMAKE_INSTALL_DATADIR}/licenses/symphony-sclv-engine/0.1.0-dev")

configure_file(cmake/uninstall.cmake.in ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake @ONLY)

symphony_install_receipt_v2_preflight(
    RECEIPT_PATH share/symphony/receipts/sclv-engine/0.1.0-dev/install-receipt.json)

install(TARGETS
    symphony-sclv
    symphony-sclv-evidence-local-git
    symphony-sclv-evidence-airgap
    RUNTIME DESTINATION ${SYMPHONY_SCLV_LIBEXEC_DIR})
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md DESTINATION ${SYMPHONY_SCLV_DATA_DIR})
install(FILES ../../LICENSE DESTINATION ${SYMPHONY_SCLV_LICENSE_DIR} RENAME LICENSE-AGPL-3.0)
install(FILES ../../libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT
    DESTINATION ${SYMPHONY_SCLV_LICENSE_DIR}
    RENAME nlohmann-json-LICENSE.MIT)

symphony_install_receipt_v2(
    COMPONENT_ID sclv-engine COMPONENT_KIND vector_engine
    MODULE_ID sclv-engine VECTOR_ID sclv ENGINE_ID symphony-sclv
    PACKAGE_ID sclv-engine VERSION 0.1.0-dev
    RECEIPT_PATH share/symphony/receipts/sclv-engine/0.1.0-dev/install-receipt.json
    OWNED_FILES
        "libexec/symphony/sclv-engine/0.1.0-dev/symphony-sclv|executable"
        "libexec/symphony/sclv-engine/0.1.0-dev/symphony-sclv-evidence-airgap|executable"
        "libexec/symphony/sclv-engine/0.1.0-dev/symphony-sclv-evidence-local-git|executable"
        "share/doc/symphony/sclv-engine/0.1.0-dev/INSTALL.md|regular"
        "share/doc/symphony/sclv-engine/0.1.0-dev/INTENT.md|regular"
        "share/doc/symphony/sclv-engine/0.1.0-dev/MANIFEST.md|regular"
        "share/doc/symphony/sclv-engine/0.1.0-dev/SKILL.md|regular"
        "share/doc/symphony/sclv-engine/0.1.0-dev/SPEC.md|regular"
        "share/licenses/symphony-sclv-engine/0.1.0-dev/LICENSE-AGPL-3.0|regular"
        "share/licenses/symphony-sclv-engine/0.1.0-dev/nlohmann-json-LICENSE.MIT|regular"
    ENTRY_POINTS
        "symphony-sclv|executable|libexec/symphony/sclv-engine/0.1.0-dev/symphony-sclv|symphony.knowledge.engine-process.v1"
        "symphony-sclv-evidence-airgap|adapter|libexec/symphony/sclv-engine/0.1.0-dev/symphony-sclv-evidence-airgap|symphony.knowledge.provider-evidence.v1"
        "symphony-sclv-evidence-local-git|adapter|libexec/symphony/sclv-engine/0.1.0-dev/symphony-sclv-evidence-local-git|symphony.knowledge.provider-evidence.v1"
    COMPATIBLE_RECEPTORS symphony.maestro.knowledge-engine.v1)

add_custom_target(uninstall-sclv-engine
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
            -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
    COMMENT "Remove only sclv-engine receipt-owned files"
)
