include("${CMAKE_CURRENT_LIST_DIR}/common.cmake")
file(READ "${MVP_MAP}" MVP_MAP_JSON)
include("${MVP_ARGUMENTS_FILE}")
find_program(MVP_CTEST NAMES ctest REQUIRED)
if(NOT MVP_JOBS MATCHES "^[1-9][0-9]*$")
  message(FATAL_ERROR "MVP_BUILD_JOBS must be a positive integer")
endif()
mvp_run(proof-scope-regressions "${MVP_WORK}" "${CMAKE_COMMAND}"
  "-DMVP_REPOSITORY=${MVP_REPOSITORY}" "-DMVP_PROOF_TEST_WORK=${MVP_WORK}/proof-regressions"
  -P "${CMAKE_CURRENT_LIST_DIR}/proof-regressions.cmake")
set(MVP_SELECTED_PREFIX "${MVP_WORK}/prefix")
set(external_prefix "$ENV{CMAKE_PREFIX_PATH}")
set(extra "")
foreach(argument IN LISTS MVP_EXTRA_ARGS)
  if(argument MATCHES "^-DCMAKE_PREFIX_PATH(:[^=]+)?=(.*)$")
    set(external_prefix "${CMAKE_MATCH_2}")
  else()
    list(APPEND extra "${argument}")
  endif()
endforeach()
# Keep the selected SDK prefix as one cache argument. Passing a semicolon list
# through multiple CMake function/list layers can split the external paths into
# unrelated command arguments; POSIX CMAKE_PREFIX_PATH uses colons instead.
string(REPLACE ";" ":" external_prefix "${external_prefix}")
set(ENV{CMAKE_PREFIX_PATH} "${external_prefix}")
set(base_arguments ${extra}
  "-DCMAKE_PREFIX_PATH=${MVP_SELECTED_PREFIX}" "-DCMAKE_INSTALL_PREFIX=${MVP_SELECTED_PREFIX}"
  -DCMAKE_BUILD_TYPE=Release -DCMAKE_POSITION_INDEPENDENT_CODE=ON
  -DCMAKE_INSTALL_RPATH_USE_LINK_PATH=ON
  -DBUILD_TESTING=OFF -DSYMPHONY_KVE_BUILD_TESTING=OFF
  -DCMAKE_FIND_USE_PACKAGE_REGISTRY=OFF -DCMAKE_FIND_USE_SYSTEM_PACKAGE_REGISTRY=OFF
  -DCMAKE_FIND_PACKAGE_NO_PACKAGE_REGISTRY=ON -DCMAKE_EXPORT_NO_PACKAGE_REGISTRY=ON
  "-DCMAKE_PROJECT_TOP_LEVEL_INCLUDES=${CMAKE_CURRENT_LIST_DIR}/prove-imports.cmake"
  "-DMVP_SELECTED_PREFIX=${MVP_SELECTED_PREFIX}" "-DMVP_EXTERNAL_PREFIX=${external_prefix}"
  "-DMVP_REPOSITORY=${MVP_REPOSITORY}")

function(select_runtime_closure path)
  get_property(selected GLOBAL PROPERTY MVP_SELECTION)
  if(path IN_LIST selected)
    return()
  endif()
  set_property(GLOBAL APPEND PROPERTY MVP_SELECTION "${path}")
  mvp_component("${path}" component)
  mvp_indices(indices "${component}" first_party_sdk_dependencies)
  foreach(index IN LISTS indices)
    string(JSON scope GET "${component}" first_party_sdk_dependencies ${index} scope)
    if(scope STREQUAL "runtime")
      string(JSON dependency GET "${component}" first_party_sdk_dependencies ${index} path)
      select_runtime_closure("${dependency}")
    endif()
  endforeach()
endfunction()

mvp_indices(indices "${MVP_MAP_JSON}" native_components)
foreach(index IN LISTS indices)
  string(JSON component GET "${MVP_MAP_JSON}" native_components ${index})
  string(JSON group GET "${component}" qualification_group)
  string(JSON installable GET "${component}" installable)
  if(group STREQUAL MVP_GROUP AND (installable OR MVP_GROUP STREQUAL "build_support"))
    string(JSON path GET "${component}" path)
    select_runtime_closure("${path}")
  endif()
endforeach()
get_property(selected GLOBAL PROPERTY MVP_SELECTION)
if(NOT selected)
  message(FATAL_ERROR "Qualification group selected no components")
