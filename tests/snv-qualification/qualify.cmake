cmake_minimum_required(VERSION 3.30)
include("${MVP_REPOSITORY}/tests/mvp-dependencies/common.cmake")
file(READ "${MVP_MAP}" MVP_MAP_JSON)
set(components libraries/knowledge-vector-engine-cpp libraries/snv-common-cpp
  modules/sniv-engine modules/snrv-engine modules/sciv-engine modules/scnv-engine
  modules/snv-engine modules/snv-local-observer)
set(dependencies "[]")
foreach(path IN LISTS components)
  mvp_component("${path}" component)
  string(JSON id GET "${component}" component_id)
  file(WRITE "${MVP_WORK}/metadata/${id}.json" "${component}\n")
  mvp_run("${id}-receipt" "${MVP_WORK}" "${CMAKE_COMMAND}"
    "-DMVP_REPOSITORY=${MVP_REPOSITORY}" "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/${id}.json"
    "-DMVP_SELECTED_PREFIX=${MVP_SELECTED_PREFIX}"
    -P "${MVP_REPOSITORY}/tests/mvp-dependencies/verify-receipt.cmake")
  mvp_indices(exports "${component}" exported_packages)
  foreach(index IN LISTS exports)
    mvp_indices(targets "${component}" exported_packages ${index} targets)
    foreach(target IN LISTS targets)
      string(JSON name GET "${component}" exported_packages ${index} targets ${target})
      string(JSON length LENGTH "${dependencies}")
      string(JSON dependencies SET "${dependencies}" ${length} "{\"target\":\"${name}\",\"scope\":\"runtime\"}")
    endforeach()
  endforeach()
endforeach()
file(COPY "${CMAKE_CURRENT_LIST_DIR}/sdk-consumer/" DESTINATION "${MVP_WORK}/consumer")
set(metadata "{\"project_version\":null,\"artifact_targets\":[\"snv-installed-qualification\"],\"first_party_sdk_dependencies\":${dependencies}}")
file(WRITE "${MVP_WORK}/metadata/consumer.json" "${metadata}\n")
mvp_run(sdk-configure "${MVP_WORK}" "${CMAKE_COMMAND}" -S "${MVP_WORK}/consumer" -B "${MVP_WORK}/consumer/build"
  "-DCMAKE_PREFIX_PATH=${MVP_SELECTED_PREFIX}" -DCMAKE_BUILD_TYPE=Release -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DBUILD_TESTING=OFF
  -DCMAKE_FIND_USE_PACKAGE_REGISTRY=OFF -DCMAKE_FIND_USE_SYSTEM_PACKAGE_REGISTRY=OFF
  "-DCMAKE_PROJECT_TOP_LEVEL_INCLUDES=${MVP_REPOSITORY}/tests/mvp-dependencies/prove-imports.cmake"
  "-DMVP_REPOSITORY=${MVP_REPOSITORY}" "-DMVP_SELECTED_PREFIX=${MVP_SELECTED_PREFIX}"
  "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/consumer.json" "-DMVP_PROOF_LOG=${MVP_WORK}/logs/sdk-imports.log")
mvp_run(sdk-build "${MVP_WORK}" "${CMAKE_COMMAND}" --build "${MVP_WORK}/consumer/build" --parallel 2)
mvp_run(sdk-replay "${MVP_WORK}/tmp" "${MVP_WORK}/consumer/build/snv-installed-qualification"
  "${MVP_REPOSITORY}/modules/snv-engine/tests/fixtures/snv_inspect.json")
mvp_run(harness-boundaries "${MVP_WORK}/tmp" go test -race -count=1 -v
  "${CMAKE_CURRENT_LIST_DIR}/qualification.go" "${CMAKE_CURRENT_LIST_DIR}/qualification_test.go")
mvp_run(supplied-evidence-and-native-observer "${MVP_WORK}/tmp" go run
  "${CMAKE_CURRENT_LIST_DIR}/qualification.go" --repo "${MVP_REPOSITORY}" --prefix "${MVP_SELECTED_PREFIX}"
  --work "${MVP_WORK}" --qxctl "${SNV_QXCTL}")
# Retain the already admitted installed boundary/adversarial gates, fail closed
# if any selected test is skipped, and do not turn missing fixtures into coverage.
mvp_run(installed-go "${MVP_REPOSITORY}" go run "${MVP_REPOSITORY}/tests/release-qualification/go-runner.go"
  --repo "${MVP_REPOSITORY}" --manifest "${MVP_REPOSITORY}/tests/release-qualification/manifest.json"
  --work "${MVP_WORK}/logs/installed-go" --mode installed --group knowledge --prefix "${MVP_SELECTED_PREFIX}")
# Verify the caller's package bytes again after all consumers and tamper fixtures.
foreach(path IN LISTS components)
  mvp_component("${path}" component)
  string(JSON id GET "${component}" component_id)
  mvp_run("${id}-preserved" "${MVP_WORK}" "${CMAKE_COMMAND}"
    "-DMVP_REPOSITORY=${MVP_REPOSITORY}" "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/${id}.json"
    "-DMVP_SELECTED_PREFIX=${MVP_SELECTED_PREFIX}" -P "${MVP_REPOSITORY}/tests/mvp-dependencies/verify-receipt.cmake")
endforeach()
file(READ "${MVP_WORK}/evidence.pending.json" evidence)
string(JSON evidence SET "${evidence}" complete true)
file(GLOB_RECURSE completed RELATIVE "${MVP_WORK}" "${MVP_WORK}/logs/*")
set(logs "[]")
foreach(step IN LISTS completed)
  file(SHA256 "${MVP_WORK}/${step}" hash)
  string(JSON count LENGTH "${logs}")
  string(JSON logs SET "${logs}" ${count} "{\"path\":\"${step}\",\"sha256\":\"${hash}\"}")
endforeach()
string(JSON evidence SET "${evidence}" retained_log_sha256 "${logs}")
file(WRITE "${MVP_WORK}/evidence.json" "${evidence}\n")
message(STATUS "PASS SNV supplied-evidence and separately recorded observer qualification")
