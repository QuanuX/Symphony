cmake_minimum_required(VERSION 3.30)
project(SymphonySqmvMetadata VERSION 0.2.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

if(CMAKE_CONFIGURATION_TYPES)
    message(FATAL_ERROR "sqmv-metadata-cpp requires a single-configuration generator for exact receipt-owned exports")
endif()
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()
string(TOLOWER "${CMAKE_BUILD_TYPE}" SQMV_BUILD_TYPE_LOWER)

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")

option(SYMPHONY_SQMV_USE_INSTALLED_DEPENDENCIES "Use exact installed SQMV dependencies" OFF)
if(SYMPHONY_SQMV_USE_INSTALLED_DEPENDENCIES)
    find_package(SymphonyKnowledgeVectorEngine 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqfvBatch 0.3.0 EXACT REQUIRED CONFIG)
else()
    function(sqmv_source_dependencies)
        set(BUILD_TESTING OFF)
        set(SYMPHONY_KVE_BUILD_TESTING OFF)
        if(NOT TARGET Symphony::KnowledgeVectorEngine)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/knowledge-vector-engine-cpp" EXCLUDE_FROM_ALL)
        endif()
        if(NOT TARGET Symphony::SqfvBatch)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqfv-batch-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqfv-batch-cpp" EXCLUDE_FROM_ALL)
        endif()
    endfunction()
    sqmv_source_dependencies()
endif()


set(SQMV_MODULE sqmv-metadata-cpp)
set(SQMV_VERSION "${PROJECT_VERSION}-dev")
set(SQMV_DOCS "share/doc/symphony/${SQMV_MODULE}/${SQMV_VERSION}")
set(SQMV_INCLUDE_ROOT "${CMAKE_INSTALL_INCLUDEDIR}/symphony/${SQMV_MODULE}/${SQMV_VERSION}")
set(SQMV_HEADER_DIR "${SQMV_INCLUDE_ROOT}/symphony/sqmv")
set(SQMV_ARCHIVE_DIR "${CMAKE_INSTALL_LIBDIR}/symphony/${SQMV_MODULE}/${SQMV_VERSION}")
set(SQMV_CMAKE_DIR "${CMAKE_INSTALL_LIBDIR}/cmake/SymphonySqmvMetadata-${SQMV_VERSION}")
set(SQMV_LICENSE_DIR "share/licenses/symphony-${SQMV_MODULE}/${SQMV_VERSION}")
set(SQMV_RECEIPT "share/symphony/receipts/${SQMV_MODULE}/${SQMV_VERSION}/install-receipt.json")

# Apple ar and ranlib otherwise embed changing member and symbol-table timestamps.
if(APPLE)
    set(CMAKE_CXX_ARCHIVE_CREATE "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> qc <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_APPEND "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> q <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_FINISH "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_RANLIB> <TARGET>")
endif()

add_library(sqmv-metadata STATIC src/metadata.cpp)
add_library(Symphony::SqmvMetadata ALIAS sqmv-metadata)
target_link_libraries(sqmv-metadata
    PUBLIC Symphony::SqfvBatch
    PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(sqmv-metadata PROPERTIES
    EXPORT_NAME SqmvMetadata
    CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF
)
target_compile_features(sqmv-metadata PUBLIC cxx_std_26)
target_include_directories(sqmv-metadata PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${SQMV_INCLUDE_ROOT}>)
target_compile_options(sqmv-metadata PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

symphony_install_receipt_v2_preflight(RECEIPT_PATH "${SQMV_RECEIPT}")

set(SQMV_OWNED_FILES "")
set(SQMV_INSTALLED_FILES "")
macro(sqmv_own path)
    list(APPEND SQMV_OWNED_FILES "${path}|regular")
    list(APPEND SQMV_INSTALLED_FILES "${path}")
endmacro()

install(TARGETS sqmv-metadata EXPORT SymphonySqmvMetadataTargets
    ARCHIVE DESTINATION "${SQMV_ARCHIVE_DIR}")
sqmv_own("${SQMV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqmv-metadata${CMAKE_STATIC_LIBRARY_SUFFIX}")
if(APPLE)
    # CMake invokes ranlib again during install; reseal before receipt hashing.
    install(CODE "execute_process(COMMAND \"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 \"${CMAKE_RANLIB}\" \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${SQMV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqmv-metadata${CMAKE_STATIC_LIBRARY_SUFFIX}\" COMMAND_ERROR_IS_FATAL ANY)")
endif()

install(FILES include/symphony/sqmv/metadata.hpp DESTINATION "${SQMV_HEADER_DIR}")
sqmv_own("${SQMV_HEADER_DIR}/metadata.hpp")

configure_package_config_file(cmake/SymphonySqmvMetadataConfig.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqmvMetadataConfig.cmake"
    INSTALL_DESTINATION "${SQMV_CMAKE_DIR}")
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqmvMetadataConfigVersion.cmake"
    VERSION ${PROJECT_VERSION} COMPATIBILITY ExactVersion)
install(EXPORT SymphonySqmvMetadataTargets
    FILE SymphonySqmvMetadataTargets.cmake NAMESPACE Symphony::
    DESTINATION "${SQMV_CMAKE_DIR}")
install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqmvMetadataConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqmvMetadataConfigVersion.cmake"
    DESTINATION "${SQMV_CMAKE_DIR}")
foreach(name
    SymphonySqmvMetadataConfig.cmake
    SymphonySqmvMetadataConfigVersion.cmake
    SymphonySqmvMetadataTargets.cmake
    "SymphonySqmvMetadataTargets-${SQMV_BUILD_TYPE_LOWER}.cmake")
    sqmv_own("${SQMV_CMAKE_DIR}/${name}")
endforeach()

foreach(doc INTENT.md MANIFEST.md SPEC.md SKILL.md INSTALL.md FEATURES.md)
    install(FILES "${CMAKE_CURRENT_SOURCE_DIR}/${doc}" DESTINATION "${SQMV_DOCS}")
    sqmv_own("${SQMV_DOCS}/${doc}")
endforeach()
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/LICENSE"
    DESTINATION "${SQMV_LICENSE_DIR}" RENAME LICENSE-AGPL-3.0)
