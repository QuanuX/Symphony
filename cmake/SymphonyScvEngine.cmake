include(CTest)
include(GNUInstallDirs)
if(NOT CMAKE_INSTALL_LIBEXECDIR STREQUAL "libexec" OR NOT CMAKE_INSTALL_DATADIR STREQUAL "share")
    message(FATAL_ERROR "SCV engines require exact libexec/share receipt layout")
endif()
get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_LIST_DIR}/.." ABSOLUTE)
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/SymphonyInstallReceiptV2.cmake")
option(SYMPHONY_KVE_USE_INSTALLED "Use an installed exact compatible C++ foundation" OFF)
if(SYMPHONY_KVE_USE_INSTALLED)
    find_package(SymphonyKnowledgeVectorEngine 0.1 REQUIRED CONFIG)
elseif(NOT TARGET Symphony::KnowledgeVectorEngine)
    set(SYMPHONY_KVE_BUILD_TESTING OFF CACHE BOOL "Build foundation tests separately" FORCE)
    add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp"
        "${CMAKE_CURRENT_BINARY_DIR}/knowledge-vector-engine-cpp" EXCLUDE_FROM_ALL)
endif()

set(SCV_SOURCE_DIR "${SYMPHONY_REPOSITORY_ROOT}/modules/scv-engine")
set(SCV_MODULE "${SYMPHONY_SCV_DOMAIN}-engine")
set(SCV_EXECUTABLE "symphony-${SYMPHONY_SCV_DOMAIN}")
set(SCV_VERSION "${PROJECT_VERSION}-dev")
set(SCV_RUNTIME "libexec/symphony/${SCV_MODULE}/${SCV_VERSION}")
set(SCV_DOCS "share/doc/symphony/${SCV_MODULE}/${SCV_VERSION}")
set(SCV_SCHEMAS "share/symphony/schemas/${SCV_MODULE}/${SCV_VERSION}")
include("${SYMPHONY_REPOSITORY_ROOT}/cmake/ScvInterface.generated.cmake")
if(NOT SCV_VERSION STREQUAL SCV_INTERFACE_VERSION)
    message(FATAL_ERROR "SCV package version differs from its owner interface")
endif()
set(SCV_SCHEMA_FILES ${SCV_INTERFACE_SCHEMA_FILES})
set(SCV_CONTRACTS "share/symphony/contracts/${SCV_MODULE}/${SCV_VERSION}")
set(SCV_COMPANION_OWNED "")
set(SCV_COMPANION_INSTALLED "")
foreach(companion IN LISTS SCV_INTERFACE_COMPANION_FILES)
    get_filename_component(companion_name "${companion}" NAME)
    list(APPEND SCV_COMPANION_OWNED "${SCV_DOCS}/${companion_name}|regular")
    list(APPEND SCV_COMPANION_INSTALLED "${SCV_DOCS}/${companion_name}")
endforeach()
set(SCV_SCHEMA_OWNED "")
set(SCV_SCHEMA_INSTALLED "")
foreach(schema IN LISTS SCV_SCHEMA_FILES)
    get_filename_component(schema_name "${schema}" NAME)
    list(APPEND SCV_SCHEMA_OWNED "${SCV_SCHEMAS}/${schema_name}|regular")
    list(APPEND SCV_SCHEMA_INSTALLED "${SCV_SCHEMAS}/${schema_name}")
endforeach()
set(SCV_LICENSES "share/licenses/symphony-${SCV_MODULE}/${SCV_VERSION}")

