cmake_minimum_required(VERSION 3.30)
project(SymphonySqdvDelivery VERSION 0.3.0 LANGUAGES CXX)

include(CTest)
include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

if(CMAKE_CONFIGURATION_TYPES)
    message(FATAL_ERROR "sqdv-delivery-cpp requires a single-configuration generator for exact receipt-owned exports")
endif()
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()
string(TOLOWER "${CMAKE_BUILD_TYPE}" SQDV_BUILD_TYPE_LOWER)

get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")

option(SYMPHONY_SQDV_USE_INSTALLED_DEPENDENCIES "Use exact installed SQDV dependencies" OFF)
if(SYMPHONY_SQDV_USE_INSTALLED_DEPENDENCIES)
    find_package(SymphonyKnowledgeVectorEngine 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqfvBatch 0.3.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqmvMetadata 0.2.0 EXACT REQUIRED CONFIG)
    find_package(SymphonySqpvLocalStore 0.2.0 EXACT REQUIRED CONFIG)
else()
    function(sqdv_source_dependencies)
        set(BUILD_TESTING OFF)
        set(SYMPHONY_SQPV_USE_INSTALLED_DEPENDENCIES OFF)
        if(NOT TARGET Symphony::SqpvLocalStore)
            add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/sqpv-local-store-cpp"
                "${CMAKE_CURRENT_BINARY_DIR}/sqpv-local-store-cpp" EXCLUDE_FROM_ALL)
        endif()
    endfunction()
    sqdv_source_dependencies()
endif()

set(SQDV_MODULE sqdv-delivery-cpp)
set(SQDV_VERSION "${PROJECT_VERSION}-dev")
set(SQDV_DOCS "share/doc/symphony/${SQDV_MODULE}/${SQDV_VERSION}")
set(SQDV_INCLUDE_ROOT "${CMAKE_INSTALL_INCLUDEDIR}/symphony/${SQDV_MODULE}/${SQDV_VERSION}")
set(SQDV_HEADER_DIR "${SQDV_INCLUDE_ROOT}/symphony/sqdv")
set(SQDV_ARCHIVE_DIR "${CMAKE_INSTALL_LIBDIR}/symphony/${SQDV_MODULE}/${SQDV_VERSION}")
set(SQDV_CMAKE_DIR "${CMAKE_INSTALL_LIBDIR}/cmake/SymphonySqdvDelivery-${SQDV_VERSION}")
set(SQDV_LICENSE_DIR "share/licenses/symphony-${SQDV_MODULE}/${SQDV_VERSION}")
set(SQDV_RECEIPT "share/symphony/receipts/${SQDV_MODULE}/${SQDV_VERSION}/install-receipt.json")

# Apple ar and ranlib otherwise embed changing member and symbol-table timestamps.
if(APPLE)
    set(CMAKE_CXX_ARCHIVE_CREATE "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> qc <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_APPEND "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_AR> q <TARGET> <LINK_FLAGS> <OBJECTS>")
    set(CMAKE_CXX_ARCHIVE_FINISH "\"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 <CMAKE_RANLIB> <TARGET>")
endif()

add_library(sqdv-delivery STATIC src/delivery.cpp src/checkpoint.cpp)
add_library(Symphony::SqdvDelivery ALIAS sqdv-delivery)
target_link_libraries(sqdv-delivery
    PUBLIC Symphony::SqpvLocalStore Symphony::SqmvMetadata Symphony::SqfvBatch
    PRIVATE Symphony::KnowledgeVectorEngine)
set_target_properties(sqdv-delivery PROPERTIES
    EXPORT_NAME SqdvDelivery
    CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF
)
target_compile_features(sqdv-delivery PUBLIC cxx_std_26)
target_include_directories(sqdv-delivery PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${SQDV_INCLUDE_ROOT}>)
target_compile_options(sqdv-delivery PRIVATE
    $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

symphony_install_receipt_v2_preflight(RECEIPT_PATH "${SQDV_RECEIPT}")

set(SQDV_OWNED_FILES "")
set(SQDV_INSTALLED_FILES "")
macro(sqdv_own path)
    list(APPEND SQDV_OWNED_FILES "${path}|regular")
    list(APPEND SQDV_INSTALLED_FILES "${path}")
endmacro()

install(TARGETS sqdv-delivery EXPORT SymphonySqdvDeliveryTargets
    ARCHIVE DESTINATION "${SQDV_ARCHIVE_DIR}")
sqdv_own("${SQDV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqdv-delivery${CMAKE_STATIC_LIBRARY_SUFFIX}")
if(APPLE)
    # CMake invokes ranlib again during install; reseal before receipt hashing.
    install(CODE "execute_process(COMMAND \"${CMAKE_COMMAND}\" -E env ZERO_AR_DATE=1 \"${CMAKE_RANLIB}\" \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${SQDV_ARCHIVE_DIR}/${CMAKE_STATIC_LIBRARY_PREFIX}sqdv-delivery${CMAKE_STATIC_LIBRARY_SUFFIX}\" COMMAND_ERROR_IS_FATAL ANY)")
endif()

install(FILES include/symphony/sqdv/delivery.hpp DESTINATION "${SQDV_HEADER_DIR}")
sqdv_own("${SQDV_HEADER_DIR}/delivery.hpp")
install(FILES include/symphony/sqdv/checkpoint.hpp DESTINATION "${SQDV_HEADER_DIR}")
sqdv_own("${SQDV_HEADER_DIR}/checkpoint.hpp")

configure_package_config_file(cmake/SymphonySqdvDeliveryConfig.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqdvDeliveryConfig.cmake"
    INSTALL_DESTINATION "${SQDV_CMAKE_DIR}")
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqdvDeliveryConfigVersion.cmake"
    VERSION ${PROJECT_VERSION} COMPATIBILITY ExactVersion)
install(EXPORT SymphonySqdvDeliveryTargets
    FILE SymphonySqdvDeliveryTargets.cmake NAMESPACE Symphony::
    DESTINATION "${SQDV_CMAKE_DIR}")
install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqdvDeliveryConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/SymphonySqdvDeliveryConfigVersion.cmake"
    DESTINATION "${SQDV_CMAKE_DIR}")
