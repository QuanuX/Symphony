cmake_minimum_required(VERSION 3.30)
if(NOT DEFINED SQFV_SOURCE_DIR OR NOT DEFINED SQFV_BINARY_DIR)
    message(FATAL_ERROR "package lifecycle test requires exact source and build directories")
endif()
set(test_root "${SQFV_BINARY_DIR}/package-lifecycle-test")
file(REMOVE_RECURSE "${test_root}")
file(MAKE_DIRECTORY "${test_root}")
set(receipt_relative "share/symphony/receipts/sqfv-batch-cpp/0.2.0-dev/install-receipt.json")

function(run_expect label should_succeed)
    execute_process(COMMAND ${ARGN} RESULT_VARIABLE result
        OUTPUT_VARIABLE output ERROR_VARIABLE error TIMEOUT 30)
    if(NOT result MATCHES "^[0-9]+$")
        message(FATAL_ERROR "${label} did not finish normally: ${result}")
    endif()
    if(should_succeed AND NOT result EQUAL 0)
        message(FATAL_ERROR "${label} failed (${result}):\n${output}\n${error}")
    elseif(NOT should_succeed AND result EQUAL 0)
        message(FATAL_ERROR "${label} unexpectedly succeeded")
    endif()
endfunction()

function(require_present path)
    if(NOT EXISTS "${path}")
        message(FATAL_ERROR "package test expected preserved content: ${path}")
    endif()
endfunction()

function(require_absent path)
    if(EXISTS "${path}" OR IS_SYMLINK "${path}")
        message(FATAL_ERROR "package test found unexpected content: ${path}")
    endif()
endfunction()

set(base "${test_root}/base")
run_expect("initial install" TRUE
    "${CMAKE_COMMAND}" --install "${SQFV_BINARY_DIR}" --prefix "${base}")
file(READ "${base}/${receipt_relative}" receipt)
string(JSON count LENGTH "${receipt}" files)
if(NOT count EQUAL 13)
    message(FATAL_ERROR "SQFV exact package inventory changed unexpectedly")
endif()
set(SQFV_INSTALLED_FILES "")
math(EXPR last "${count} - 1")
foreach(index RANGE 0 ${last})
    string(JSON relative GET "${receipt}" files ${index} path)
    list(APPEND SQFV_INSTALLED_FILES "${relative}")
    if(relative MATCHES "libsqfv-batch\\.a$")
        set(archive_relative "${relative}")
    endif()
endforeach()
if(NOT DEFINED archive_relative)
    message(FATAL_ERROR "SQFV package archive is missing from receipt")
endif()

# Configure the actual module template with a source path that does not exist.
# The detached scripts must still remove the exact installed package.
set(detached "${test_root}/detached")
file(MAKE_DIRECTORY "${detached}")
set(SYMPHONY_REPOSITORY_ROOT "${test_root}/removed-source")
set(SQFV_RECEIPT "${receipt_relative}")
set(CMAKE_INSTALL_PREFIX "${base}")
configure_file("${SQFV_SOURCE_DIR}/cmake/uninstall.cmake.in"
    "${detached}/uninstall.cmake" @ONLY)
string(MAKE_C_IDENTIFIER "${receipt_relative}" identity_id)
foreach(name SymphonyUninstallReceiptV2.cmake SymphonyReceiptV2Support.cmake
    "symphony-receipt-v2-identity-${identity_id}.cmake")
    file(COPY "${SQFV_BINARY_DIR}/${name}" DESTINATION "${detached}")
endforeach()
include("${detached}/SymphonyReceiptV2Support.cmake")

