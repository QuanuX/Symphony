# Runs after the complete installed runtime closure; test-only SDK cycles are
# resolved through that prefix, never by compiling a sibling SDK from source.
set(RELEASE_HARNESS "${MVP_REPOSITORY}/tests/release-qualification")
include("${RELEASE_HARNESS}/verify.cmake")
file(READ "${RELEASE_HARNESS}/manifest.json" qualification)
mvp_run(release-result-gate-regressions "${MVP_WORK}" "${CMAKE_COMMAND}"
  "-DRESULT_TEST_WORK=${MVP_WORK}/result-regressions" -P "${RELEASE_HARNESS}/result-regressions.cmake")
file(MAKE_DIRECTORY "${MVP_WORK}/fixtures")
if(MVP_GROUP STREQUAL "knowledge" OR MVP_GROUP STREQUAL "quantitative")
  execute_process(COMMAND "${CMAKE_COMMAND}" -E env GOTOOLCHAIN=local GOENV=off go env GOVERSION
    OUTPUT_VARIABLE go_version OUTPUT_STRIP_TRAILING_WHITESPACE RESULT_VARIABLE go_status)
  if(NOT go_status EQUAL 0 OR NOT go_version STREQUAL "go1.26.5")
    message(FATAL_ERROR "Installed Go release qualification requires Go1.26.5 on PATH")
  endif()
endif()

if(MVP_GROUP STREQUAL "knowledge")
  file(READ "${MVP_REPOSITORY}/modules/snv-engine/tests/fixtures/snv_inspect.json" bundle)
  mvp_indices(indices "${bundle}" bundle artifacts)
  set(found_sciv FALSE)
  foreach(index IN LISTS indices)
    string(JSON owner GET "${bundle}" bundle artifacts ${index} owner)
    if(owner STREQUAL "sciv")
      string(JSON input GET "${bundle}" bundle artifacts ${index} source_utf8)
      file(WRITE "${MVP_WORK}/fixtures/sciv_validate.json" "${input}\n")
      set(found_sciv TRUE)
    endif()
  endforeach()
  if(NOT found_sciv)
    message(FATAL_ERROR "Retained SNV bundle lacks the SCIV process fixture")
  endif()
  mvp_run(release-qxctl-build "${MVP_REPOSITORY}/tools/qxctl" "${CMAKE_COMMAND}" -E env
    GOWORK=off GOTOOLCHAIN=local GOENV=off "GOMODCACHE=${MVP_WORK}/go/modules"
    "GOCACHE=${MVP_WORK}/go/build" "GOPATH=${MVP_WORK}/go/path"
    GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GOAUTH=off "GOVCS=*:off"
    GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOFLAGS=
    go build -mod=readonly -trimpath -o "${MVP_WORK}/qxctl" ./cmd/qxctl)
endif()

if(MVP_GROUP STREQUAL "quantitative")
  # Copy only fixture code; all three writer SDKs are imported from the selected
  # installed prefix. Production inspection still contains no writer linkage.
  set(copy "${MVP_WORK}/consumers/retained-store-fixtures")
  file(COPY "${RELEASE_HARNESS}/sqv-fixtures/" DESTINATION "${copy}")
  file(COPY "${MVP_REPOSITORY}/tests/sqv-administration/fixtures.cpp"
    "${MVP_REPOSITORY}/tools/sqv-administration-cpp/common.hpp" DESTINATION "${copy}")
  file(WRITE "${MVP_WORK}/metadata/release-sqv-fixtures.json" [=[
{"project_version":null,"artifact_targets":["sqv-admin-fixtures"],"first_party_sdk_dependencies":[{"target":"Symphony::KnowledgeVectorEngine","scope":"runtime"},{"target":"Symphony::SqdvDelivery","scope":"runtime"},{"target":"Symphony::SqavDatabentoDbn","scope":"runtime"}]}
]=])
  mvp_run(release-sqv-fixtures-configure "${MVP_WORK}" "${CMAKE_COMMAND}" -S "${copy}" -B "${copy}/build"
    ${base_arguments} "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/release-sqv-fixtures.json"
    "-DMVP_PROOF_LOG=${MVP_WORK}/logs/release-sqv-fixtures-imports.log")
  mvp_run(release-sqv-fixtures-build "${MVP_WORK}" "${CMAKE_COMMAND}" --build "${copy}/build" --parallel "${MVP_JOBS}")
  mvp_run(release-sqv-fixtures-write "${MVP_WORK}/tmp" "${copy}/build/sqv-admin-fixtures" "${MVP_WORK}/fixtures/stores")
