#============================================================================
# GL4ES static library build for Emscripten (libGL.a)
#============================================================================
include(ExternalProject)

set(GL4ES_SOURCE_DIR "${CMAKE_SOURCE_DIR}/lib/gl4es")
set(GL4ES_BINARY_DIR "${CMAKE_BINARY_DIR}/gl4es-build")
set(GL4ES_INSTALL_LIB_DIR "${CMAKE_BINARY_DIR}/gl4es-lib")
set(GL4ES_ARCHIVE_IN_SOURCE "${GL4ES_SOURCE_DIR}/lib/libGL.a")
set(GL4ES_ARCHIVE "${GL4ES_INSTALL_LIB_DIR}/libGL.a")

set(GL4ES_C_FLAGS "-fPIC -msimd128 -O2")
if(WasmUsePthreads)
	string(APPEND GL4ES_C_FLAGS " -pthread")
endif()

ExternalProject_Add(gl4es_external
	SOURCE_DIR "${GL4ES_SOURCE_DIR}"
	BINARY_DIR "${GL4ES_BINARY_DIR}"
	CMAKE_ARGS
		-DCMAKE_TOOLCHAIN_FILE=${CMAKE_TOOLCHAIN_FILE}
		-DCMAKE_BUILD_TYPE=$<IF:$<CONFIG:Debug>,Debug,Release>
		-DNOX11=ON
		-DNOEGL=ON
		-DSTATICLIB=ON
		-DNO_GBM=ON
		-DDEFAULT_ES=2
		"-DCMAKE_C_FLAGS=${GL4ES_C_FLAGS}"
	BUILD_COMMAND ${CMAKE_COMMAND} --build <BINARY_DIR> --target GL
	INSTALL_COMMAND
		${CMAKE_COMMAND} -E make_directory "${GL4ES_INSTALL_LIB_DIR}"
		COMMAND ${CMAKE_COMMAND} -E copy_if_different "${GL4ES_ARCHIVE_IN_SOURCE}" "${GL4ES_ARCHIVE}"
	BUILD_BYPRODUCTS "${GL4ES_ARCHIVE_IN_SOURCE}" "${GL4ES_ARCHIVE}"
	USES_TERMINAL_BUILD ON
)

add_library(gl4es::GL STATIC IMPORTED GLOBAL)
set_target_properties(gl4es::GL PROPERTIES
	IMPORTED_LOCATION "${GL4ES_ARCHIVE}"
	INTERFACE_INCLUDE_DIRECTORIES "${GL4ES_SOURCE_DIR}/include"
)
add_dependencies(gl4es::GL gl4es_external)
