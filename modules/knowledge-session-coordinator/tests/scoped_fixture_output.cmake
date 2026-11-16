cmake_minimum_required(VERSION 3.25)

if(NOT DEFINED BINARY OR NOT DEFINED TEST_ROOT)
    message(FATAL_ERROR "BINARY and TEST_ROOT are required")
endif()

string(RANDOM LENGTH 24 ALPHABET 0123456789abcdef nonce)
set(directory "${TEST_ROOT}/scoped-fixture-output-${nonce}")
file(MAKE_DIRECTORY "${directory}")
file(WRITE "${directory}/retained.json" "retained caller bytes\n")
file(CREATE_LINK "${directory}/retained.json" "${directory}/redirect.json" SYMBOLIC)

# The former environment-selected destination must neither overwrite a file
# through a symlink nor create a previously absent file. JSON belongs to stdout.
foreach(destination redirect.json absent.json)
    execute_process(
        COMMAND "${CMAKE_COMMAND}" -E env
                "SYMPHONY_SCOPED_APPLY_FIXTURE_OUTPUT=${directory}/${destination}"
                "${BINARY}" --scoped-apply-fixture-json
        RESULT_VARIABLE status OUTPUT_VARIABLE output ERROR_VARIABLE error
    )
    if(NOT status EQUAL 0)
        message(FATAL_ERROR "scoped fixture export failed: ${status}: ${error}")
    endif()
    string(JSON protocol GET "${output}" protocol)
    if(NOT protocol STREQUAL "symphony.knowledge.scoped-apply-native-fixtures.v1")
        message(FATAL_ERROR "fixture stdout is not the native JSON document")
    endif()
    foreach(kind activation installation)
        string(JSON scope_digest GET "${output}" "${kind}" scope scope_digest)
        string(JSON prepared_scope GET "${output}" "${kind}" prepare_result scope_digest)
        if(NOT scope_digest STREQUAL prepared_scope)
            message(FATAL_ERROR "fixture lost its ${kind} exact scope binding")
        endif()
    endforeach()
    file(READ "${directory}/retained.json" retained)
    if(NOT retained STREQUAL "retained caller bytes\n" OR
       NOT IS_SYMLINK "${directory}/redirect.json" OR EXISTS "${directory}/absent.json")
        message(FATAL_ERROR "fixture export changed an environment-selected destination")
    endif()
endforeach()

execute_process(
    COMMAND "${BINARY}" --scoped-apply-fixture-json "${directory}/unexpected.json"
    RESULT_VARIABLE status OUTPUT_VARIABLE output ERROR_VARIABLE error
)
if(status EQUAL 0 OR EXISTS "${directory}/unexpected.json" OR NOT output STREQUAL "")
    message(FATAL_ERROR "fixture export accepted a destination argument")
endif()

file(REMOVE_RECURSE "${directory}")
message(STATUS "scoped fixture stdout and destination-preservation regressions passed")