if(NOT TARGET scv-domain-core)
    add_library(scv-domain-core STATIC "${SCV_SOURCE_DIR}/src/source.cpp" "${SCV_SOURCE_DIR}/src/knowledge.cpp" "${SCV_SOURCE_DIR}/src/corpus.cpp" "${SCV_SOURCE_DIR}/src/interpretation.cpp" "${SCV_SOURCE_DIR}/src/coverage.cpp" "${SCV_SOURCE_DIR}/src/pack.cpp" "${SCV_SOURCE_DIR}/src/composition.cpp" "${SCV_SOURCE_DIR}/src/dispatch.cpp")
    target_include_directories(scv-domain-core PUBLIC "${SCV_SOURCE_DIR}/src")
    target_link_libraries(scv-domain-core PUBLIC Symphony::KnowledgeVectorEngine)
    target_compile_definitions(scv-domain-core PUBLIC SYMPHONY_SCV_VERSION="${SCV_VERSION}")
    set_target_properties(scv-domain-core PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(scv-domain-core PRIVATE $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
endif()
add_executable(${SCV_EXECUTABLE} "${SCV_SOURCE_DIR}/src/main.cpp")
target_link_libraries(${SCV_EXECUTABLE} PRIVATE scv-domain-core)
target_compile_definitions(${SCV_EXECUTABLE} PRIVATE SYMPHONY_SCV_DOMAIN="${SYMPHONY_SCV_DOMAIN}")
set_target_properties(${SCV_EXECUTABLE} PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
target_compile_options(${SCV_EXECUTABLE} PRIVATE $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)

if(BUILD_TESTING)
    find_package(Python3 COMPONENTS Interpreter REQUIRED)
    if(SYMPHONY_SCV_DOMAIN STREQUAL "scv")
        add_executable(scv-source-tests "${SCV_SOURCE_DIR}/tests/source_test.cpp")
        target_link_libraries(scv-source-tests PRIVATE scv-domain-core)
        set_target_properties(scv-source-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
        add_test(NAME scv-source-tests COMMAND scv-source-tests)
        add_executable(scv-knowledge-tests "${SCV_SOURCE_DIR}/tests/knowledge_test.cpp")
        target_link_libraries(scv-knowledge-tests PRIVATE scv-domain-core)
        set_target_properties(scv-knowledge-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
        add_test(NAME scv-knowledge-tests COMMAND scv-knowledge-tests)
        add_executable(scv-corpus-tests "${SCV_SOURCE_DIR}/tests/corpus_test.cpp")
        target_link_libraries(scv-corpus-tests PRIVATE scv-domain-core)
        set_target_properties(scv-corpus-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
        add_test(NAME scv-corpus-tests COMMAND scv-corpus-tests)
        add_executable(scv-interpretation-tests "${SCV_SOURCE_DIR}/tests/interpretation_test.cpp")
        target_link_libraries(scv-interpretation-tests PRIVATE scv-domain-core)
        set_target_properties(scv-interpretation-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
        add_test(NAME scv-interpretation-tests COMMAND scv-interpretation-tests)
        add_executable(scv-coverage-tests "${SCV_SOURCE_DIR}/tests/coverage_test.cpp")
        target_link_libraries(scv-coverage-tests PRIVATE scv-domain-core)
        set_target_properties(scv-coverage-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
        add_test(NAME scv-coverage-tests COMMAND scv-coverage-tests)
        foreach(area pack composition obligation)
            add_executable(scv-${area}-tests "${SCV_SOURCE_DIR}/tests/${area}_test.cpp")
            target_link_libraries(scv-${area}-tests PRIVATE scv-domain-core)
            set_target_properties(scv-${area}-tests PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
            add_test(NAME scv-${area}-tests COMMAND scv-${area}-tests)
        endforeach()
        add_test(NAME scv-interface-generation-tests COMMAND ${Python3_EXECUTABLE}
            "${SCV_SOURCE_DIR}/tests/interface_generation_test.py")
    endif()
    add_test(NAME scv-installed-process-integration COMMAND ${Python3_EXECUTABLE}
        "${SCV_SOURCE_DIR}/tests/installed_integration.py" --build "${CMAKE_CURRENT_BINARY_DIR}" --domain "${SYMPHONY_SCV_DOMAIN}" --version "${SCV_VERSION}")
endif()

symphony_install_receipt_v2_preflight(RECEIPT_PATH "share/symphony/receipts/${SCV_MODULE}/${SCV_VERSION}/install-receipt.json")
install(TARGETS ${SCV_EXECUTABLE} RUNTIME DESTINATION "${SCV_RUNTIME}")
install(FILES ${SCV_SCHEMA_FILES} DESTINATION "${SCV_SCHEMAS}")
install(FILES ${SCV_INTERFACE_COMPANION_FILES} DESTINATION "${SCV_DOCS}")
install(FILES ${SCV_INTERFACE_MANIFEST} DESTINATION "${SCV_CONTRACTS}")
install(FILES "${CMAKE_CURRENT_SOURCE_DIR}/INTENT.md" "${CMAKE_CURRENT_SOURCE_DIR}/MANIFEST.md"
    "${CMAKE_CURRENT_SOURCE_DIR}/INSTALL.md" "${CMAKE_CURRENT_SOURCE_DIR}/SKILL.md" "${CMAKE_CURRENT_SOURCE_DIR}/SPEC.md"
    DESTINATION "${SCV_DOCS}")
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/LICENSE" DESTINATION "${SCV_LICENSES}" RENAME LICENSE-AGPL-3.0)
install(FILES "${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp/third_party/nlohmann/LICENSE.MIT"
    DESTINATION "${SCV_LICENSES}" RENAME nlohmann-json-LICENSE.MIT)
symphony_install_receipt_v2(
    COMPONENT_ID "${SCV_MODULE}" COMPONENT_KIND vector_engine MODULE_ID "${SCV_MODULE}" VECTOR_ID "${SYMPHONY_SCV_DOMAIN}"
    ENGINE_ID "${SCV_EXECUTABLE}" PACKAGE_ID "${SCV_MODULE}" VERSION "${SCV_VERSION}"
    RECEIPT_PATH "share/symphony/receipts/${SCV_MODULE}/${SCV_VERSION}/install-receipt.json"
    OWNED_FILES
        ${SCV_SCHEMA_OWNED}
        ${SCV_COMPANION_OWNED}
        "${SCV_CONTRACTS}/OWNER-INTERFACE.json|regular"
        "${SCV_RUNTIME}/${SCV_EXECUTABLE}|executable"
        "${SCV_DOCS}/INSTALL.md|regular" "${SCV_DOCS}/INTENT.md|regular" "${SCV_DOCS}/MANIFEST.md|regular"
        "${SCV_DOCS}/SKILL.md|regular" "${SCV_DOCS}/SPEC.md|regular"
        "${SCV_LICENSES}/LICENSE-AGPL-3.0|regular" "${SCV_LICENSES}/nlohmann-json-LICENSE.MIT|regular"
    ENTRY_POINTS "${SCV_EXECUTABLE}|executable|${SCV_RUNTIME}/${SCV_EXECUTABLE}|symphony.knowledge.engine-process.v1"
    COMPATIBLE_RECEPTORS symphony.maestro.knowledge-engine.v1)
configure_file("${SCV_SOURCE_DIR}/cmake/uninstall.cmake.in" "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake" @ONLY)
add_custom_target(uninstall-${SCV_MODULE} COMMAND ${CMAKE_COMMAND} -DINSTALL_PREFIX=${CMAKE_INSTALL_PREFIX}
    -P "${CMAKE_CURRENT_BINARY_DIR}/uninstall.cmake")

# Aggregate development build shares compilation only. Each subdirectory still
# installs and uninstalls its own exact, independently runnable receipt package.
option(SYMPHONY_SCV_BUILD_ALL_DOMAINS "Build and test all supplied SCV domain packages" OFF)
if(SYMPHONY_SCV_DOMAIN STREQUAL "scv" AND SYMPHONY_SCV_BUILD_ALL_DOMAINS)
    foreach(domain IN LISTS SCV_INTERFACE_DOMAINS)
        if(domain STREQUAL "scv")
            continue()
        endif()
        add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/modules/${domain}-engine" "${CMAKE_CURRENT_BINARY_DIR}/domains/${domain}")
    endforeach()
endif()
