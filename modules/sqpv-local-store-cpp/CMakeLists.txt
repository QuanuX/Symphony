cmake_minimum_required(VERSION 3.30)
project(SymphonySqpvLocalStore VERSION 0.2.0 LANGUAGES CXX)

include(CTest)
find_package(Threads REQUIRED)
include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

if(CMAKE_CONFIGURATION_TYPES)
    message(FATAL_ERROR "sqpv-local-store-cpp requires a single-configuration generator for exact receipt-owned exports")
endif()
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()
string(TOLOWER "${CMAKE_BUILD_TYPE}" SQPV_BUILD_TYPE_LOWER)

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")

option(SYMPHONY_SQPV_USE_INSTALLED_DEPENDENCIES "Use exact installed SQPV dependencies" OFF)
if(SYMPHONY_SQPV_USE_INSTALLED_DEPENDENCIES)
    find_package(SymphonyKnowledgeVectorEngine 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqmvMetadata 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqfvBatch 0.3.0 EXACT REQUIRED CONFIG)
else()
    function(sqpv_source_dependencies)
        set(BUILD_TESTING OFF)
        set(SYMPHONY_SQMV_USE_INSTALLED_DEPENDENCIES OFF)
        if(NOT TARGET Symphony::SqmvMetadata)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqmv-metadata-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqmv-metadata-cpp" EXCLUDE_FROM_ALL)
        endif()
        if(NOT TARGET Symphony::SqfvBatch)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqfv-batch-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqfv-batch-cpp" EXCLUDE_FROM_ALL)
        endif()
    endfunction()
    sqpv_source_dependencies()
endif()


set(SQPV_MODULE sqpv-local-store-cpp)
set(SQPV_VERSION "${PROJECT_VERSION}-dev")
set(SQPV_DOCS "share/doc/symphony/${SQPV_MODULE}/${SQPV_VERSION}")
set(SQPV_INCLUDE_ROOT "${CMAKE_INSTALL_INCLUDEDIR}/symphony/${SQPV_MODULE}/${SQPV_VERSION}")
set(SQPV_HEADER_DIR "${SQPV_INCLUDE_ROOT}/symphony/sqpv")
set(SQPV_ARCHIVE_DIR "${CMAKE_INSTALL_LIBDIR}/symphony/${SQPV_MODULE}/${SQPV_VERSION}")
set(SQPV_CMAKE_DIR "${CMAKE_INSTALL_LIBDIR}/cmake/SymphonySqpvLocalStore-${SQPV_VERSION}")
set(SQPV_LICENSE_DIR "share/licenses/symphony-${SQPV_MODULE}/${SQPV_VERSION}")
set(SQPV_RECEIPT "share/symphony/receipts/${SQPV_MODULE}/${SQPV_VERSION}/install-receipt.json")

# Apple ar and ranlib otherwise embed changing member and symbol-table timestamps.
if(APPLE)
    set(CMAKE_CXX_ARCHIVE_CREATE "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> qc <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_APPEND "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> q <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_FINISH "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_RANLIB> <TARGET>")
endif()

add_library(sqpv-local-store STATIC src/local_store.cpp src/async_store.cpp)
add_library(Symphony::SqpvLocalStore ALIAS sqpv-local-store)
target_link_libraries(sqpv-local-store
    PUBLIC Symphony::SqmvMetadata Symphony::SqfvBatch Threads::Threads
    PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(sqpv-local-store PROPERTIES
    EXPORT_NAME SqpvLocalStore
    CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF
)
target_compile_features(sqpv-local-store PUBLIC cxx_std_26)
target_include_directories(sqpv-local-store PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${SQPV_INCLUDE_ROOT}>)
target_compile_options(sqpv-local-store PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

symphony_install_receipt_v2_preflight(RECEIPT_PATH "${SQPV_RECEIPT}")

set(SQPV_OWNED_FILES "")
set(SQPV_INSTALLED_FILES "")
macro(sqpv_own path)
    list(APPEND SQPV_OWNED_FILES "${path}|regular")
    list(APPEND SQPV_INSTALLED_FILES "${path}")
endmacro()

install(TARGETS sqpv-local-store EXPORT SymphonySqpvLocalStoreTargets
    ARCHIVE DESTINATION "${SQPV_ARCHIVE_DIR}")
sqpv_own("${SQPV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqpv-local-store${CMAKE_STATIC_LIBRARY_SUFFIX}")
if(APPLE)
    # CMake invokes ranlib again during install; reseal before receipt hashing.
    install(CODE "execute_process(COMMAND \"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 \"${CMAKE_RANLIB}\" \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${SQPV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqpv-local-store${CMAKE_STATIC_LIBRARY_SUFFIX}\" COMMAND_ERROR_IS_FATAL ANY)")
endif()

install(FILES include/symphony/sqpv/local_store.hpp include/symphony/sqpv/async_store.hpp DESTINATION "${SQPV_HEADER_DIR}")
sqpv_own("${SQPV_HEADER_DIR}/local_store.hpp")
sqpv_own("${SQPV_HEADER_DIR}/async_store.hpp")

configure_package_config_file(cmake/SymphonySqpvLocalStoreConfig.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqpvLocalStoreConfig.cmake"
    INSTALL_DESTINATION "${SQPV_CMAKE_DIR}")
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqpvLocalStoreConfigVersion.cmake"
    VERSION ${PROJECT_VERSION} COMPATIBILITY ExactVersion)
install(EXPORT SymphonySqpvLocalStoreTargets
    FILE SymphonySqpvLocalStoreTargets.cmake NAMESPACE Symphony::
    DESTINATION "${SQPV_CMAKE_DIR}")
install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqpvLocalStoreConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqpvLocalStoreConfigVersion.cmake"
    DESTINATION "${SQPV_CMAKE_DIR}")
