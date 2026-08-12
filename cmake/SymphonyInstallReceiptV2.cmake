include_guard(GLOBAL)

function(symphony_install_receipt_v2_preflight)
    set(one_value_args RECEIPT_PATH)
    cmake_parse_arguments(PREFLIGHT "" "${one_value_args}" "" ${ARGN})

    if(NOT DEFINED PREFLIGHT_RECEIPT_PATH OR PREFLIGHT_RECEIPT_PATH STREQUAL "")
        message(FATAL_ERROR "receipt-v2 preflight RECEIPT_PATH is required")
    endif()
    if(PREFLIGHT_RECEIPT_PATH MATCHES "(^/|(^|/)\\.\\.?(/|$)|//|\\\\)")
        message(FATAL_ERROR "receipt-v2 preflight path is unsafe: ${PREFLIGHT_RECEIPT_PATH}")
    endif()

    set(SYMPHONY_RECEIPT_PREFLIGHT_PATH "${PREFLIGHT_RECEIPT_PATH}")
    string(MAKE_C_IDENTIFIER "${PREFLIGHT_RECEIPT_PATH}" receipt_preflight_id)
    set(preflight_script
        "${CMAKE_CURRENT_BINARY_DIR}/symphony-install-receipt-v2-preflight-${receipt_preflight_id}.cmake")
    configure_file(
        "${CMAKE_CURRENT_FUNCTION_LIST_DIR}/SymphonyInstallReceiptV2Preflight.cmake.in"
        "${preflight_script}"
        @ONLY)
    configure_file(
        "${CMAKE_CURRENT_FUNCTION_LIST_DIR}/SymphonyUninstallReceiptV2.cmake"
        "${CMAKE_CURRENT_BINARY_DIR}/SymphonyUninstallReceiptV2.cmake"
        COPYONLY)
    install(SCRIPT "${preflight_script}")
endfunction()

function(symphony_install_receipt_v2)
    set(one_value_args
        COMPONENT_ID COMPONENT_KIND MODULE_ID VECTOR_ID ENGINE_ID PACKAGE_ID
        VERSION RECEIPT_PATH)
    set(multi_value_args
        OWNED_FILES ENTRY_POINTS PROVIDES_CAPABILITIES REQUIRES_CAPABILITIES
        COMPATIBLE_RECEPTORS)
    cmake_parse_arguments(RECEIPT "" "${one_value_args}" "${multi_value_args}" ${ARGN})

    foreach(required COMPONENT_ID COMPONENT_KIND MODULE_ID PACKAGE_ID VERSION RECEIPT_PATH)
        if(NOT DEFINED RECEIPT_${required} OR RECEIPT_${required} STREQUAL "")
            message(FATAL_ERROR "receipt-v2 ${required} is required")
        endif()
    endforeach()
    if(NOT RECEIPT_OWNED_FILES)
        message(FATAL_ERROR "receipt-v2 requires at least one owned file")
    endif()
    if(RECEIPT_RECEIPT_PATH MATCHES "(^/|(^|/)\\.\\.?(/|$)|//|\\\\)")
        message(FATAL_ERROR "receipt-v2 path is unsafe: ${RECEIPT_RECEIPT_PATH}")
    endif()
    foreach(owned_spec IN LISTS RECEIPT_OWNED_FILES)
        string(REPLACE "|" ";" owned_fields "${owned_spec}")
        list(LENGTH owned_fields owned_field_count)
        if(NOT owned_field_count EQUAL 2)
            message(FATAL_ERROR "invalid receipt-v2 owned-file declaration: ${owned_spec}")
        endif()
        list(GET owned_fields 0 owned_path)
        if(owned_path MATCHES "^\\.symphony-" OR
           owned_path MATCHES "^share/symphony/receipts(/|$)")
            message(FATAL_ERROR "receipt-v2 package cannot own a reserved lifecycle path: ${owned_path}")
        endif()
    endforeach()

    if(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
        set(SYMPHONY_RECEIPT_OS "macos")
    elseif(CMAKE_SYSTEM_NAME STREQUAL "Linux")
        set(SYMPHONY_RECEIPT_OS "linux")
    else()
        message(FATAL_ERROR "receipt-v2 packaging supports Linux and macOS only")
    endif()
    string(TOLOWER "${CMAKE_SYSTEM_PROCESSOR}" SYMPHONY_RECEIPT_ARCH)
    if(SYMPHONY_RECEIPT_ARCH MATCHES "^(x86_64|amd64)$")
        set(SYMPHONY_RECEIPT_ARCH "amd64")
    elseif(SYMPHONY_RECEIPT_ARCH MATCHES "^(aarch64|arm64)$")
        set(SYMPHONY_RECEIPT_ARCH "arm64")
    else()
        message(FATAL_ERROR "receipt-v2 packaging does not recognize target architecture ${CMAKE_SYSTEM_PROCESSOR}")
    endif()

    foreach(name COMPONENT_ID COMPONENT_KIND MODULE_ID VECTOR_ID ENGINE_ID PACKAGE_ID VERSION RECEIPT_PATH)
        set(SYMPHONY_RECEIPT_${name} "${RECEIPT_${name}}")
    endforeach()
    foreach(name OWNED_FILES ENTRY_POINTS PROVIDES_CAPABILITIES REQUIRES_CAPABILITIES COMPATIBLE_RECEPTORS)
        set(SYMPHONY_RECEIPT_${name} "${RECEIPT_${name}}")
    endforeach()

    string(MAKE_C_IDENTIFIER "${RECEIPT_PACKAGE_ID}_${RECEIPT_VERSION}" receipt_script_id)
    set(receipt_script
        "${CMAKE_CURRENT_BINARY_DIR}/symphony-install-receipt-v2-${receipt_script_id}.cmake")
    configure_file(
        "${CMAKE_CURRENT_FUNCTION_LIST_DIR}/SymphonyInstallReceiptV2.cmake.in"
        "${receipt_script}"
        @ONLY)
    install(SCRIPT "${receipt_script}")
endfunction()
