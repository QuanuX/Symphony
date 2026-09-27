cmake_minimum_required(VERSION 3.30)
project(SymphonySqavCapture VERSION 0.1.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

if(CMAKE_CONFIGURATION_TYPES)
    message(FATAL_ERROR "sqav-capture-cpp requires a single-configuration generator for exact receipt-owned exports")
endif()
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()
string(TOLOWER "${CMAKE_BUILD_TYPE}" SQAV_BUILD_TYPE_LOWER)

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")

option(SYMPHONY_SQAV_USE_INSTALLED_DEPENDENCIES "Use exact installed SQAV dependencies" OFF)
if(SYMPHONY_SQAV_USE_INSTALLED_DEPENDENCIES)
    find_package(SymphonyKnowledgeVectorEngine 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqfvBatch 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqmvMetadata 0.1.0 EXACT REQUIRED CONFIG)
else()
    function(sqav_source_dependencies)
        set(BUILD_TESTING OFF)
        set(SYMPHONY_SQMV_USE_INSTALLED_DEPENDENCIES OFF)
        if(NOT TARGET Symphony::SqmvMetadata)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqmv-metadata-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqmv-metadata-cpp" EXCLUDE_FROM_ALL)
        endif()
    endfunction()
    sqav_source_dependencies()
endif()


set(SQAV_MODULE sqav-capture-cpp)
set(SQAV_VERSION "${PROJECT_VERSION}-dev")
set(SQAV_DOCS "share/doc/symphony/${SQAV_MODULE}/${SQAV_VERSION}")
set(SQAV_INCLUDE_ROOT "${CMAKE_INSTALL_INCLUDEDIR}/symphony/${SQAV_MODULE}/${SQAV_VERSION}")
set(SQAV_HEADER_DIR "${SQAV_INCLUDE_ROOT}/symphony/sqav")
set(SQAV_ARCHIVE_DIR "${CMAKE_INSTALL_LIBDIR}/symphony/${SQAV_MODULE}/${SQAV_VERSION}")
set(SQAV_CMAKE_DIR "${CMAKE_INSTALL_LIBDIR}/cmake/SymphonySqavCapture-${SQAV_VERSION}")
set(SQAV_LICENSE_DIR "share/licenses/symphony-${SQAV_MODULE}/${SQAV_VERSION}")
set(SQAV_RECEIPT "share/symphony/receipts/${SQAV_MODULE}/${SQAV_VERSION}/install-receipt.json")

# Apple ar and ranlib otherwise embed changing member and symbol-table timestamps.
if(APPLE)
    set(CMAKE_CXX_ARCHIVE_CREATE "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> qc <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_APPEND "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> q <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_FINISH "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_RANLIB> <TARGET>")
endif()

add_library(sqav-capture STATIC src/capture.cpp)
add_library(Symphony::SqavCapture ALIAS sqav-capture)
target_link_libraries(sqav-capture
    PUBLIC Symphony::SqmvMetadata Symphony::SqfvBatch
    PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(sqav-capture PROPERTIES
    EXPORT_NAME SqavCapture
    CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF
)
target_compile_features(sqav-capture PUBLIC cxx_std_26)
target_include_directories(sqav-capture PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${SQAV_INCLUDE_ROOT}>)
target_compile_options(sqav-capture PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

symphony_install_receipt_v2_preflight(RECEIPT_PATH "${SQAV_RECEIPT}")

set(SQAV_OWNED_FILES "")
set(SQAV_INSTALLED_FILES "")
macro(sqav_own path)
    list(APPEND SQAV_OWNED_FILES "${path}|regular")
    list(APPEND SQAV_INSTALLED_FILES "${path}")
endmacro()

install(TARGETS sqav-capture EXPORT SymphonySqavCaptureTargets
    ARCHIVE DESTINATION "${SQAV_ARCHIVE_DIR}")
sqav_own("${SQAV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqav-capture${CMAKE_STATIC_LIBRARY_SUFFIX}")
if(APPLE)
    # CMake invokes ranlib again during install; reseal before receipt hashing.
    install(CODE "execute_process(COMMAND \"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 \"${CMAKE_RANLIB}\" \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${SQAV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqav-capture${CMAKE_STATIC_LIBRARY_SUFFIX}\" COMMAND_ERROR_IS_FATAL ANY)")
endif()

install(FILES include/symphony/sqav/capture.hpp DESTINATION "${SQAV_HEADER_DIR}")
sqav_own("${SQAV_HEADER_DIR}/capture.hpp")

configure_package_config_file(cmake/SymphonySqavCaptureConfig.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqavCaptureConfig.cmake"
    INSTALL_DESTINATION "${SQAV_CMAKE_DIR}")
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqavCaptureConfigVersion.cmake"
    VERSION ${PROJECT_VERSION} COMPATIBILITY ExactVersion)
install(EXPORT SymphonySqavCaptureTargets
    FILE SymphonySqavCaptureTargets.cmake NAMESPACE Symphony::
    DESTINATION "${SQAV_CMAKE_DIR}")
install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqavCaptureConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqavCaptureConfigVersion.cmake"
    DESTINATION "${SQAV_CMAKE_DIR}")
foreach(name
    SymphonySqavCaptureConfig.cmake
    SymphonySqavCaptureConfigVersion.cmake
    SymphonySqavCaptureTargets.cmake
    "SymphonySqavCaptureTargets-${SQAV_BUILD_TYPE_LOWER}.cmake")
    sqav_own("${SQAV_CMAKE_DIR}/${name}")
endforeach()

foreach(doc INTENT.md MANIFEST.md SPEC.md SKILL.md INSTALL.md FEATURES.md)
    install(FILES "${CMAKE_CURRENT_SOURCE_DIR}/${doc}" DESTINATION "${SQAV_DOCS}")
    sqav_own("${SQAV_DOCS}/${doc}")
endforeach()
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/LICENSE"
    DESTINATION "${SQAV_LICENSE_DIR}" RENAME LICENSE-AGPL-3.0)
