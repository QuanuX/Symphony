cmake_minimum_required(VERSION 3.30)
project(SymphonySnvCommon VERSION 0.1.0 LANGUAGES CXX)
include(../../cmake/SymphonySnvEngine.cmake)
symphony_snv_engine(MECHANICS OWNER snv-common SDK SnvCommon SOURCES src/common.cpp)
