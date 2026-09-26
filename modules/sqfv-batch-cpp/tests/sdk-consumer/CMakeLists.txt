cmake_minimum_required(VERSION 3.25)
project(IndependentInstalledSqfvCConsumer LANGUAGES C CXX)

find_package(SymphonySqfvBatch 0.1.0 EXACT REQUIRED CONFIG)
if(NOT SymphonySqfvBatch_RELEASE_VERSION STREQUAL "0.1.0-dev")
    message(FATAL_ERROR "unexpected exact SQFV batch release")
endif()

# main.c is compiled as C11; the C++ linker supplies the static archive runtime.
add_executable(consumer main.c)
set_target_properties(consumer PROPERTIES
    C_STANDARD 11 C_STANDARD_REQUIRED ON C_EXTENSIONS OFF LINKER_LANGUAGE CXX)
target_link_libraries(consumer PRIVATE Symphony::SqfvBatch)
