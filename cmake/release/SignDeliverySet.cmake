# Explicit signing action. This helper supplies no publisher authorization and
# never creates a key, trust policy, package, installation or provider effect.
include("${CMAKE_CURRENT_LIST_DIR}/Support.cmake")
foreach(name IN ITEMS RELEASE_DELIVERY_TOOL RELEASE_DELIVERY_ROOT RELEASE_DELIVERY_SELECTION
    RELEASE_ALLOWED_SIGNERS RELEASE_SIGN_KEY)
  if(NOT DEFINED ${name} OR NOT IS_ABSOLUTE "${${name}}")
    message(FATAL_ERROR "${name} must be an explicit absolute path")
  endif()
  if("${${name}}" MATCHES "[;\\[\\]\r\n]")
    message(FATAL_ERROR "Signing paths must not contain CMake delimiters or line breaks")
  endif()
endforeach()
foreach(name IN ITEMS RELEASE_DELIVERY_TOOL RELEASE_DELIVERY_SELECTION RELEASE_ALLOWED_SIGNERS RELEASE_SIGN_KEY)
  symphony_receipt_v2_regular_file("${${name}}")
endforeach()
if(NOT DEFINED RELEASE_EXPECTED_PROFILE OR NOT DEFINED RELEASE_SIGNER OR RELEASE_SIGNER STREQUAL "")
  message(FATAL_ERROR "Signing requires independent expected profile and signer identity")
endif()
set(index "${RELEASE_DELIVERY_ROOT}/delivery-set.json")
set(signature "${index}.sig")
if(EXISTS "${signature}" OR IS_SYMLINK "${signature}")
  message(FATAL_ERROR "Delivery signature already exists; it is never overwritten")
endif()
execute_process(COMMAND "${RELEASE_DELIVERY_TOOL}" -mode validate
  -root "${RELEASE_DELIVERY_ROOT}" -selection "${RELEASE_DELIVERY_SELECTION}"
  -allowed-signers "${RELEASE_ALLOWED_SIGNERS}" -profile "${RELEASE_EXPECTED_PROFILE}"
  COMMAND_ERROR_IS_FATAL ANY OUTPUT_VARIABLE prepared)
string(JSON scope GET "${prepared}" scope)
if(NOT scope STREQUAL "unsigned_bytes_only")
  message(FATAL_ERROR "Unexpected delivery preflight result")
endif()
file(SHA256 "${index}" before)
find_program(RELEASE_SSH_KEYGEN ssh-keygen REQUIRED)
execute_process(COMMAND "${RELEASE_SSH_KEYGEN}" -Y sign -f "${RELEASE_SIGN_KEY}"
  -n symphony-delivery-set "${index}" COMMAND_ERROR_IS_FATAL ANY)
file(SHA256 "${index}" after)
if(NOT before STREQUAL after)
  message(FATAL_ERROR "Index changed during signing; preserve work and inspect, do not retry automatically")
endif()
execute_process(COMMAND "${RELEASE_DELIVERY_TOOL}" -mode verify
  -root "${RELEASE_DELIVERY_ROOT}" -selection "${RELEASE_DELIVERY_SELECTION}"
  -allowed-signers "${RELEASE_ALLOWED_SIGNERS}" -profile "${RELEASE_EXPECTED_PROFILE}"
  -signer "${RELEASE_SIGNER}" COMMAND_ERROR_IS_FATAL ANY OUTPUT_VARIABLE verification)
string(JSON scope GET "${verification}" scope)
if(NOT scope STREQUAL "signed_bytes_only")
  message(FATAL_ERROR "Unexpected delivery verification result")
endif()
message(STATUS "Delivery index signed and byte-verified; owner/fresh-host/provider/release acceptance remains separate")
