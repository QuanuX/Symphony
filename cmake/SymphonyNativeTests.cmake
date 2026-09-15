include_guard(GLOBAL)
# Acceptance executables share process mechanics; test expectations remain owned
# by the relevant module and are compiled without a scripting interpreter.
function(symphony_native_test target source)
  add_executable(${target} "${source}")
  target_include_directories(${target} PRIVATE
    "${SYMPHONY_REPOSITORY_ROOT}/libraries/knowledge-vector-engine-cpp/tests/support"
    "${SYMPHONY_REPOSITORY_ROOT}/tools/qxctl/tests/shv-invariants")
  target_link_libraries(${target} PRIVATE Symphony::KnowledgeVectorEngine ${CMAKE_DL_LIBS})
  set_target_properties(${target} PROPERTIES CXX_STANDARD 26 CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
  target_compile_options(${target} PRIVATE $<$<CXX_COMPILER_ID:AppleClang,Clang,GNU>:-Wall;-Wextra;-Wpedantic;-Werror>)
endfunction()
