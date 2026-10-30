cmake_minimum_required(VERSION 3.21)
project(sentinel_custom_notification LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

option(SENTINEL_QSS_HOT_RELOAD "Enable development QSS reload" OFF)

find_package(Qt6 REQUIRED COMPONENTS Widgets)

set(CMAKE_AUTORCC ON)
add_executable(sentinel-custom-notification custom-notification.cpp custom-notification.qrc)
target_link_libraries(sentinel-custom-notification PRIVATE Qt6::Widgets)

if(SENTINEL_QSS_HOT_RELOAD)
    target_compile_definitions(sentinel-custom-notification PRIVATE
        SENTINEL_QSS_HOT_RELOAD
        SENTINEL_CUSTOM_NOTIFICATION_STYLESHEET="${CMAKE_CURRENT_SOURCE_DIR}/custom-notification.qss"
    )
endif()

if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    find_package(LayerShellQt REQUIRED)
    target_link_libraries(sentinel-custom-notification PRIVATE LayerShellQt::Interface)
endif()

# Stylesheet and fallback artwork are embedded for data-driven delivery.
set_target_properties(sentinel-custom-notification PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)