endif()

mvp_indices(suites "${qualification}" native_suites)
foreach(index IN LISTS suites)
  string(JSON suite GET "${qualification}" native_suites ${index})
  string(JSON group GET "${suite}" group)
  if(NOT group STREQUAL MVP_GROUP)
    continue()
  endif()
  string(JSON path GET "${suite}" path)
  if(NOT path IN_LIST ordered)
    message(FATAL_ERROR "Selected native suite has no installed build: ${path}")
  endif()
  mvp_component("${path}" component)
  string(JSON id GET "${component}" component_id)
  set(build "${MVP_WORK}/build/${id}")
  component_arguments("${component}" arguments)
  string(JSON options GET "${suite}" configure_options)
  mvp_indices(option_indices "${options}")
  foreach(option_index IN LISTS option_indices)
    string(JSON name MEMBER "${options}" ${option_index})
    string(JSON value GET "${options}" "${name}")
    list(APPEND arguments "-D${name}=${value}")
  endforeach()
  mvp_run("${id}-release-test-configure" "${MVP_WORK}" "${CMAKE_COMMAND}"
    -S "${MVP_REPOSITORY}/${path}" -B "${build}" ${arguments}
    "-DMVP_PROOF_LOG=${MVP_WORK}/logs/${id}-test-imports.log")
  mvp_run("${id}-release-test-build" "${MVP_WORK}" "${CMAKE_COMMAND}" --build "${build}" --parallel "${MVP_JOBS}")
  mvp_run("${id}-release-test-discovery" "${MVP_WORK}" "${MVP_CTEST}" --test-dir "${build}" --show-only=json-v1)
  file(READ "${MVP_WORK}/logs/${id}-release-test-discovery.log" discovery)
  set(actual "")
  mvp_indices(test_indices "${discovery}" tests)
  foreach(test_index IN LISTS test_indices)
    string(JSON name GET "${discovery}" tests ${test_index} name)
    list(APPEND actual "${name}")
  endforeach()
  set(expected "")
  mvp_indices(test_indices "${suite}" ctests)
  foreach(test_index IN LISTS test_indices)
    string(JSON name GET "${suite}" ctests ${test_index})
    list(APPEND expected "${name}")
  endforeach()
  list(SORT actual)
  list(SORT expected)
  if(NOT actual STREQUAL expected)
    message(FATAL_ERROR "Configured tests differ from manifest for ${path}: expected [${expected}], found [${actual}]")
  endif()
  list(LENGTH expected count)
  if(count GREATER 0)
    mvp_run("${id}-release-tests" "${MVP_WORK}" "${MVP_CTEST}" --test-dir "${build}"
      -V --no-tests=error --output-junit "${MVP_WORK}/logs/${id}-release-tests.xml")
    set(skips "")
    mvp_indices(skip_indices "${suite}" known_internal_skips)
    foreach(skip_index IN LISTS skip_indices)
      string(JSON skip GET "${suite}" known_internal_skips ${skip_index})
      list(APPEND skips "${skip}")
    endforeach()
    mvp_run("${id}-release-test-results" "${MVP_WORK}" "${CMAKE_COMMAND}"
      "-DRESULT_JUNIT=${MVP_WORK}/logs/${id}-release-tests.xml"
      "-DRESULT_LOG=${MVP_WORK}/logs/${id}-release-tests.log" "-DEXPECTED_TEST_COUNT=${count}"
      "-DALLOWED_INTERNAL_SKIPS=${skips}" -P "${RELEASE_HARNESS}/check-results.cmake")
  endif()

  # No-CTest roots still have real process tests. Each invocation uses its
  # retained owner-specific input, exact installed release and scratch output.
  set(REPO "${MVP_REPOSITORY}")
  set(SOURCE "${MVP_REPOSITORY}")
  set(WORK "${MVP_WORK}")
  set(PREFIX "${MVP_SELECTED_PREFIX}")
  set(QXCTL "${MVP_WORK}/qxctl")
  set(SQV_GENERATED_FIXTURES "${MVP_WORK}/fixtures/stores")
  set(PARTITION_PREFIX "${MVP_SELECTED_PREFIX}")
  mvp_component("modules/shv-partition-engine" partition)
  string(JSON PARTITION_VERSION GET "${partition}" package_release_version)
  string(JSON VERSION GET "${component}" package_release_version)
  file(STRINGS "${build}/CMakeCache.txt" compiler_line REGEX "^CMAKE_CXX_COMPILER:(FILEPATH|STRING)=")
  string(REGEX REPLACE "^[^=]*=" "" COMPILER "${compiler_line}")
  mvp_indices(direct_indices "${suite}" direct_tests)
  foreach(direct_index IN LISTS direct_indices)
    string(JSON target GET "${suite}" direct_tests ${direct_index} target)
    set(OUTPUT "${MVP_WORK}/logs/${target}-evidence.json")
    set(command "${build}/${target}")
    mvp_indices(argument_indices "${suite}" direct_tests ${direct_index} arguments)
    foreach(argument_index IN LISTS argument_indices)
      string(JSON argument GET "${suite}" direct_tests ${direct_index} arguments ${argument_index})
      string(REGEX MATCHALL "[$][{][A-Za-z_]+[}]" substitutions "${argument}")
      foreach(substitution IN LISTS substitutions)
        string(REGEX REPLACE "^[$][{](.*)[}]$" "\\1" variable "${substitution}")
        if(NOT variable MATCHES "^(REPO|SOURCE|WORK|PREFIX|QXCTL|SQV_GENERATED_FIXTURES|PARTITION_PREFIX|PARTITION_VERSION|VERSION|COMPILER|OUTPUT)$" OR
           NOT DEFINED ${variable} OR "${${variable}}" STREQUAL "")
          message(FATAL_ERROR "Unqualified direct test substitution: ${substitution}")
        endif()
      endforeach()
      string(CONFIGURE "${argument}" argument)
      if(argument MATCHES "[$][{]")
        message(FATAL_ERROR "Unresolved direct test argument: ${argument}")
      endif()
      list(APPEND command "${argument}")
    endforeach()
    mvp_run("${target}-release-direct" "${MVP_WORK}/tmp" ${command})
    file(READ "${MVP_WORK}/logs/${target}-release-direct.log" result)
    if(result MATCHES "(^|[\n :])SKIP[ :]")
      message(FATAL_ERROR "Selected direct test skipped: ${target}")
    endif()
  endforeach()
  file(WRITE "${MVP_WORK}/logs/${id}-release-selection.json" "${suite}\n")
