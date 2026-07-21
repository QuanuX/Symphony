cmake_minimum_required(VERSION 3.25)
project(SymphonySacvEngine VERSION 0.1.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)

if(NOT CMAKE_INSTALL_LIBEXECDIR STREQUAL "libexec" OR
   NOT CMAKE_INSTALL_DATADIR STREQUAL "share")
    message(FATAL_ERROR
        "sacv-engine 0.1.0-dev requires the exact libexec/share receipt layout used by qxctl")
endif()

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

add_executable(symphony-sacv src/main.cpp src/sacv.cpp)
target_include_directories(symphony-sacv PRIVATE src)
target_link_libraries(symphony-sacv PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(symphony-sacv PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
target_compile_options(symphony-sacv PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

if(BUILD_TESTING)
    add_executable(sacv-engine-tests tests/sacv_test.cpp src/sacv.cpp)
    target_include_directories(sacv-engine-tests PRIVATE src)
    target_link_libraries(sacv-engine-tests PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(sacv-engine-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sacv-engine-tests PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sacv-engine-tests COMMAND sacv-engine-tests ${SYMPHONY_REPOSITORY_ROOT})
    add_test(NAME sacv-engine-process-smoke
        COMMAND ${CMAKE_CURRENT_SOURCE_DIR}/tests/process_smoke.sh
                $<TARGET_FILE:symphony-sacv> ${SYMPHONY_REPOSITORY_ROOT})
endif()

set(SYMPHONY_SACV_LIBEXEC_DIR "${CMAKE_INSTALL_LIBEXECDIR}/symphony/sacv-engine/0.1.0-dev")
set(SYMPHONY_SACV_DATA_DIR "${CMAKE_INSTALL_DATADIR}/doc/symphony/sacv-engine/0.1.0-dev")
set(SYMPHONY_SACV_LICENSE_DIR "${CMAKE_INSTALL_DATADIR}/licenses/symphony-sacv-engine/0.1.0-dev")

configure_file(cmake/install-receipt.json.in ${CMAKE_CURRENT_BINARY_DIR}/install-receipt.json @ONLY)
configure_file(cmake/uninstall.cmake.in ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake @ONLY)

install(TARGETS symphony-sacv RUNTIME DESTINATION ${SYMPHONY_SACV_LIBEXEC_DIR})
install(FILES ${CMAKE_CURRENT_BINARY_DIR}/install-receipt.json
    DESTINATION ${CMAKE_INSTALL_DATADIR}/symphony/receipts/sacv-engine/0.1.0-dev)
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md DESTINATION ${SYMPHONY_SACV_DATA_DIR})
install(FILES ../../LICENSE DESTINATION ${SYMPHONY_SACV_LICENSE_DIR} RENAME LICENSE-AGPL-3.0)
install(FILES ../../libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT
    DESTINATION ${SYMPHONY_SACV_LICENSE_DIR} RENAME nlohmann-json-LICENSE.MIT)

add_custom_target(uninstall-sacv-engine
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
            -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
    COMMENT "Remove only sacv-engine receipt-owned files")