endif()
set(ordered "")
mvp_indices(indices "${MVP_MAP_JSON}" native_build_order)
foreach(index IN LISTS indices)
  string(JSON path GET "${MVP_MAP_JSON}" native_build_order ${index})
  if(path IN_LIST selected)
    list(APPEND ordered "${path}")
  endif()
endforeach()
list(LENGTH selected count)
list(LENGTH ordered ordered_count)
if(NOT count EQUAL ordered_count)
  message(FATAL_ERROR "Runtime closure is incomplete in mapped build order")
endif()
file(WRITE "${MVP_WORK}/selected-components.txt" "${MVP_GROUP}\n")
foreach(path IN LISTS ordered)
  file(APPEND "${MVP_WORK}/selected-components.txt" "${path}\n")
endforeach()

function(component_arguments component output)
  string(JSON id GET "${component}" component_id)
  set(arguments ${base_arguments}
    "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/${id}.json"
    "-DMVP_PROOF_LOG=${MVP_WORK}/logs/${id}-imports.log")
  string(JSON options GET "${component}" installed_options)
  mvp_indices(indices "${options}")
  foreach(index IN LISTS indices)
    string(JSON name MEMBER "${options}" ${index})
    string(JSON value GET "${options}" "${name}")
    list(APPEND arguments "-D${name}=${value}")
  endforeach()
  set(${output} "${arguments}" PARENT_SCOPE)
endfunction()

set(negative_checked FALSE)
foreach(path IN LISTS ordered)
  mvp_component("${path}" component)
  string(JSON id GET "${component}" component_id)
  file(WRITE "${MVP_WORK}/metadata/${id}.json" "${component}\n")
  component_arguments("${component}" arguments)
  if(NOT negative_checked)
    mvp_indices(dependencies "${component}" first_party_sdk_dependencies)
    foreach(index IN LISTS dependencies)
      string(JSON scope GET "${component}" first_party_sdk_dependencies ${index} scope)
      if(scope STREQUAL "runtime")
        string(JSON package GET "${component}" first_party_sdk_dependencies ${index} package)
        set(log "${MVP_WORK}/logs/missing-sdk-refusal.log")
        execute_process(COMMAND "${CMAKE_COMMAND}" -S "${MVP_REPOSITORY}/${path}"
          -B "${MVP_WORK}/negative/${id}" ${arguments}
          "-DCMAKE_DISABLE_FIND_PACKAGE_${package}=TRUE"
          RESULT_VARIABLE status OUTPUT_FILE "${log}" ERROR_FILE "${log}")
        file(READ "${log}" refusal)
        if(status EQUAL 0 OR NOT refusal MATCHES "CMAKE_DISABLE_FIND_PACKAGE_${package}")
          message(FATAL_ERROR "Missing required installed SDK did not fail explicitly: ${package}\n${refusal}")
        endif()
        message(STATUS "PASS required installed SDK refusal: ${package}")
        set(negative_checked TRUE)
        break()
      endif()
    endforeach()
  endif()
  mvp_run("${id}-configure" "${MVP_WORK}" "${CMAKE_COMMAND}"
    -S "${MVP_REPOSITORY}/${path}" -B "${MVP_WORK}/build/${id}" ${arguments})
  # Build the complete package: auxiliary installed entry points must exist too.
  # artifact_targets are independently checked declarations, not a partial build.
  mvp_run("${id}-build" "${MVP_WORK}" "${CMAKE_COMMAND}"
    --build "${MVP_WORK}/build/${id}" --parallel "${MVP_JOBS}")
  string(JSON installable GET "${component}" installable)
  if(installable)
    mvp_run("${id}-install" "${MVP_WORK}" "${CMAKE_COMMAND}" --install "${MVP_WORK}/build/${id}")
    mvp_run("${id}-receipt" "${MVP_WORK}" "${CMAKE_COMMAND}"
      "-DMVP_REPOSITORY=${MVP_REPOSITORY}" "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/${id}.json"
      "-DMVP_SELECTED_PREFIX=${MVP_SELECTED_PREFIX}" -P "${CMAKE_CURRENT_LIST_DIR}/verify-receipt.cmake")
  endif()
endforeach()

