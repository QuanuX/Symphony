cmake_minimum_required(VERSION 3.30)
project(SymphonyPackageLifecycleFixture LANGUAGES C)
include("${SYMPHONY_REPOSITORY}/cmake/SymphonyInstallReceiptV2.cmake")
if(NOT FIXTURE_VERSION MATCHES "^[12]\\.0\\.0$")
  message(FATAL_ERROR "Select an exact fixture version")
endif()
set(directory "libexec/symphony/package-fixture/${FIXTURE_VERSION}")
set(receipt "share/symphony/receipts/package-fixture/${FIXTURE_VERSION}/install-receipt.json")
add_library(payload SHARED payload.c)
add_executable(probe probe.c)
target_link_libraries(probe PRIVATE payload)
if(APPLE)
  set_target_properties(probe PROPERTIES INSTALL_RPATH "@loader_path")
else()
  set_target_properties(probe PROPERTIES INSTALL_RPATH "$ORIGIN")
endif()
set(library "${CMAKE_SHARED_LIBRARY_PREFIX}payload${CMAKE_SHARED_LIBRARY_SUFFIX}")
symphony_install_receipt_v2_preflight(RECEIPT_PATH "${receipt}")
install(TARGETS payload probe RUNTIME DESTINATION "${directory}" LIBRARY DESTINATION "${directory}")
symphony_install_receipt_v2(
  COMPONENT_ID package-fixture COMPONENT_KIND module MODULE_ID package-fixture
  PACKAGE_ID package-fixture VERSION "${FIXTURE_VERSION}" RECEIPT_PATH "${receipt}"
  OWNED_FILES "${directory}/probe|executable" "${directory}/${library}|regular"
  ENTRY_POINTS "probe|executable|${directory}/probe|symphony.package-fixture.v1")
file(WRITE "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake"
  "set(_files [==[${directory}/probe;${directory}/${library};${receipt}]==])\ninclude([==[${CMAKE_CURRENT_BINARY_DIR}/SymphonyUninstallReceiptV2.cmake]==])\n")
