mvp_run(duckdb-dependency-refusals "${MVP_WORK}" "${CMAKE_COMMAND}"
  "-DDUCKDB_REPOSITORY=${MVP_REPOSITORY}" "-DDUCKDB_NEGATIVE_WORK=${MVP_WORK}/dependency-refusals"
  "-DDUCKDB_CONFIGURED_BUILD=${MVP_WORK}/build/scv-graph-duckdb-connector"
  -P "${MVP_REPOSITORY}/tests/duckdb-connectors/dependency-refusals.cmake")
# Invoked only after this group's installed packages and native suites succeed.
foreach(domain scv shv)
  set(id "${domain}-graph-duckdb-connector")
  mvp_run("${id}-fault-build" "${MVP_WORK}" "${CMAKE_COMMAND}" --build "${MVP_WORK}/build/${id}"
    --target "${domain}-graph-duckdb-commit-fault-test" --parallel "${MVP_JOBS}")
endforeach()
set(copy "${MVP_WORK}/consumers/duckdb")
file(COPY "${MVP_REPOSITORY}/tests/duckdb-connectors/CMakeLists.txt"
  "${MVP_REPOSITORY}/tests/duckdb-connectors/recovery.cpp"
  "${MVP_REPOSITORY}/modules/scv-graph-duckdb-connector/tests/connector_test_support.hpp"
  "${MVP_REPOSITORY}/libraries/knowledge-vector-engine-cpp/tests/support/native_test.hpp" DESTINATION "${copy}")
file(COPY "${MVP_REPOSITORY}/modules/scv-graph-duckdb-connector/tests/fixtures" DESTINATION "${copy}")
file(COPY_FILE "${MVP_REPOSITORY}/modules/scv-graph-duckdb-connector/tests/connector_test.cpp" "${copy}/scv-connector-test.cpp")
file(COPY_FILE "${MVP_REPOSITORY}/modules/shv-graph-duckdb-connector/tests/connector_test.cpp" "${copy}/shv-connector-test.cpp")
file(WRITE "${MVP_WORK}/metadata/duckdb-consumer.json" [=[
{"project_version":null,"artifact_targets":["scv-process","shv-process","recovery"],"first_party_sdk_dependencies":[{"target":"Symphony::KnowledgeVectorEngine","scope":"runtime"}]}
]=])
mvp_run(duckdb-consumer-configure "${MVP_WORK}" "${CMAKE_COMMAND}" -S "${copy}" -B "${copy}/build"
  ${base_arguments} -DBUILD_TESTING=ON "-DDUCKDB_SDK_PREFIX=${MVP_SELECTED_PREFIX}"
  "-DDUCKDB_INSTALL_PREFIX=${MVP_SELECTED_PREFIX}" "-DDUCKDB_BUILD_ROOT=${MVP_WORK}/build"
  "-DDUCKDB_RETAINED_ROOT=${MVP_WORK}/caller-databases"
  "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/duckdb-consumer.json"
  "-DMVP_PROOF_LOG=${MVP_WORK}/logs/duckdb-consumer-imports.log")
mvp_run(duckdb-consumer-build "${MVP_WORK}" "${CMAKE_COMMAND}" --build "${copy}/build" --parallel "${MVP_JOBS}")
mvp_run(duckdb-consumer-tests "${MVP_WORK}/tmp" "${MVP_CTEST}" --test-dir "${copy}/build"
  -V --no-tests=error --output-junit "${MVP_WORK}/logs/duckdb-consumer-tests.xml")
mvp_run(duckdb-consumer-results "${MVP_WORK}" "${CMAKE_COMMAND}"
  "-DRESULT_JUNIT=${MVP_WORK}/logs/duckdb-consumer-tests.xml"
  "-DRESULT_LOG=${MVP_WORK}/logs/duckdb-consumer-tests.log" -DEXPECTED_TEST_COUNT=4
  -P "${MVP_REPOSITORY}/tests/release-qualification/check-results.cmake")
mvp_run(duckdb-qxctl-admission "${MVP_WORK}" sh
  "${MVP_REPOSITORY}/tests/duckdb-connectors/qxctl-admission.sh"
  "${MVP_SELECTED_PREFIX}" "${MVP_WORK}/qxctl-admission")
mvp_run(duckdb-loader-removal "${MVP_WORK}" "${CMAKE_COMMAND}"
  "-DMVP_REPOSITORY=${MVP_REPOSITORY}" "-DDUCKDB_WORK=${MVP_WORK}"
  "-DDUCKDB_PREFIX=${MVP_SELECTED_PREFIX}" -P "${MVP_REPOSITORY}/tests/duckdb-connectors/lifecycle.cmake")
