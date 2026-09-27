cmake_minimum_required(VERSION 3.30)
project(SymphonySqavDatabentoDbn VERSION 0.2.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

if(CMAKE_CONFIGURATION_TYPES)
    message(FATAL_ERROR "sqav-databento-dbn-cpp requires a single-configuration generator for exact receipt-owned exports")
endif()
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()
string(TOLOWER "${CMAKE_BUILD_TYPE}" SQAV_DBN_BUILD_TYPE_LOWER)

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")

option(SYMPHONY_SQAV_DBN_USE_INSTALLED_DEPENDENCIES "Use exact installed SQAV_DBN dependencies" OFF)
if(SYMPHONY_SQAV_DBN_USE_INSTALLED_DEPENDENCIES)
    find_package(SymphonyKnowledgeVectorEngine 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqfvBatch 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqmvMetadata 0.1.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqavCapture 0.1.0 EXACT REQUIRED CONFIG)
else()
    function(sqav_dbn_source_dependencies)
        set(BUILD_TESTING OFF)
        set(SYMPHONY_SQAV_USE_INSTALLED_DEPENDENCIES OFF)
        if(NOT TARGET Symphony::SqavCapture)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqav-capture-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqav-capture-cpp" EXCLUDE_FROM_ALL)
        endif()
    endfunction()
    sqav_dbn_source_dependencies()
endif()


set(SQAV_DBN_MODULE sqav-databento-dbn-cpp)
set(SQAV_DBN_VERSION "${PROJECT_VERSION}-dev")
set(SQAV_DBN_DOCS "share/doc/symphony/${SQAV_DBN_MODULE}/${SQAV_DBN_VERSION}")
set(SQAV_DBN_INCLUDE_ROOT "${CMAKE_INSTALL_INCLUDEDIR}/symphony/${SQAV_DBN_MODULE}/${SQAV_DBN_VERSION}")
set(SQAV_DBN_HEADER_DIR "${SQAV_DBN_INCLUDE_ROOT}/symphony/sqav/databento")
set(SQAV_DBN_ARCHIVE_DIR "${CMAKE_INSTALL_LIBDIR}/symphony/${SQAV_DBN_MODULE}/${SQAV_DBN_VERSION}")
set(SQAV_DBN_CMAKE_DIR "${CMAKE_INSTALL_LIBDIR}/cmake/SymphonySqavDatabentoDbn-${SQAV_DBN_VERSION}")
set(SQAV_DBN_LICENSE_DIR "share/licenses/symphony-${SQAV_DBN_MODULE}/${SQAV_DBN_VERSION}")
set(SQAV_DBN_RECEIPT "share/symphony/receipts/${SQAV_DBN_MODULE}/${SQAV_DBN_VERSION}/install-receipt.json")

# Apple ar and ranlib otherwise embed changing member and symbol-table timestamps.
if(APPLE)
    set(CMAKE_CXX_ARCHIVE_CREATE "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> qc <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_APPEND "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> q <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_FINISH "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_RANLIB> <TARGET>")
endif()

add_library(sqav-databento-dbn STATIC src/dbn.cpp)
add_library(Symphony::SqavDatabentoDbn ALIAS sqav-databento-dbn)
target_link_libraries(sqav-databento-dbn
    PUBLIC Symphony::SqavCapture Symphony::SqmvMetadata Symphony::SqfvBatch
    PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(sqav-databento-dbn PROPERTIES
    EXPORT_NAME SqavDatabentoDbn
    CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF
)
target_compile_features(sqav-databento-dbn PUBLIC cxx_std_26)
target_include_directories(sqav-databento-dbn PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${SQAV_DBN_INCLUDE_ROOT}>)
target_compile_options(sqav-databento-dbn PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

symphony_install_receipt_v2_preflight(RECEIPT_PATH "${SQAV_DBN_RECEIPT}")

set(SQAV_DBN_OWNED_FILES "")
set(SQAV_DBN_INSTALLED_FILES "")
macro(sqav_dbn_own path)
    list(APPEND SQAV_DBN_OWNED_FILES "${path}|regular")
    list(APPEND SQAV_DBN_INSTALLED_FILES "${path}")
endmacro()

install(TARGETS sqav-databento-dbn EXPORT SymphonySqavDatabentoDbnTargets
    ARCHIVE DESTINATION "${SQAV_DBN_ARCHIVE_DIR}")
sqav_dbn_own("${SQAV_DBN_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqav-databento-dbn${CMAKE_STATIC_LIBRARY_SUFFIX}")
if(APPLE)
    # CMake invokes ranlib again during install; reseal before receipt hashing.
    install(CODE "execute_process(COMMAND \"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 \"${CMAKE_RANLIB}\" \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${SQAV_DBN_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqav-databento-dbn${CMAKE_STATIC_LIBRARY_SUFFIX}\" COMMAND_ERROR_IS_FATAL ANY)")
endif()

install(FILES include/symphony/sqav/databento/dbn.hpp DESTINATION "${SQAV_DBN_HEADER_DIR}")
sqav_dbn_own("${SQAV_DBN_HEADER_DIR}/dbn.hpp")

configure_package_config_file(cmake/SymphonySqavDatabentoDbnConfig.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqavDatabentoDbnConfig.cmake"
    INSTALL_DESTINATION "${SQAV_DBN_CMAKE_DIR}")
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqavDatabentoDbnConfigVersion.cmake"
    VERSION ${PROJECT_VERSION} COMPATIBILITY ExactVersion)
install(EXPORT SymphonySqavDatabentoDbnTargets
    FILE SymphonySqavDatabentoDbnTargets.cmake NAMESPACE Symphony::
    DESTINATION "${SQAV_DBN_CMAKE_DIR}")
install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqavDatabentoDbnConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqavDatabentoDbnConfigVersion.cmake"
    DESTINATION "${SQAV_DBN_CMAKE_DIR}")
foreach(name
    SymphonySqavDatabentoDbnConfig.cmake
    SymphonySqavDatabentoDbnConfigVersion.cmake
    SymphonySqavDatabentoDbnTargets.cmake
    "SymphonySqavDatabentoDbnTargets-${SQAV_DBN_BUILD_TYPE_LOWER}.cmake")
    sqav_dbn_own("${SQAV_DBN_CMAKE_DIR}/${name}")
endforeach()

foreach(doc INTENT.md MANIFEST.md SPEC.md SKILL.md INSTALL.md FEATURES.md)
    install(FILES "${CMAKE_CURRENT_SOURCE_DIR}/${doc}" DESTINATION "${SQAV_DBN_DOCS}")
    sqav_dbn_own("${SQAV_DBN_DOCS}/${doc}")
endforeach()
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/LICENSE"
    DESTINATION "${SQAV_DBN_LICENSE_DIR}" RENAME LICENSE-AGPL-3.0)