sqav_own("${SQAV_LICENSE_DIR}/LICENSE-AGPL-3.0")

symphony_install_receipt_v2(
    COMPONENT_ID "${SQAV_MODULE}" COMPONENT_KIND module MODULE_ID "${SQAV_MODULE}"
    VECTOR_ID sqav PACKAGE_ID "${SQAV_MODULE}" VERSION "${SQAV_VERSION}"
    RECEIPT_PATH "${SQAV_RECEIPT}" OWNED_FILES ${SQAV_OWNED_FILES})

configure_file(cmake/uninstall.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake" @ONLY)
add_custom_target(uninstall-${SQAV_MODULE}
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
        -P "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake")

if(BUILD_TESTING)
    if(SYMPHONY_SQAV_USE_INSTALLED_DEPENDENCIES)
        find_package(SymphonySqdvDelivery 0.1.0 EXACT REQUIRED CONFIG)
    else()
        function(sqav_test_composition)
            set(BUILD_TESTING OFF)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqdv-delivery-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqdv-delivery-cpp" EXCLUDE_FROM_ALL)
        endfunction()
        sqav_test_composition()
    endif()
    add_executable(sqav-capture-test tests/capture_test.cpp)
    target_link_libraries(sqav-capture-test PRIVATE
        Symphony::SqavCapture Symphony::KnowledgeVectorEngine Symphony::SqdvDelivery)
    set_target_properties(sqav-capture-test PROPERTIES
        CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqav-capture-test PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sqav-capture COMMAND sqav-capture-test)
    add_test(NAME sqav-package-lifecycle
        COMMAND "${CMAKE_COMMAND}"
            "-DPACKAGE_BINARY_DIR=${CMAKE_CURRENT_BINARY_DIR}"
            "-DPACKAGE_MODULE=sqav-capture-cpp"
            "-DPACKAGE_CMAKE_NAME=SymphonySqavCapture"
            -P "${CMAKE_CURRENT_SOURCE_DIR}/tests/package_lifecycle_test.cmake")
endif()
