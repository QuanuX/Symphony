cmake_minimum_required(VERSION 3.30)
project(SymphonyReleaseRetainedStoreFixtures LANGUAGES CXX)
find_package(SymphonyKnowledgeVectorEngine 0.2.0 EXACT REQUIRED CONFIG)
find_package(SymphonySqdvDelivery 0.3.0 EXACT REQUIRED CONFIG)
find_package(SymphonySqavDatabentoDbn 0.5.0 EXACT REQUIRED CONFIG)
add_executable(sqv-admin-fixtures fixtures.cpp)
target_link_libraries(sqv-admin-fixtures PRIVATE Symphony::SqdvDelivery
  Symphony::SqavDatabentoDbn Symphony::KnowledgeVectorEngine)
set_target_properties(sqv-admin-fixtures PROPERTIES CXX_STANDARD 26
  CXX_STANDARD_REQUIRED ON CXX_EXTENSIONS OFF)
target_compile_options(sqv-admin-fixtures PRIVATE -Wall -Wextra -Wpedantic -Werror)