# Copy existing public consumers out of the checkout after all runtime SDKs are installed.
foreach(path IN LISTS ordered)
  mvp_component("${path}" component)
  string(JSON id GET "${component}" component_id)
  mvp_indices(indices "${component}" installed_sdk_consumer_checks)
  foreach(index IN LISTS indices)
    string(JSON source GET "${component}" installed_sdk_consumer_checks ${index} path)
    if(NOT source MATCHES "/tests/sdk-consumer/CMakeLists.txt$")
      continue()
    endif()
    get_filename_component(source "${MVP_REPOSITORY}/${source}" DIRECTORY)
    set(copy "${MVP_WORK}/consumers/${id}")
    file(COPY "${source}/" DESTINATION "${copy}")
    set(consumer "${component}")
    string(JSON consumer SET "${consumer}" project_version null)
    string(JSON consumer SET "${consumer}" artifact_targets "[\"consumer\"]")
    string(JSON dependencies GET "${consumer}" first_party_sdk_dependencies)
    mvp_indices(exports "${component}" exported_packages)
    foreach(export IN LISTS exports)
      string(JSON package GET "${component}" exported_packages ${export})
      mvp_indices(targets "${package}" targets)
      foreach(target IN LISTS targets)
        string(JSON name GET "${package}" targets ${target})
        string(JSON length LENGTH "${dependencies}")
        string(JSON dependencies SET "${dependencies}" ${length} "{\"target\":\"${name}\",\"scope\":\"runtime\"}")
      endforeach()
    endforeach()
    string(JSON consumer SET "${consumer}" first_party_sdk_dependencies "${dependencies}")
    file(WRITE "${MVP_WORK}/metadata/${id}-consumer.json" "${consumer}\n")
    mvp_run("${id}-consumer-configure" "${MVP_WORK}" "${CMAKE_COMMAND}" -S "${copy}" -B "${copy}/build"
      ${base_arguments} "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/${id}-consumer.json"
      "-DMVP_PROOF_LOG=${MVP_WORK}/logs/${id}-consumer-imports.log")
    mvp_run("${id}-consumer-build" "${MVP_WORK}" "${CMAKE_COMMAND}" --build "${copy}/build" --parallel "${MVP_JOBS}")
    mvp_run("${id}-consumer-run" "${MVP_WORK}/tmp" "${copy}/build/consumer")
    if(id STREQUAL "sqav-capture-cpp")
      file(READ "${MVP_WORK}/logs/${id}-consumer-run.log" result)
      if(NOT result MATCHES "3 groups passed")
        message(FATAL_ERROR "Capture SDK private-export regressions did not execute")
      endif()
    endif()
  endforeach()
endforeach()

if("modules/sbv-engine" IN_LIST ordered)
  set(copy "${MVP_WORK}/consumers/sbv-exported-sdks")
  file(COPY "${CMAKE_CURRENT_LIST_DIR}/sbv-consumer/" DESTINATION "${copy}")
  file(WRITE "${MVP_WORK}/metadata/sbv-consumers.json" [=[
{"project_version":null,"artifact_targets":["sbv-installed-c-consumer","sbv-installed-cpp-consumer","sbv-installed-provider-abi","sbv-installed-interop"],"first_party_sdk_dependencies":[{"target":"Symphony::SbvSdk","scope":"runtime"},{"target":"Symphony::SbvProvider","scope":"runtime"},{"target":"Symphony::SbvInterop","scope":"runtime"}]}
]=])
  mvp_run(sbv-consumer-configure "${MVP_WORK}" "${CMAKE_COMMAND}" -S "${copy}" -B "${copy}/build"
    ${base_arguments} "-DSBV_PREFIX=${MVP_SELECTED_PREFIX}" -DBUILD_TESTING=ON
    "-DMVP_COMPONENT_FILE=${MVP_WORK}/metadata/sbv-consumers.json"
    "-DMVP_PROOF_LOG=${MVP_WORK}/logs/sbv-consumer-imports.log")
  mvp_run(sbv-consumer-build "${MVP_WORK}" "${CMAKE_COMMAND}" --build "${copy}/build" --parallel "${MVP_JOBS}")
  mvp_run(sbv-consumer-tests "${MVP_WORK}" "${MVP_CTEST}" --test-dir "${copy}/build"
    --output-on-failure --no-tests=error)
  if(CMAKE_HOST_SYSTEM_NAME STREQUAL "Darwin")
    set(loader_environment DYLD_PRINT_LIBRARIES=1)
  elseif(CMAKE_HOST_SYSTEM_NAME STREQUAL "Linux")
    set(loader_environment LD_TRACE_LOADED_OBJECTS=1)
  else()
    message(FATAL_ERROR "SBV loader qualification supports macOS and Linux only")
  endif()
  # Keep loader diagnostics separate from the byte-correspondence CTest, which
  # deliberately requires empty stderr from both SDK consumer invocations.
  mvp_run(sbv-consumer-loader "${MVP_WORK}/tmp" "${CMAKE_COMMAND}" -E env
    "${loader_environment}" "${copy}/build/sbv-installed-c-consumer"
    "${copy}/build/descriptor-request.json")
  mvp_run(sbv-consumer-loader-proof "${MVP_WORK}" "${CMAKE_COMMAND}"
    "-DMVP_LOADER_LOG=${MVP_WORK}/logs/sbv-consumer-loader.log"
    "-DMVP_EXTERNAL_PREFIX=${external_prefix}" "-DMVP_LOADER_SYSTEM=${CMAKE_HOST_SYSTEM_NAME}"
    -P "${CMAKE_CURRENT_LIST_DIR}/verify-loader.cmake")
