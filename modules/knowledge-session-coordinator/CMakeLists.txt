cmake_minimum_required(VERSION 3.25)
project(SymphonyKnowledgeSession VERSION 0.1.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)

option(SYMPHONY_KVE_USE_INSTALLED "Use an installed Symphony knowledge-vector foundation" OFF)
get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
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
    add_test(
        NAME knowledge-session-process-smoke
        COMMAND ${CMAKE_CURRENT_SOURCE_DIR}/tests/process_smoke.sh
                $<TARGET_FILE:symphony-knowledge-session>
                ${SYMPHONY_REPOSITORY_ROOT}
    )
endif()

configure_file(
    cmake/install-receipt.json.in
    ${CMAKE_CURRENT_BINARY_DIR}/install-receipt.json
    @ONLY
)
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

install(TARGETS symphony-knowledge-session RUNTIME DESTINATION ${SYMPHONY_SESSION_LIBEXEC_DIR})
install(FILES ${CMAKE_CURRENT_BINARY_DIR}/install-receipt.json
    DESTINATION ${CMAKE_INSTALL_DATADIR}/symphony/receipts/knowledge-session-coordinator/0.1.0-dev)
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md DESTINATION ${SYMPHONY_SESSION_DATA_DIR})
install(FILES ../../LICENSE DESTINATION ${SYMPHONY_SESSION_LICENSE_DIR} RENAME LICENSE-AGPL-3.0)
install(FILES ../../libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT
    DESTINATION ${SYMPHONY_SESSION_LICENSE_DIR}
    RENAME nlohmann-json-LICENSE.MIT)

add_custom_target(uninstall-knowledge-session-coordinator
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
            -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
    COMMENT "Remove only knowledge-session-coordinator receipt-owned files"
)
