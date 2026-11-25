cmake_minimum_required(VERSION 3.16)
project(open_gallery CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_library(open_gallery
    src/generators/grid.cpp
)

target_include_directories(open_gallery PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/include)

include(CTest)

if(BUILD_TESTING)
    add_executable(open_gallery_tests
        tests/test_open_gallery.cpp
    )
    target_link_libraries(open_gallery_tests PRIVATE open_gallery)
    add_test(NAME open_gallery_tests COMMAND open_gallery_tests)

    add_executable(open_gallery_grid_tests tests/test_grid.cpp)
    target_link_libraries(open_gallery_grid_tests PRIVATE open_gallery)
    add_test(NAME open_gallery_grid_tests COMMAND open_gallery_grid_tests)
endif()