sqav_dbn_own("${SQAV_DBN_LICENSE_DIR}/LICENSE-AGPL-3.0")

symphony_install_receipt_v2(
    COMPONENT_ID "${SQAV_DBN_MODULE}" COMPONENT_KIND module MODULE_ID "${SQAV_DBN_MODULE}"
    VECTOR_ID sqav PACKAGE_ID "${SQAV_DBN_MODULE}" VERSION "${SQAV_DBN_VERSION}"
    RECEIPT_PATH "${SQAV_DBN_RECEIPT}" OWNED_FILES ${SQAV_DBN_OWNED_FILES})

configure_file(cmake/uninstall.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake" @ONLY)
add_custom_target(uninstall-${SQAV_DBN_MODULE}
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
        -P "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake")

if(BUILD_TESTING)
    if(SYMPHONY_SQAV_DBN_USE_INSTALLED_DEPENDENCIES)
        find_package(SymphonySqdvDelivery 0.1.0 EXACT REQUIRED CONFIG)
    else()
        function(sqav_dbn_test_composition)
            set(BUILD_TESTING OFF)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqdv-delivery-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqdv-delivery-cpp" EXCLUDE_FROM_ALL)
        endfunction()
        sqav_dbn_test_composition()
    endif()
    add_executable(sqav-databento-dbn-test tests/dbn_test.cpp)
    target_link_libraries(sqav-databento-dbn-test PRIVATE
        Symphony::SqavDatabentoDbn Symphony::KnowledgeVectorEngine Symphony::SqdvDelivery)
    set_target_properties(sqav-databento-dbn-test PROPERTIES
        CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqav-databento-dbn-test PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sqav-databento-dbn COMMAND sqav-databento-dbn-test)
    add_test(NAME sqav-dbn-package-lifecycle
        COMMAND "${CMAKE_COMMAND}"
            "-DPACKAGE_BINARY_DIR=${CMAKE_CURRENT_BINARY_DIR}"
            "-DPACKAGE_MODULE=sqav-databento-dbn-cpp"
            "-DPACKAGE_CMAKE_NAME=SymphonySqavDatabentoDbn"
            -P "${CMAKE_CURRENT_SOURCE_DIR}/tests/package_lifecycle_test.cmake")
endif()
