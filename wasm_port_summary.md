# OpenJK WASM Port Summary

## Goal
The objective is to port the OpenJK Jedi Academy single-player engine to WebAssembly via Emscripten, enabling browser-based gameplay. **Milestone 1** (achieving a clean, monolithic compilation and linkage) is now **complete**.

## Key Architectural Changes

1. **Monolithic Build Strategy**
   The original desktop engine relied on dynamically loading modules (`rd-vanilla`, `jagame`, `cgame`, `ui`) as shared libraries (`.dll`/`.so`). In the WebAssembly environment, dynamic linking via Emscripten (`dlopen`) is complex and degrades performance.
   - **Action:** We created a standalone `wasm/CMakeLists.txt` build configuration to compile the core engine, renderer, game code, UI, and CGame code seamlessly into a single static WebAssembly binary (`openjk.wasm`).
   - **Action:** We stubbed out the engine's dynamic library loader (`Sys_LoadDll`, `Sys_LoadGameDll`, etc.) using `#ifdef __EMSCRIPTEN__` blocks. Instead of loading paths from disk, the engine now returns explicit pointers to the statically linked APIs directly (`GetRefAPI`, `GetGameAPI`, `CG_vmMain`, `UI_vmMain`).

2. **Resolving Global Symbol Conflicts**
   Because all previously isolated modules now reside in the same global address space, the linker surfaced severe symbol collisions between C/C++ globals.
   - **Action:** Refactored various global arrays (like `animTable`) and module-specific main hooks (`vmMain`, `dllEntry` specifically for the `cgame` and `ui` pipelines) to avoid definition overlapping.
   - **Action:** Standardized and wrapped specific crossover points using `extern "C"` linkage blocks to prevent C++ name mangling inconsistencies when passing statically defined hook functions across C/C++ subsystem bounds.

3. **OpenGL Compatibility via GL4ES**
   The Jedi Academy single-player renderer (`rd-vanilla`) heavily relies on legacy, fixed-function OpenGL 1.x operations such as `glBegin`, `glEnd`, and display list compilations (`glNewList`, `glEndList`). WebGL 2.0 (GLES 3.0) intrinsically lacks support for these immediate mode calls.
   - **Action:** We evaluated Emscripten's built-in `Regal` compatibility layer, but verified it was fundamentally missing `display list` mechanisms relied upon heavily by the engine.
   - **Action:** We opted to integrate **GL4ES**—a highly accurate and well-maintained OpenGL 1.x to GLES 2.0 translation layer. We cloned it into `lib/gl4es`, tailored its source code by disabling the missing `X11`/`GLX` hooks, and built it into a static wrapper (`libGL.a`).
   - **Action:** Configured `qgl.h` (the engine's OpenGL macro definition table) and the CMake linker to transparently bridge all immediate-mode rendering calls safely through the GL4ES emulator layer without touching the vintage renderer's original behavior. All `gl*` linker errors successfully vanished.

4. **Emscripten Core Loop Hooking**
   - **Action:** A traditional desktop PC runs an infinite blocking loop: `while(true) { Com_Frame(); }`. For a browser execution context, this immediately crashes the Javascript thread. We intercepted the primary loop inside `sys_main.cpp` to yield cleanly using Emscripten's `emscripten_set_main_loop(Com_Frame, 0, 1)`, letting the browser gracefully handle drawing intervals.

5. **Virtual File System (VFS) and Assets**
   - **Action:** Implemented Emscripten's `--preload-file` packaging to map the large `base/` directory and `.pk3` files into the WASM virtual filesystem, allowing `FS_Startup` to succeed and load the game assets correctly without modifying the engine's I/O layer.

6. **Input and Pointer Lock**
   - **Action:** Wired up SDL2's input handling through Emscripten. We added `#canvas` element targeting explicitly for Emscripten's keyboard and mouse hooks, ensuring that clicking the game properly captures pointer lock and correctly feeds relative mouse movements to the game camera.

7. **Audio and VSync Pacing**
   - **Action:** Enabled Emscripten's built-in OpenAL emulation (`-s USE_OPENAL=1`) which intercepts the engine's OpenAL calls and implements them over Web Audio API, restoring music and sound effects.
   - **Action:** Fixed severe main loop stuttering by forcing `LIBGL_VSYNC=1` in the GL4ES environment variables and `SDL_GL_SetSwapInterval(1)`, which instructs Emscripten to bypass legacy thread-sleeping mechanisms and rely on `requestAnimationFrame` for buttery-smooth pacing.

## Next Steps

With the monolithic binary compiling, rendering, and actively playable in the browser, the next immediate goals are:
- Fixing the canvas and resolution cutoff issues, particularly in menus (e.g., lightsaber configuration) where the game attempts to switch internal `r_mode` resolutions and doesn't map perfectly to the browser viewport.
- Setting up persistent file storage (e.g., IndexedDB / IDBFS) for save games and configuration files.
