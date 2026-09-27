cmake_minimum_required(VERSION 3.30)
project(SymphonySqfvBatch VERSION 0.2.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

if(CMAKE_CONFIGURATION_TYPES)
    message(FATAL_ERROR "sqfv-batch-cpp requires a single-configuration generator for exact receipt-owned exports")
endif()
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()
string(TOLOWER "${CMAKE_BUILD_TYPE}" SQFV_BUILD_TYPE_LOWER)

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")

set(SQFV_MODULE sqfv-batch-cpp)
set(SQFV_VERSION "${PROJECT_VERSION}-dev")
set(SQFV_DOCS "share/doc/symphony/${SQFV_MODULE}/${SQFV_VERSION}")
set(SQFV_INCLUDE_ROOT "${CMAKE_INSTALL_INCLUDEDIR}/symphony/${SQFV_MODULE}/${SQFV_VERSION}")
set(SQFV_HEADER_DIR "${SQFV_INCLUDE_ROOT}/symphony/sqfv")
set(SQFV_ARCHIVE_DIR "${CMAKE_INSTALL_LIBDIR}/symphony/${SQFV_MODULE}/${SQFV_VERSION}")
set(SQFV_CMAKE_DIR "${CMAKE_INSTALL_LIBDIR}/cmake/SymphonySqfvBatch-${SQFV_VERSION}")
set(SQFV_LICENSE_DIR "share/licenses/symphony-${SQFV_MODULE}/${SQFV_VERSION}")
set(SQFV_RECEIPT "share/symphony/receipts/${SQFV_MODULE}/${SQFV_VERSION}/install-receipt.json")

# Apple ar and ranlib otherwise embed changing member and symbol-table timestamps.
if(APPLE)
    set(CMAKE_CXX_ARCHIVE_CREATE "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> qc <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_APPEND "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> q <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_FINISH "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_RANLIB> <TARGET>")
endif()

add_library(sqfv-batch STATIC src/batch.cpp src/frame.cpp)
add_library(Symphony::SqfvBatch ALIAS sqfv-batch)
set_target_properties(sqfv-batch PROPERTIES
    EXPORT_NAME SqfvBatch
    CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF
)
target_compile_features(sqfv-batch PUBLIC cxx_std_26)
target_include_directories(sqfv-batch PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${SQFV_INCLUDE_ROOT}>)
target_compile_options(sqfv-batch PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

symphony_install_receipt_v2_preflight(RECEIPT_PATH "${SQFV_RECEIPT}")

set(SQFV_OWNED_FILES "")
set(SQFV_INSTALLED_FILES "")
macro(sqfv_own path)
    list(APPEND SQFV_OWNED_FILES "${path}|regular")
    list(APPEND SQFV_INSTALLED_FILES "${path}")
endmacro()

install(TARGETS sqfv-batch EXPORT SymphonySqfvBatchTargets
    ARCHIVE DESTINATION "${SQFV_ARCHIVE_DIR}")
sqfv_own("${SQFV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqfv-batch${CMAKE_STATIC_LIBRARY_SUFFIX}")
if(APPLE)
    # CMake invokes ranlib again during install; reseal before receipt hashing.
    install(CODE "execute_process(COMMAND \"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 \"${CMAKE_RANLIB}\" \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${SQFV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqfv-batch${CMAKE_STATIC_LIBRARY_SUFFIX}\" COMMAND_ERROR_IS_FATAL ANY)")
endif()

install(FILES include/symphony/sqfv/batch.hpp DESTINATION "${SQFV_HEADER_DIR}")
sqfv_own("${SQFV_HEADER_DIR}/batch.hpp")

configure_package_config_file(cmake/SymphonySqfvBatchConfig.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqfvBatchConfig.cmake"
    INSTALL_DESTINATION "${SQFV_CMAKE_DIR}")
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqfvBatchConfigVersion.cmake"
    VERSION ${PROJECT_VERSION} COMPATIBILITY ExactVersion)
install(EXPORT SymphonySqfvBatchTargets
    FILE SymphonySqfvBatchTargets.cmake NAMESPACE Symphony::
    DESTINATION "${SQFV_CMAKE_DIR}")
install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqfvBatchConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqfvBatchConfigVersion.cmake"
    DESTINATION "${SQFV_CMAKE_DIR}")
foreach(name
    SymphonySqfvBatchConfig.cmake
    SymphonySqfvBatchConfigVersion.cmake
    SymphonySqfvBatchTargets.cmake
    "SymphonySqfvBatchTargets-${SQFV_BUILD_TYPE_LOWER}.cmake")
    sqfv_own("${SQFV_CMAKE_DIR}/${name}")
endforeach()

foreach(doc INTENT.md MANIFEST.md SPEC.md SKILL.md INSTALL.md FEATURES.md)
    install(FILES "${CMAKE_CURRENT_SOURCE_DIR}/${doc}" DESTINATION "${SQFV_DOCS}")
    sqfv_own("${SQFV_DOCS}/${doc}")
endforeach()
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/LICENSE"
    DESTINATION "${SQFV_LICENSE_DIR}" RENAME LICENSE-AGPL-3.0)
sqfv_own("${SQFV_LICENSE_DIR}/LICENSE-AGPL-3.0")

symphony_install_receipt_v2(
    COMPONENT_ID "${SQFV_MODULE}" COMPONENT_KIND module MODULE_ID "${SQFV_MODULE}"
    VECTOR_ID sqfv PACKAGE_ID "${SQFV_MODULE}" VERSION "${SQFV_VERSION}"
    RECEIPT_PATH "${SQFV_RECEIPT}" OWNED_FILES ${SQFV_OWNED_FILES})

configure_file(cmake/uninstall.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake" @ONLY)
add_custom_target(uninstall-${SQFV_MODULE}
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
        -P "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake")

if(BUILD_TESTING)
    find_package(Threads REQUIRED)
    foreach(test_name batch frame allocation_failure)
        if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/tests/${test_name}_test.cpp")
            string(REPLACE "_" "-" test_target "${test_name}")
            add_executable(sqfv-${test_target}-test "tests/${test_name}_test.cpp")
            target_link_libraries(sqfv-${test_target}-test PRIVATE Symphony::SqfvBatch)
            if(test_name STREQUAL "batch")
                target_link_libraries(sqfv-${test_name}-test PRIVATE Threads::Threads)
            endif()
            set_target_properties(sqfv-${test_target}-test PROPERTIES
                CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
            target_compile_options(sqfv-${test_target}-test PRIVATE
                $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
            add_test(NAME sqfv-${test_target} COMMAND sqfv-${test_target}-test)
        endif()
    endforeach()
    add_test(NAME sqfv-package-lifecycle
        COMMAND "${CMAKE_COMMAND}"
            "-DSQFV_SOURCE_DIR=${CMAKE_CURRENT_SOURCE_DIR}"
            "-DSQFV_BINARY_DIR=${CMAKE_CURRENT_BINARY_DIR}"
            -P "${CMAKE_CURRENT_SOURCE_DIR}/tests/package_lifecycle_test.cmake")
endif()
