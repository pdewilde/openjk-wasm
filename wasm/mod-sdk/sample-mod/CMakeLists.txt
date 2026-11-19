cmake_minimum_required(VERSION 3.16)
project(openjk_wasm_sample_mod C)

if(NOT EMSCRIPTEN)
	message(FATAL_ERROR "This sample mod targets Emscripten WebAssembly (use emcmake cmake or CMakePresets.json).")
endif()

# Enable Emscripten dynamic SIDE_MODULE=2 linking in CMake
set_property(GLOBAL PROPERTY TARGET_SUPPORTS_SHARED_LIBS TRUE)
set(CMAKE_SHARED_LIBRARY_SUFFIX ".wasm")
set(CMAKE_SHARED_LIBRARY_CREATE_C_FLAGS "-sSIDE_MODULE=2 -msimd128 -fwasm-exceptions")
set(CMAKE_SHARED_LIBRARY_CREATE_CXX_FLAGS "-sSIDE_MODULE=2 -msimd128 -fwasm-exceptions")
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# Path to the OpenJK repository root (contains codemp/ and shared/)
set(OPENJK_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../../.." CACHE PATH "Path to OpenJK source root")

add_library(cgamewasm32 SHARED sample_cgame.c)

set_target_properties(cgamewasm32 PROPERTIES
	PREFIX ""
	OUTPUT_NAME "cgamewasm32"
	SUFFIX ".wasm"
)

target_include_directories(cgamewasm32 PRIVATE
	"${OPENJK_ROOT}/codemp"
	"${OPENJK_ROOT}/shared"
)

target_compile_definitions(cgamewasm32 PRIVATE
	_CGAME
	ARCH_STRING="wasm32"
)

# Exact OpenJK WebAssembly SIDE_MODULE=2 ABI flags (must match root CMakeLists.txt)
target_compile_options(cgamewasm32 PRIVATE
	-O2
	-fPIC
	-msimd128
	-fwasm-exceptions
	-fvisibility=hidden
)

target_link_options(cgamewasm32 PRIVATE
	-sSIDE_MODULE=2
	-sEXPORTED_FUNCTIONS=_GetModuleAPI
	-msimd128
	-fwasm-exceptions
)