sqmv_own("${SQMV_LICENSE_DIR}/LICENSE-AGPL-3.0")

symphony_install_receipt_v2(
    COMPONENT_ID "${SQMV_MODULE}" COMPONENT_KIND module MODULE_ID "${SQMV_MODULE}"
    VECTOR_ID sqmv PACKAGE_ID "${SQMV_MODULE}" VERSION "${SQMV_VERSION}"
    RECEIPT_PATH "${SQMV_RECEIPT}" OWNED_FILES ${SQMV_OWNED_FILES})

configure_file(cmake/uninstall.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake" @ONLY)
add_custom_target(uninstall-${SQMV_MODULE}
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
        -P "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake")

if(BUILD_TESTING)
    add_executable(sqmv-metadata-test tests/metadata_test.cpp)
    target_link_libraries(sqmv-metadata-test PRIVATE
        Symphony::SqmvMetadata Symphony::KnowledgeVectorEngine)
    set_target_properties(sqmv-metadata-test PROPERTIES
        CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqmv-metadata-test PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sqmv-metadata COMMAND sqmv-metadata-test)
    add_test(NAME sqmv-package-lifecycle
        COMMAND "${CMAKE_COMMAND}"
            "-DPACKAGE_BINARY_DIR=${CMAKE_CURRENT_BINARY_DIR}"
            "-DPACKAGE_MODULE=sqmv-metadata-cpp"
            "-DPACKAGE_CMAKE_NAME=SymphonySqmvMetadata"
            -P "${CMAKE_CURRENT_SOURCE_DIR}/tests/package_lifecycle_test.cmake")
endif()