foreach(name
    SymphonySqpvLocalStoreConfig.cmake
    SymphonySqpvLocalStoreConfigVersion.cmake
    SymphonySqpvLocalStoreTargets.cmake
    "SymphonySqpvLocalStoreTargets-${SQPV_BUILD_TYPE_LOWER}.cmake")
    sqpv_own("${SQPV_CMAKE_DIR}/${name}")
endforeach()

foreach(doc INTENT.md MANIFEST.md SPEC.md SKILL.md INSTALL.md FEATURES.md)
    install(FILES "${CMAKE_CURRENT_SOURCE_DIR}/${doc}" DESTINATION "${SQPV_DOCS}")
    sqpv_own("${SQPV_DOCS}/${doc}")
endforeach()
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/LICENSE"
    DESTINATION "${SQPV_LICENSE_DIR}" RENAME LICENSE-AGPL-3.0)
sqpv_own("${SQPV_LICENSE_DIR}/LICENSE-AGPL-3.0")

symphony_install_receipt_v2(
    COMPONENT_ID "${SQPV_MODULE}" COMPONENT_KIND module MODULE_ID "${SQPV_MODULE}"
    VECTOR_ID sqpv PACKAGE_ID "${SQPV_MODULE}" VERSION "${SQPV_VERSION}"
    RECEIPT_PATH "${SQPV_RECEIPT}" OWNED_FILES ${SQPV_OWNED_FILES})

configure_file(cmake/uninstall.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake" @ONLY)
add_custom_target(uninstall-${SQPV_MODULE}
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
        -P "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake")

if(BUILD_TESTING)
    # Fault/crash hooks are compiled only into this private test archive.
    add_library(sqpv-local-store-testing STATIC src/local_store.cpp src/async_store.cpp)
    target_link_libraries(sqpv-local-store-testing
        PUBLIC Symphony::SqmvMetadata Symphony::SqfvBatch Threads::Threads
        PRIVATE Symphony::KnowledgeVectorEngine)
    target_include_directories(sqpv-local-store-testing PUBLIC
        "${CMAKE_CURRENT_SOURCE_DIR}/include"
        PRIVATE "${CMAKE_CURRENT_SOURCE_DIR}/tests")
    target_compile_definitions(sqpv-local-store-testing PRIVATE SQPV_TESTING=1)
    set_target_properties(sqpv-local-store-testing PROPERTIES
        CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqpv-local-store-testing PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_executable(sqpv-local-store-test tests/local_store_test.cpp)
    target_link_libraries(sqpv-local-store-test PRIVATE sqpv-local-store-testing)
    target_compile_definitions(sqpv-local-store-test PRIVATE SQPV_TESTING=1)
    target_include_directories(sqpv-local-store-test PRIVATE
        "${CMAKE_CURRENT_SOURCE_DIR}/tests")
    set_target_properties(sqpv-local-store-test PROPERTIES
        CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqpv-local-store-test PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sqpv-local-store COMMAND sqpv-local-store-test)
    add_executable(sqpv-async-store-test tests/async_store_test.cpp)
    target_link_libraries(sqpv-async-store-test PRIVATE sqpv-local-store-testing)
    target_include_directories(sqpv-async-store-test PRIVATE "${CMAKE_CURRENT_SOURCE_DIR}/tests")
    set_target_properties(sqpv-async-store-test PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqpv-async-store-test PRIVATE -Wall -Wextra -Wpedantic -Werror)
    add_test(NAME sqpv-async-store COMMAND sqpv-async-store-test)
    add_test(NAME sqpv-package-lifecycle
        COMMAND "${CMAKE_COMMAND}"
            "-DPACKAGE_BINARY_DIR=${CMAKE_CURRENT_BINARY_DIR}"
            "-DPACKAGE_MODULE=sqpv-local-store-cpp"
            "-DPACKAGE_CMAKE_NAME=SymphonySqpvLocalStore"
            -P "${CMAKE_CURRENT_SOURCE_DIR}/tests/package_lifecycle_test.cmake")
endif()
