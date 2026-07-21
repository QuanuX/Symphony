cmake_minimum_required(VERSION 3.25)
project(SymphonyKnowledgeVectorEngine VERSION 0.1.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

if(CMAKE_CONFIGURATION_TYPES)
    message(FATAL_ERROR
        "knowledge-vector-engine-cpp 0.1.0-dev requires a single-configuration "
        "generator so its deterministic install receipt names the exact CMake export file")
endif()

if(NOT CMAKE_CONFIGURATION_TYPES AND NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()
string(TOLOWER "${CMAKE_BUILD_TYPE}" SYMPHONY_KVE_BUILD_TYPE_LOWER)
if(SYMPHONY_KVE_BUILD_TYPE_LOWER STREQUAL "")
    set(SYMPHONY_KVE_BUILD_TYPE_LOWER "release")
endif()
set(SYMPHONY_KVE_EXPORT_CONFIG_FILE
    "SymphonyKnowledgeVectorEngineTargets-${SYMPHONY_KVE_BUILD_TYPE_LOWER}.cmake")

set(SYMPHONY_KVE_INCLUDE_DIR
    "${CMAKE_INSTALL_INCLUDEDIR}/symphony/knowledge-vector-engine/0.1.0-dev")
set(SYMPHONY_KVE_CMAKE_DIR
    "${CMAKE_INSTALL_LIBDIR}/cmake/SymphonyKnowledgeVectorEngine-0.1.0-dev")

add_library(symphony-knowledge-vector-engine STATIC
    src/digest.cpp
    src/error.cpp
    src/path.cpp
    src/protocol.cpp
)
add_library(Symphony::KnowledgeVectorEngine ALIAS symphony-knowledge-vector-engine)

target_include_directories(symphony-knowledge-vector-engine
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:${SYMPHONY_KVE_INCLUDE_DIR}>
)
target_include_directories(symphony-knowledge-vector-engine
    SYSTEM PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/third_party>
        $<INSTALL_INTERFACE:${SYMPHONY_KVE_INCLUDE_DIR}>
)
set_target_properties(symphony-knowledge-vector-engine PROPERTIES
    CXX_STANDARD 26
    CXX_STANDARD_REQUIRED ON
    CXX_EXTENSIONS OFF
    EXPORT_NAME KnowledgeVectorEngine
)
target_compile_options(symphony-knowledge-vector-engine PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
)

option(SYMPHONY_KVE_BUILD_TESTING "Build knowledge-vector foundation tests" ${BUILD_TESTING})
if(SYMPHONY_KVE_BUILD_TESTING)
    get_filename_component(SYMPHONY_KVE_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
    add_executable(knowledge-vector-engine-foundation-tests tests/foundation_test.cpp)
    target_link_libraries(knowledge-vector-engine-foundation-tests PRIVATE Symphony::KnowledgeVectorEngine)
    set_target_properties(knowledge-vector-engine-foundation-tests PROPERTIES
        CXX_STANDARD 26
        CXX_STANDARD_REQUIRED ON
        CXX_EXTENSIONS OFF
    )
    target_compile_options(knowledge-vector-engine-foundation-tests PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>
    )
    add_test(
        NAME knowledge-vector-engine-foundation-tests
        COMMAND knowledge-vector-engine-foundation-tests ${SYMPHONY_KVE_REPOSITORY_ROOT}
    )
endif()

install(TARGETS symphony-knowledge-vector-engine
    EXPORT SymphonyKnowledgeVectorEngineTargets
    ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}/symphony/knowledge-vector-engine/0.1.0-dev
    INCLUDES DESTINATION ${SYMPHONY_KVE_INCLUDE_DIR}
)
install(DIRECTORY include/ DESTINATION ${SYMPHONY_KVE_INCLUDE_DIR})
install(FILES third_party/nlohmann/json.hpp DESTINATION ${SYMPHONY_KVE_INCLUDE_DIR}/nlohmann)
install(FILES third_party/nlohmann/LICENSE.MIT
    DESTINATION ${CMAKE_INSTALL_DATADIR}/licenses/symphony-knowledge-vector-engine/0.1.0-dev
    RENAME nlohmann-json-LICENSE.MIT
)

configure_package_config_file(
    cmake/SymphonyKnowledgeVectorEngineConfig.cmake.in
    ${CMAKE_CURRENT_BINARY_DIR}/SymphonyKnowledgeVectorEngineConfig.cmake
    INSTALL_DESTINATION ${SYMPHONY_KVE_CMAKE_DIR}
)
write_basic_package_version_file(
    ${CMAKE_CURRENT_BINARY_DIR}/SymphonyKnowledgeVectorEngineConfigVersion.cmake
    VERSION ${PROJECT_VERSION}
    COMPATIBILITY SameMajorVersion
)
install(EXPORT SymphonyKnowledgeVectorEngineTargets
    FILE SymphonyKnowledgeVectorEngineTargets.cmake
    NAMESPACE Symphony::
    DESTINATION ${SYMPHONY_KVE_CMAKE_DIR}
)
install(FILES
    ${CMAKE_CURRENT_BINARY_DIR}/SymphonyKnowledgeVectorEngineConfig.cmake
    ${CMAKE_CURRENT_BINARY_DIR}/SymphonyKnowledgeVectorEngineConfigVersion.cmake
    DESTINATION ${SYMPHONY_KVE_CMAKE_DIR}
)

configure_file(cmake/install-receipt.json.in ${CMAKE_CURRENT_BINARY_DIR}/install-receipt.json @ONLY)
configure_file(cmake/uninstall.cmake.in ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake @ONLY)
install(FILES ${CMAKE_CURRENT_BINARY_DIR}/install-receipt.json
    DESTINATION ${CMAKE_INSTALL_DATADIR}/symphony/receipts/knowledge-vector-engine-cpp/0.1.0-dev)
install(FILES INTENT.md MANIFEST.md INSTALL.md SKILL.md SPEC.md
    DESTINATION ${CMAKE_INSTALL_DATADIR}/doc/symphony/knowledge-vector-engine-cpp/0.1.0-dev)
install(FILES ../../LICENSE
    DESTINATION ${CMAKE_INSTALL_DATADIR}/licenses/symphony-knowledge-vector-engine/0.1.0-dev
    RENAME LICENSE-AGPL-3.0)

add_custom_target(uninstall-knowledge-vector-engine-cpp
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
            -P ${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake
    COMMENT "Remove only knowledge-vector-engine-cpp receipt-owned files"
)
