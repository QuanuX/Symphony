cmake_minimum_required(VERSION 3.30)
project(SymphonySqtvIntegerConversion VERSION 0.2.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

if(CMAKE_CONFIGURATION_TYPES)
    message(FATAL_ERROR "sqtv-integer-conversion-cpp requires a single-configuration generator for exact receipt-owned exports")
endif()
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()
string(TOLOWER "${CMAKE_BUILD_TYPE}" SQTV_BUILD_TYPE_LOWER)

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")

option(SYMPHONY_SQTV_USE_INSTALLED_DEPENDENCIES "Use exact installed SQTV dependencies" OFF)
if(SYMPHONY_SQTV_USE_INSTALLED_DEPENDENCIES)
    find_package(SymphonyKnowledgeVectorEngine 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqfvBatch 0.3.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqmvMetadata 0.2.0 EXACT REQUIRED CONFIG)
else()
    function(sqtv_source_dependencies)
        set(BUILD_TESTING OFF)
        set(SYMPHONY_SQMV_USE_INSTALLED_DEPENDENCIES OFF)
        if(NOT TARGET Symphony::SqmvMetadata)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqmv-metadata-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqmv-metadata-cpp" EXCLUDE_FROM_ALL)
        endif()
    endfunction()
    sqtv_source_dependencies()
endif()


set(SQTV_MODULE sqtv-integer-conversion-cpp)
set(SQTV_VERSION "${PROJECT_VERSION}-dev")
set(SQTV_DOCS "share/doc/symphony/${SQTV_MODULE}/${SQTV_VERSION}")
set(SQTV_INCLUDE_ROOT "${CMAKE_INSTALL_INCLUDEDIR}/symphony/${SQTV_MODULE}/${SQTV_VERSION}")
set(SQTV_HEADER_DIR "${SQTV_INCLUDE_ROOT}/symphony/sqtv")
set(SQTV_ARCHIVE_DIR "${CMAKE_INSTALL_LIBDIR}/symphony/${SQTV_MODULE}/${SQTV_VERSION}")
set(SQTV_CMAKE_DIR "${CMAKE_INSTALL_LIBDIR}/cmake/SymphonySqtvIntegerConversion-${SQTV_VERSION}")
set(SQTV_LICENSE_DIR "share/licenses/symphony-${SQTV_MODULE}/${SQTV_VERSION}")
set(SQTV_RECEIPT "share/symphony/receipts/${SQTV_MODULE}/${SQTV_VERSION}/install-receipt.json")

# Apple ar and ranlib otherwise embed changing member and symbol-table timestamps.
if(APPLE)
    set(CMAKE_CXX_ARCHIVE_CREATE "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> qc <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_APPEND "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> q <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_FINISH "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_RANLIB> <TARGET>")
endif()

add_library(sqtv-integer-conversion STATIC src/integer_conversion.cpp)
add_library(Symphony::SqtvIntegerConversion ALIAS sqtv-integer-conversion)
target_link_libraries(sqtv-integer-conversion
    PUBLIC Symphony::SqmvMetadata Symphony::SqfvBatch
    PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(sqtv-integer-conversion PROPERTIES
    EXPORT_NAME SqtvIntegerConversion
    CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF
)
target_compile_features(sqtv-integer-conversion PUBLIC cxx_std_26)
target_include_directories(sqtv-integer-conversion PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${SQTV_INCLUDE_ROOT}>)
target_compile_options(sqtv-integer-conversion PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

symphony_install_receipt_v2_preflight(RECEIPT_PATH "${SQTV_RECEIPT}")

set(SQTV_OWNED_FILES "")
set(SQTV_INSTALLED_FILES "")
macro(sqtv_own path)
    list(APPEND SQTV_OWNED_FILES "${path}|regular")
    list(APPEND SQTV_INSTALLED_FILES "${path}")
endmacro()

install(TARGETS sqtv-integer-conversion EXPORT SymphonySqtvIntegerConversionTargets
    ARCHIVE DESTINATION "${SQTV_ARCHIVE_DIR}")
sqtv_own("${SQTV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqtv-integer-conversion${CMAKE_STATIC_LIBRARY_SUFFIX}")
if(APPLE)
    # CMake invokes ranlib again during install; reseal before receipt hashing.
    install(CODE "execute_process(COMMAND \"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 \"${CMAKE_RANLIB}\" \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${SQTV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqtv-integer-conversion${CMAKE_STATIC_LIBRARY_SUFFIX}\" COMMAND_ERROR_IS_FATAL ANY)")
endif()

install(FILES include/symphony/sqtv/integer_conversion.hpp DESTINATION "${SQTV_HEADER_DIR}")
sqtv_own("${SQTV_HEADER_DIR}/integer_conversion.hpp")

configure_package_config_file(cmake/SymphonySqtvIntegerConversionConfig.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqtvIntegerConversionConfig.cmake"
    INSTALL_DESTINATION "${SQTV_CMAKE_DIR}")
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqtvIntegerConversionConfigVersion.cmake"
    VERSION ${PROJECT_VERSION} COMPATIBILITY ExactVersion)
install(EXPORT SymphonySqtvIntegerConversionTargets
    FILE SymphonySqtvIntegerConversionTargets.cmake NAMESPACE Symphony::
    DESTINATION "${SQTV_CMAKE_DIR}")
install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqtvIntegerConversionConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqtvIntegerConversionConfigVersion.cmake"
    DESTINATION "${SQTV_CMAKE_DIR}")
foreach(name
    SymphonySqtvIntegerConversionConfig.cmake
    SymphonySqtvIntegerConversionConfigVersion.cmake
    SymphonySqtvIntegerConversionTargets.cmake
    "SymphonySqtvIntegerConversionTargets-${SQTV_BUILD_TYPE_LOWER}.cmake")
    sqtv_own("${SQTV_CMAKE_DIR}/${name}")
endforeach()

foreach(doc INTENT.md MANIFEST.md SPEC.md SKILL.md INSTALL.md FEATURES.md)
    install(FILES "${CMAKE_CURRENT_SOURCE_DIR}/${doc}" DESTINATION "${SQTV_DOCS}")
    sqtv_own("${SQTV_DOCS}/${doc}")
endforeach()
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/LICENSE"
    DESTINATION "${SQTV_LICENSE_DIR}" RENAME LICENSE-AGPL-3.0)
