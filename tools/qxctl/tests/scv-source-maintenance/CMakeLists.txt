cmake_minimum_required(VERSION 3.25)
project(SymphonyScvSourceMaintenanceAcceptance LANGUAGES CXX)
get_filename_component(SYMPHONY_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../../../.." ABSOLUTE)
if(NOT TARGET Symphony::KnowledgeVectorEngine)
    set(SYMPHONY_KVE_BUILD_TESTING OFF CACHE BOOL "Build foundation tests separately" FORCE)
    add_subdirectory("${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp"
        "${CMAKE_CURRENT_BINARY_DIR}/knowledge-vector-engine-cpp" EXCLUDE_FROM_ALL)
endif()
foreach(campaign pack retention semantics authority)
    add_executable(scv-verify-${campaign} "${CMAKE_CURRENT_SOURCE_DIR}/verify_${campaign}.cpp")
    target_include_directories(scv-verify-${campaign} PRIVATE
        "${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp/tests/support"
        "${SYMPHONY_REPOSITORY_ROOT}/modules/scv-graph-duckdb-connector/tests")
    target_link_libraries(scv-verify-${campaign} PRIVATE Symphony::KnowledgeVectorEngine ${CMAKE_DL_LIBS})
    set_target_properties(scv-verify-${campaign} PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
    target_compile_options(scv-verify-${campaign} PRIVATE
        $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
endforeach()
# These are explicit-input acceptance campaigns. Their caller selects exact
# installed owners, retained fixture files and a fresh private output directory.