foreach(case trailing duplicate escaped_duplicate comment)
    set(prefix "${test_root}/malformed-${case}")
    file(COPY "${base}/" DESTINATION "${prefix}")
    if(case STREQUAL "trailing")
        set(changed "${receipt}not-json")
    elseif(case STREQUAL "comment")
        set(changed "/* changed */${receipt}")
    elseif(case STREQUAL "escaped_duplicate")
        string(REPLACE "\"component_kind\":\"module\""
            "\"component\\u005fkind\":\"adapter\",\"component_kind\":\"module\"" changed "${receipt}")
    else()
        string(REPLACE "\"component_kind\":\"module\""
            "\"component_kind\":\"adapter\",\"component_kind\":\"module\"" changed "${receipt}")
    endif()
    file(WRITE "${prefix}/${receipt_relative}" "${changed}")
    run_expect("malformed receipt ${case}" FALSE
        "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
    require_present("${prefix}/${archive_relative}")
    require_present("${prefix}/${receipt_relative}")
endforeach()

set(prefix "${test_root}/valid-pretty-receipt")
file(COPY "${base}/" DESTINATION "${prefix}")
string(JSON changed SET "${receipt}" component_kind "\"module\"")
file(WRITE "${prefix}/${receipt_relative}" " \n\t${changed}\n")
run_expect("valid alternate JSON whitespace and member order" TRUE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
require_absent("${prefix}/${receipt_relative}")

set(prefix "${test_root}/missing-identity-prefix")
set(incomplete "${test_root}/incomplete")
file(COPY "${base}/" DESTINATION "${prefix}")
file(COPY "${detached}/" DESTINATION "${incomplete}")
file(REMOVE "${incomplete}/symphony-receipt-v2-identity-${identity_id}.cmake")
run_expect("missing configured identity helper" FALSE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${incomplete}/uninstall.cmake")
require_present("${prefix}/${archive_relative}")
require_present("${prefix}/${receipt_relative}")

foreach(case owned receipt)
    set(prefix "${test_root}/fifo-${case}")
    file(COPY "${base}/" DESTINATION "${prefix}")
    if(case STREQUAL "owned")
        set(fifo "${prefix}/${archive_relative}")
    else()
        set(fifo "${prefix}/${receipt_relative}")
    endif()
    file(REMOVE "${fifo}")
    run_expect("create ${case} FIFO" TRUE /usr/bin/mkfifo "${fifo}")
    run_expect("reject ${case} FIFO before reading" FALSE
        "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
    require_present("${prefix}/${archive_relative}")
    require_present("${prefix}/${receipt_relative}")
endforeach()

set(prefix "${test_root}/fifo-install")
get_filename_component(parent "${prefix}/${archive_relative}" DIRECTORY)
file(MAKE_DIRECTORY "${parent}")
run_expect("create install FIFO" TRUE /usr/bin/mkfifo "${prefix}/${archive_relative}")
run_expect("reject install FIFO before copying" FALSE
    "${CMAKE_COMMAND}" --install "${SQFV_BINARY_DIR}" --prefix "${prefix}")
require_absent("${prefix}/${receipt_relative}")

run_expect("same version overwrite" FALSE
    "${CMAKE_COMMAND}" --install "${SQFV_BINARY_DIR}" --prefix "${base}")
require_present("${base}/${archive_relative}")

set(prefix "${test_root}/install-symlink")
set(outside "${test_root}/install-outside")
file(MAKE_DIRECTORY "${prefix}" "${outside}")
string(REGEX REPLACE "/.*$" "" archive_top "${archive_relative}")
file(CREATE_LINK "${outside}" "${prefix}/${archive_top}" SYMBOLIC)
run_expect("install through an intermediate symlink" FALSE
    "${CMAKE_COMMAND}" --install "${SQFV_BINARY_DIR}" --prefix "${prefix}")
file(GLOB_RECURSE outside_files "${outside}/*")
if(outside_files)
    message(FATAL_ERROR "rejected install wrote outside its prefix")
endif()
require_absent("${prefix}/${receipt_relative}")

set(prefix "${test_root}/uninstall-symlink")
set(outside "${test_root}/uninstall-outside")
file(COPY "${base}/" DESTINATION "${prefix}")
file(RENAME "${prefix}/${archive_top}" "${outside}")
file(CREATE_LINK "${outside}" "${prefix}/${archive_top}" SYMBOLIC)
run_expect("uninstall through an intermediate symlink" FALSE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
require_present("${prefix}/${archive_relative}")
require_present("${prefix}/${receipt_relative}")

set(prefix "${test_root}/receipt-digest")
file(COPY "${base}/" DESTINATION "${prefix}")
string(JSON changed SET "${receipt}" component_kind "\"adapter\"")
file(WRITE "${prefix}/${receipt_relative}" "${changed}\n")
run_expect("receipt self-digest mismatch" FALSE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
require_present("${prefix}/${archive_relative}")
require_present("${prefix}/${receipt_relative}")

# Even an internally consistent digest cannot change the configured identity.
string(JSON input REMOVE "${changed}" receipt_digest)
symphony_receipt_v2_canonical("${input}" OBJECT 0 canonical)
string(SHA256 digest "${canonical}")
string(JSON changed SET "${changed}" receipt_digest "\"sha256:${digest}\"")
file(WRITE "${prefix}/${receipt_relative}" "${changed}\n")
run_expect("receipt configured identity mismatch" FALSE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
require_present("${prefix}/${archive_relative}")
require_present("${prefix}/${receipt_relative}")

set(prefix "${test_root}/owned-content")
file(COPY "${base}/" DESTINATION "${prefix}")
file(APPEND "${prefix}/${archive_relative}" "changed")
run_expect("changed owned content" FALSE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
require_present("${prefix}/${archive_relative}")
require_present("${prefix}/${receipt_relative}")

set(prefix "${test_root}/managed-root")
file(COPY "${base}/" DESTINATION "${prefix}")
file(WRITE "${prefix}/.symphony-lifecycle-ownership-v1.json" "{}")
run_expect("shared-root fence" FALSE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
require_present("${prefix}/${archive_relative}")

run_expect("unsafe GNUInstallDirs at configure time" FALSE
    "${CMAKE_COMMAND}" -S "${SQFV_SOURCE_DIR}" -B "${test_root}/unsafe-build"
    -DBUILD_TESTING=OFF "-DCMAKE_INSTALL_LIBDIR=${test_root}/absolute-library")
require_absent("${test_root}/absolute-library")

set(spaces_build "${test_root}/spaces-build")
set(spaces_prefix "${test_root}/prefix with spaces")
run_expect("configure relative GNUInstallDirs with spaces" TRUE
    "${CMAKE_COMMAND}" -S "${SQFV_SOURCE_DIR}" -B "${spaces_build}"
    -DBUILD_TESTING=OFF "-DCMAKE_INSTALL_LIBDIR=lib custom"
    "-DCMAKE_INSTALL_INCLUDEDIR=include custom")
run_expect("build package with spaces" TRUE
    "${CMAKE_COMMAND}" --build "${spaces_build}" --target sqfv-batch -j 2)
run_expect("install relative GNUInstallDirs with spaces" TRUE
    "${CMAKE_COMMAND}" --install "${spaces_build}" --prefix "${spaces_prefix}")
require_present("${spaces_prefix}/${receipt_relative}")
run_expect("uninstall relative GNUInstallDirs with spaces" TRUE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${spaces_prefix}" -P "${spaces_build}/uninstall.cmake")
require_absent("${spaces_prefix}/${receipt_relative}")

set(prefix "${test_root}/detached-prefix")
file(COPY "${base}/" DESTINATION "${prefix}")
run_expect("detached source-independent uninstall" TRUE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
require_absent("${prefix}/${archive_relative}")
require_absent("${prefix}/${receipt_relative}")
run_expect("idempotent absent package" TRUE
    "${CMAKE_COMMAND}" "-DINSTALL_PREFIX=${prefix}" -P "${detached}/uninstall.cmake")
require_present("${base}/${archive_relative}")
message(STATUS "SQFV package lifecycle: prefix containment, receipt integrity/identity, fences, detached uninstall passed")