sqtv_own("${SQTV_LICENSE_DIR}/LICENSE-AGPL-3.0")

symphony_install_receipt_v2(
    COMPONENT_ID "${SQTV_MODULE}" COMPONENT_KIND module MODULE_ID "${SQTV_MODULE}"
    VECTOR_ID sqtv PACKAGE_ID "${SQTV_MODULE}" VERSION "${SQTV_VERSION}"
    RECEIPT_PATH "${SQTV_RECEIPT}" OWNED_FILES ${SQTV_OWNED_FILES})

configure_file(cmake/uninstall.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake" @ONLY)
add_custom_target(uninstall-${SQTV_MODULE}
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
        -P "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake")

if(BUILD_TESTING)
    if(SYMPHONY_SQTV_USE_INSTALLED_DEPENDENCIES)
        find_package(SymphonySqdvDelivery 0.2.0 EXACT REQUIRED CONFIG)
    else()
        function(sqtv_test_composition)
            set(BUILD_TESTING OFF)
            if(NOT TARGET Symphony::SqdvDelivery)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqdv-delivery-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqdv-delivery-cpp" EXCLUDE_FROM_ALL)
            endif()
        endfunction()
        sqtv_test_composition()
    endif()
    add_executable(sqtv-integer-conversion-test tests/integer_conversion_test.cpp)
    target_link_libraries(sqtv-integer-conversion-test PRIVATE
        Symphony::SqtvIntegerConversion Symphony::KnowledgeVectorEngine Symphony::SqdvDelivery)
    set_target_properties(sqtv-integer-conversion-test PROPERTIES
        CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqtv-integer-conversion-test PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sqtv-integer-conversion COMMAND sqtv-integer-conversion-test)
    add_test(NAME sqtv-package-lifecycle
        COMMAND "${CMAKE_COMMAND}"
            "-DPACKAGE_BINARY_DIR=${CMAKE_CURRENT_BINARY_DIR}"
            "-DPACKAGE_MODULE=sqtv-integer-conversion-cpp"
            "-DPACKAGE_CMAKE_NAME=SymphonySqtvIntegerConversion"
            -P "${CMAKE_CURRENT_SOURCE_DIR}/tests/package_lifecycle_test.cmake")
endif()
