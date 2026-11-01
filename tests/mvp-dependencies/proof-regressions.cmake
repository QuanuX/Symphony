include("${CMAKE_CURRENT_LIST_DIR}/common.cmake")
if(NOT DEFINED MVP_PROOF_TEST_WORK OR NOT IS_ABSOLUTE "${MVP_PROOF_TEST_WORK}" OR
   EXISTS "${MVP_PROOF_TEST_WORK}")
  message(FATAL_ERROR "Supply a fresh absolute MVP_PROOF_TEST_WORK")
endif()
file(MAKE_DIRECTORY "${MVP_PROOF_TEST_WORK}/source" "${MVP_PROOF_TEST_WORK}/prefix/lib")
set(prefix "${MVP_PROOF_TEST_WORK}/prefix")
foreach(version 0.1.0 0.2.0)
  set(config "${prefix}/lib/cmake/SymphonyProofCollision-${version}")
  file(MAKE_DIRECTORY "${config}")
  # Configure-only fixture: the imported archive is never linked or executed.
  file(WRITE "${prefix}/lib/fixture-${version}.a" "configure-only fixture\n")
  file(WRITE "${config}/SymphonyProofCollisionConfig.cmake"
    "if(NOT TARGET Symphony::ProofCollision)\n  add_library(Symphony::ProofCollision STATIC IMPORTED)\n  set_target_properties(Symphony::ProofCollision PROPERTIES IMPORTED_LOCATION \"${prefix}/lib/fixture-${version}.a\")\nendif()\n")
  file(WRITE "${config}/SymphonyProofCollisionConfigVersion.cmake"
    "set(PACKAGE_VERSION ${version})\nif(PACKAGE_FIND_VERSION VERSION_EQUAL PACKAGE_VERSION)\n  set(PACKAGE_VERSION_EXACT TRUE)\n  set(PACKAGE_VERSION_COMPATIBLE TRUE)\nelse()\n  set(PACKAGE_VERSION_COMPATIBLE FALSE)\nendif()\n")
endforeach()
file(WRITE "${MVP_PROOF_TEST_WORK}/map.json" [=[
{"native_components":[{"exported_packages":[{"package":"SymphonyProofCollision","version":"0.2.0","receipt_release":"0.2.0-dev","targets":["Symphony::ProofCollision"],"cmake_version_file":true}]}]}
]=])
file(WRITE "${MVP_PROOF_TEST_WORK}/component.json" [=[
{"project_version":"0.1.0","artifact_targets":[],"first_party_sdk_dependencies":[{"target":"Symphony::ProofCollision","scope":"runtime"}]}
]=])
file(WRITE "${MVP_PROOF_TEST_WORK}/source/CMakeLists.txt" [=[
cmake_minimum_required(VERSION 3.30)
project(SymphonyProofCollision VERSION 0.1.0 LANGUAGES NONE)
function(select_sdk version)
  find_package(SymphonyProofCollision ${version} EXACT REQUIRED CONFIG)
endfunction()
if(CASE STREQUAL "project-collision")
  select_sdk(0.2.0)
elseif(CASE STREQUAL "wrong-scoped-package")
  select_sdk(0.1.0)
else()
  find_package(SymphonyProofCollision 0.1.0 EXACT REQUIRED CONFIG)
endif()
]=])
foreach(case project-collision wrong-scoped-package wrong-visible-package)
  execute_process(COMMAND "${CMAKE_COMMAND}"
    -S "${MVP_PROOF_TEST_WORK}/source" -B "${MVP_PROOF_TEST_WORK}/${case}"
    "-DCASE=${case}" -DCMAKE_BUILD_TYPE=Release -DCMAKE_POSITION_INDEPENDENT_CODE=ON
    "-DCMAKE_PREFIX_PATH=${prefix}" "-DMVP_SELECTED_PREFIX=${prefix}"
    "-DMVP_MAP=${MVP_PROOF_TEST_WORK}/map.json"
    "-DMVP_COMPONENT_FILE=${MVP_PROOF_TEST_WORK}/component.json"
    "-DMVP_PROOF_LOG=${MVP_PROOF_TEST_WORK}/${case}-imports.log"
    "-DCMAKE_PROJECT_TOP_LEVEL_INCLUDES=${CMAKE_CURRENT_LIST_DIR}/prove-imports.cmake"
    RESULT_VARIABLE status OUTPUT_VARIABLE output ERROR_VARIABLE error)
  file(WRITE "${MVP_PROOF_TEST_WORK}/${case}.log" "${output}${error}")
  if(case STREQUAL "project-collision")
    if(NOT status EQUAL 0)
      message(FATAL_ERROR "Project/package name collision was rejected: ${output}${error}")
    endif()
  elseif(status EQUAL 0)
    message(FATAL_ERROR "Wrong original package version was admitted: ${case}")
  elseif(NOT "${output}${error}" MATCHES "(Originally selected SDK version differs|Could not find a configuration file for package)")
    message(FATAL_ERROR "Wrong-package fixture failed for an unexpected reason: ${output}${error}")
  endif()
endforeach()
message(STATUS "PASS package/project scope collision and two wrong-package refusals")