endforeach()

if(MVP_GROUP STREQUAL "knowledge" OR MVP_GROUP STREQUAL "quantitative")
  mvp_run(release-installed-go "${MVP_REPOSITORY}" "${CMAKE_COMMAND}" -E env
    GOWORK=off GOTOOLCHAIN=local GOENV=off "GOMODCACHE=${MVP_WORK}/go/modules"
    "GOCACHE=${MVP_WORK}/go/build" "GOPATH=${MVP_WORK}/go/path"
    GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GOAUTH=off "GOVCS=*:off"
    GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOFLAGS= CGO_ENABLED=1
    go run "${RELEASE_HARNESS}/go-runner.go" --repo "${MVP_REPOSITORY}"
    --manifest "${RELEASE_HARNESS}/manifest.json" --work "${MVP_WORK}/logs/installed-go"
    --mode installed --group "${MVP_GROUP}" --prefix "${MVP_SELECTED_PREFIX}")
endif()
if(MVP_GROUP STREQUAL "knowledge")
  mvp_component("tools/symphony-validator" validator)
  string(JSON release GET "${validator}" package_release_version)
  mvp_run(release-installed-validator "${MVP_WORK}/tmp"
    "${MVP_SELECTED_PREFIX}/libexec/symphony/symphony-validator/${release}/symphony-validator"
    check --repo "${MVP_REPOSITORY}" --json)
endif()