foreach(name
    SymphonySqdvDeliveryConfig.cmake
    SymphonySqdvDeliveryConfigVersion.cmake
    SymphonySqdvDeliveryTargets.cmake
    "SymphonySqdvDeliveryTargets-${SQDV_BUILD_TYPE_LOWER}.cmake")
    sqdv_own("${SQDV_CMAKE_DIR}/${name}")
endforeach()

foreach(doc INTENT.md MANIFEST.md SPEC.md SKILL.md INSTALL.md FEATURES.md)
    install(FILES "${CMAKE_CURRENT_SOURCE_DIR}/${doc}" DESTINATION "${SQDV_DOCS}")
    sqdv_own("${SQDV_DOCS}/${doc}")
endforeach()
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/LICENSE"
    DESTINATION "${SQDV_LICENSE_DIR}" RENAME LICENSE-AGPL-3.0)
sqdv_own("${SQDV_LICENSE_DIR}/LICENSE-AGPL-3.0")

symphony_install_receipt_v2(
    COMPONENT_ID "${SQDV_MODULE}" COMPONENT_KIND module MODULE_ID "${SQDV_MODULE}"
    VECTOR_ID sqdv PACKAGE_ID "${SQDV_MODULE}" VERSION "${SQDV_VERSION}"
    RECEIPT_PATH "${SQDV_RECEIPT}" OWNED_FILES ${SQDV_OWNED_FILES})

configure_file(cmake/uninstall.cmake.in
    "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake" @ONLY)
add_custom_target(uninstall-${SQDV_MODULE}
    COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
        -P "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake")

if(BUILD_TESTING)
    add_executable(sqdv-delivery-test tests/delivery_test.cpp)
    target_link_libraries(sqdv-delivery-test PRIVATE
        Symphony::SqdvDelivery Symphony::KnowledgeVectorEngine)
    set_target_properties(sqdv-delivery-test PROPERTIES
        CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqdv-delivery-test PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
    add_test(NAME sqdv-delivery COMMAND sqdv-delivery-test)
    add_executable(sqdv-checkpoint-test tests/checkpoint_test.cpp)
    target_link_libraries(sqdv-checkpoint-test PRIVATE Symphony::SqdvDelivery)
    set_target_properties(sqdv-checkpoint-test PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(sqdv-checkpoint-test PRIVATE -Wall -Wextra -Wpedantic -Werror)
    add_test(NAME sqdv-checkpoint COMMAND sqdv-checkpoint-test)
    add_test(NAME sqdv-package-lifecycle
        COMMAND "${CMAKE_COMMAND}"
            "-DPACKAGE_BINARY_DIR=${CMAKE_CURRENT_BINARY_DIR}"
            "-DPACKAGE_MODULE=sqdv-delivery-cpp"
            "-DPACKAGE_CMAKE_NAME=SymphonySqdvDelivery"
            -P "${CMAKE_CURRENT_SOURCE_DIR}/tests/package_lifecycle_test.cmake")
endif()
