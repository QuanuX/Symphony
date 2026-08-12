cmake_minimum_required(VERSION 3.25)
project(SymphonyKnowledgeSession VERSION 0.1.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)

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

add_executable(symphony-knowledge-session
    src/main.cpp
    src/coordinator.cpp
    src/reconciliation.cpp
    src/authority_session.cpp
    src/lifecycle.cpp
    src/lifecycle_journal.cpp
    src/ssfv_maintenance.cpp
)
target_include_directories(symphony-knowledge-session PRIVATE src)
target_link_libraries(symphony-knowledge-session PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(symphony-knowledge-session PROPERTIES
    CXX_STANDARD 26
    CXX_STANDARD_REQUIRED ON
    CXX_EXTENSIONS OFF
)
target_compile_options(symphony-knowledge-session PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
)

if(BUILD_TESTING)
    add_executable(knowledge-session-coordinator-tests
        tests/coordinator_test.cpp
        src/coordinator.cpp
        src/reconciliation.cpp
        src/authority_session.cpp
        src/lifecycle.cpp
        src/lifecycle_journal.cpp
        src/ssfv_maintenance.cpp
    )
    target_include_directories(knowledge-session-coordinator-tests PRIVATE src)
    target_link_libraries(knowledge-session-coordinator-tests PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(knowledge-session-coordinator-tests PROPERTIES
        CXX_STANDARD 26
        CXX_STANDARD_REQUIRED ON
        CXX_EXTENSIONS OFF
    )
    target_compile_options(knowledge-session-coordinator-tests PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
    )
    add_test(NAME knowledge-session-coordinator-tests COMMAND knowledge-session-coordinator-tests)
    add_executable(knowledge-lifecycle-planner-tests
        tests/lifecycle_test.cpp
        src/coordinator.cpp
        src/reconciliation.cpp
        src/authority_session.cpp
        src/lifecycle.cpp
        src/lifecycle_journal.cpp
        src/ssfv_maintenance.cpp
    )
    target_include_directories(knowledge-lifecycle-planner-tests PRIVATE src)
    target_link_libraries(knowledge-lifecycle-planner-tests PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(knowledge-lifecycle-planner-tests PROPERTIES
        CXX_STANDARD 26
        CXX_STANDARD_REQUIRED ON
        CXX_EXTENSIONS OFF
    )
    target_compile_options(knowledge-lifecycle-planner-tests PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
    )
    add_test(NAME knowledge-lifecycle-planner-tests COMMAND knowledge-lifecycle-planner-tests)
    add_executable(knowledge-ssfv-maintenance-tests
        tests/ssfv_maintenance_test.cpp
        src/coordinator.cpp
        src/reconciliation.cpp
        src/authority_session.cpp
        src/lifecycle.cpp
        src/lifecycle_journal.cpp
        src/ssfv_maintenance.cpp
    )
    target_include_directories(knowledge-ssfv-maintenance-tests PRIVATE src)
    target_link_libraries(knowledge-ssfv-maintenance-tests PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(knowledge-ssfv-maintenance-tests PROPERTIES
        CXX_STANDARD 26
        CXX_STANDARD_REQUIRED ON
        CXX_EXTENSIONS OFF
    )
    target_compile_options(knowledge-ssfv-maintenance-tests PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
    )
    add_test(NAME knowledge-ssfv-maintenance-tests COMMAND knowledge-ssfv-maintenance-tests)
    add_test(
        NAME knowledge-session-process-smoke
        COMMAND ${CMAKE_CURRENT_SOURCE_DIR}/tests/process_smoke.sh
                $<TARGET_FILE:symphony-knowledge-session>
                ${SYMPHONY_REPOSITORY_ROOT}
    )
endif()

configure_file(
    cmake/uninstall.cmake.in
    ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
    @ONLY
)

set(SYMPHONY_SESSION_LIBEXEC_DIR
    "${CMAKE_INSTALL_LIBEXECDIR}/symphony/knowledge-session-coordinator/0.1.0-dev")
set(SYMPHONY_SESSION_DATA_DIR
    "${CMAKE_INSTALL_DATADIR}/doc/symphony/knowledge-session-coordinator/0.1.0-dev")
set(SYMPHONY_SESSION_LICENSE_DIR
    "${CMAKE_INSTALL_DATADIR}/licenses/symphony-knowledge-session/0.1.0-dev")

symphony_install_receipt_v2_preflight(
    RECEIPT_PATH share/symphony/receipts/knowledge-session-coordinator/0.1.0-dev/install-receipt.json)

install(TARGETS symphony-knowledge-session RUNTIME DESTINATION ${SYMPHONY_SESSION_LIBEXEC_DIR})
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md DESTINATION ${SYMPHONY_SESSION_DATA_DIR})
install(FILES ../../LICENSE DESTINATION ${SYMPHONY_SESSION_LICENSE_DIR} RENAME LICENSE-AGPL-3.0)
install(FILES ../../libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT
    DESTINATION ${SYMPHONY_SESSION_LICENSE_DIR}
    RENAME nlohmann-json-LICENSE.MIT)

symphony_install_receipt_v2(
    COMPONENT_ID knowledge-session-coordinator COMPONENT_KIND coordinator
    MODULE_ID knowledge-session-coordinator ENGINE_ID symphony-knowledge-session
    PACKAGE_ID knowledge-session-coordinator VERSION 0.1.0-dev
    RECEIPT_PATH share/symphony/receipts/knowledge-session-coordinator/0.1.0-dev/install-receipt.json
    OWNED_FILES
        "libexec/symphony/knowledge-session-coordinator/0.1.0-dev/symphony-knowledge-session|executable"
        "share/doc/symphony/knowledge-session-coordinator/0.1.0-dev/INSTALL.md|regular"
        "share/doc/symphony/knowledge-session-coordinator/0.1.0-dev/INTENT.md|regular"
        "share/doc/symphony/knowledge-session-coordinator/0.1.0-dev/MANIFEST.md|regular"
        "share/doc/symphony/knowledge-session-coordinator/0.1.0-dev/SKILL.md|regular"
        "share/doc/symphony/knowledge-session-coordinator/0.1.0-dev/SPEC.md|regular"
        "share/licenses/symphony-knowledge-session/0.1.0-dev/LICENSE-AGPL-3.0|regular"
        "share/licenses/symphony-knowledge-session/0.1.0-dev/nlohmann-json-LICENSE.MIT|regular"
    ENTRY_POINTS
        "symphony-knowledge-session|executable|libexec/symphony/knowledge-session-coordinator/0.1.0-dev/symphony-knowledge-session|symphony.knowledge.engine-process.v1")

add_custom_target(uninstall-knowledge-session-coordinator
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
            -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
    COMMENT "Remove only knowledge-session-coordinator receipt-owned files"
)
