cmake_minimum_required(VERSION 3.30)
project(IndependentInstalledSqfvCppConsumer LANGUAGES CXX)

find_package(SymphonySqfvBatch 0.3.0 EXACT REQUIRED CONFIG)
if(NOT SymphonySqfvBatch_RELEASE_VERSION STREQUAL "0.3.0-dev")
    message(FATAL_ERROR "unexpected exact SQFV batch release")
endif()

# This consumer is copied outside the source tree before configuration.
add_executable(consumer main.cpp)
set_target_properties(consumer PROPERTIES CXX_EXTENSIONS OFF)
target_link_libraries(consumer PRIVATE Symphony::SqfvBatch)
