cmake_minimum_required(VERSION 3.16)
project(open_gallery CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_library(open_gallery
    src/generators/grid.cpp
)

target_include_directories(open_gallery PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/include)
