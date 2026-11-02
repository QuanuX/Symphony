cmake_minimum_required(VERSION 3.30)
# Native generated-interface parity invokes gofmt. Pin the formatter together
# with Go instead of depending on an optional runner-image PATH installation.
execute_process(COMMAND "${CMAKE_COMMAND}" -E env GOTOOLCHAIN=local GOENV=off
  go env -json GOVERSION GOROOT OUTPUT_VARIABLE toolchain ERROR_VARIABLE diagnostic
  RESULT_VARIABLE status TIMEOUT 15)
if(NOT status EQUAL 0)
  message(FATAL_ERROR "Native release qualification requires Go1.26.5 and its gofmt on PATH: ${diagnostic}")
endif()
string(JSON version GET "${toolchain}" GOVERSION)
string(JSON root GET "${toolchain}" GOROOT)
if(NOT version STREQUAL "go1.26.5")
  message(FATAL_ERROR "Native release qualification requires Go1.26.5; selected ${version}")
endif()
find_program(formatter NAMES gofmt NO_CACHE REQUIRED)
file(REAL_PATH "${formatter}" actual_formatter)
file(REAL_PATH "${root}/bin/gofmt" expected_formatter)
if(NOT actual_formatter STREQUAL expected_formatter)
  message(FATAL_ERROR "Native generated-interface parity requires gofmt from the selected Go1.26.5 toolchain")
endif()
message(STATUS "PASS native Go/gofmt prerequisite: ${version}; ${actual_formatter}")