endif()

include("${MVP_REPOSITORY}/tests/release-qualification/native.cmake")
if(MVP_GROUP STREQUAL "duckdb")
  include("${MVP_REPOSITORY}/tests/duckdb-connectors/installed.cmake")
endif()

if(MVP_GROUP STREQUAL "cloud")
  execute_process(COMMAND "${CMAKE_COMMAND}" -E env GOTOOLCHAIN=local GOENV=off go env GOVERSION
    OUTPUT_VARIABLE go_version OUTPUT_STRIP_TRAILING_WHITESPACE RESULT_VARIABLE go_status)
  if(NOT go_status EQUAL 0 OR NOT go_version STREQUAL "go1.26.5")
    message(FATAL_ERROR "Cloud CLI qualification requires Go1.26.5 on PATH")
  endif()
  mvp_run(qxctl-build "${MVP_REPOSITORY}/tools/qxctl" "${CMAKE_COMMAND}" -E env
    GOWORK=off GOTOOLCHAIN=local GOENV=off "GOMODCACHE=${MVP_WORK}/go/modules"
    "GOCACHE=${MVP_WORK}/go/build" "GOPATH=${MVP_WORK}/go/path"
    GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GOAUTH=off "GOVCS=*:off"
    GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= GOFLAGS=
    go build -mod=readonly -trimpath -o "${MVP_WORK}/qxctl" ./cmd/qxctl)
  foreach(domain scv schv scev schv-aws schv-azure schv-do schv-gcp scev-cf)
    mvp_run("${domain}-cli-inspect" "${MVP_WORK}/tmp" "${MVP_WORK}/qxctl" scv inspect
      --domain "${domain}" --prefix "${MVP_SELECTED_PREFIX}" --version 0.10.0-dev --json)
    mvp_run("${domain}-cli-interface" "${MVP_WORK}/tmp" "${MVP_WORK}/qxctl" scv interface show
      --domain "${domain}" --prefix "${MVP_SELECTED_PREFIX}" --version 0.10.0-dev --json)
    mvp_run("${domain}-cli-schema" "${MVP_WORK}/tmp" "${MVP_WORK}/qxctl" scv schema list
      --domain "${domain}" --prefix "${MVP_SELECTED_PREFIX}" --version 0.10.0-dev --json)
    file(READ "${MVP_WORK}/logs/${domain}-cli-inspect.log" response)
    string(JSON version GET "${response}" engine_version)
    if(NOT version STREQUAL "0.10.0-dev")
      message(FATAL_ERROR "Actual SCV CLI selected an unexpected package")
    endif()
  endforeach()
  execute_process(COMMAND "${MVP_WORK}/qxctl" scv inspect --prefix "${MVP_SELECTED_PREFIX}" --json
    RESULT_VARIABLE old_status OUTPUT_FILE "${MVP_WORK}/logs/scv-legacy-default-refusal.log"
    ERROR_FILE "${MVP_WORK}/logs/scv-legacy-default-refusal.log")
  if(old_status EQUAL 0)
    message(FATAL_ERROR "SCV legacy default silently selected current-only installation")
  endif()
endif()
message(STATUS "PASS ${MVP_GROUP}: ${count} components; evidence retained in ${MVP_WORK}")
