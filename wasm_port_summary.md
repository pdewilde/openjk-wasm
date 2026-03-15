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

8. **Resolution and Viewport Scaling**
   - **Action:** Fixed WebGL scissor clipping errors caused by the UI viewport scaling bounds exceeding the browser's Canvas dimensions.
   - **Action:** Patched `vid_restart` sequence to skip `SDL_QuitSubSystem` and `IN_Shutdown` to prevent Emscripten from losing the WebGL context and pointer lock state. We now rely on CSS (`width: 100%; height: 100%; object-fit: contain`) to upscale the underlying viewport to fill the screen rather than risking a context drop.

## Next Steps

With the monolithic binary compiling, rendering, and actively playable in the browser, the following singleplayer engine integration fixes have also been implemented:
- **Persistent Filesystem / Saves:** Created a virtual `/saves` directory initialization path inside Emscripten to allow the VFS to natively support writing checkpoint and savegame data.

Current ongoing singleplayer work:
- Continuing to isolate and debug remaining ICARUS scripting anomalies that manifest specifically after checkpoint restoration (e.g. `tree3` sequenced events failing to run properly).

## Multiplayer Port Context
The Multiplayer WebAssembly port is currently underway. Key achievements and ongoing challenges include:

1. **Directory Structure:** Singleplayer code resides in `code/`, but the Multiplayer source tree resides in `codemp/`.
2. **Build Strategy:** The build has been configured in `wasm/CMakeLists.txt` to statically link the core engine, renderer (`rd-vanilla`), `cgame`, `ui`, and `qagame` seamlessly into a monolithic binary (`openjk_mp.wasm`).
3. **Dynamic Library Interception:** Dynamic module loading (`Sys_LoadDll`) has been stubbed out using `#ifdef __EMSCRIPTEN__` blocks, wiring them directly to the statically linked APIs (`GetRefAPI`, `GetGameAPI`, `CG_vmMain`, `UI_vmMain`).
4. **Main Loop & Graphics:** Emscripten's `emscripten_set_main_loop` is used to prevent thread blocking. OpenGL legacy translation via `GL4ES` and Web Audio `OpenAL` emulation are applied.
5. **Symbol Conflicts & State Collisions:**
   - **Issue:** The `cgame` and `ui` modules both relied on a shared file (`ui_shared.c`) which contained core menu system logic, including global variables like the `Menus` array and `menuCount`. In a monolithic build, they shared the exact same memory space, causing `cgame` to wipe out the UI menus upon initialization.
   - **Resolution:** `ui_shared.c` was added to the `mp_cgame` build target so it has its own copy. A Python script (`gen_renames.py`) was used to extract all global symbols from `ui_shared.c` and add them to `ui_rename.h` and `cg_rename.h`. This ensures the `ui` module uses `ui_Menus` and the `cgame` module uses `cg_Menus`, completely isolating their state.
6. **Memory Allocation:** The initial memory allocation for the WASM build was increased from 64MB to 128MB (`-sINITIAL_MEMORY=134217728`) to accommodate the duplicated state and prevent linking errors.
7. **Networking & WebTransport MTU:**
   - **Issue:** Browsers do not support raw UDP sockets. We use a Go-based WebTransport tunnel (`wasm/wtunnel`) to proxy traffic. However, WebTransport Datagrams have a strict ~1200 byte MTU, while the engine's default `MAX_PACKETLEN` is 1400 bytes.
   - **Current Resolution:** We reduced `MAX_PACKETLEN` to 1150 bytes in `codemp/qcommon/net_chan.cpp`. This forces the WASM client to fragment messages earlier, allowing them to fit inside WebTransport Datagrams.
   - **Future Implementation:** This "easy way" breaks compatibility with standard (non-WASM) servers, which will still send 1400-byte packets that the tunnel will drop. To support standard servers in the future, the Go tunnel must be updated to inspect incoming UDP packet sizes. Packets < 1200 bytes can be sent as Datagrams, while packets >= 1200 bytes must be sent over reliable WebTransport Unidirectional Streams (which the browser will automatically fragment and reassemble), bypassing the MTU limit without modifying the standard server.

8. **WebTransport Client Implementation:**
   - **Action:** Implemented the WebTransport client directly in `codemp/qcommon/net_ip.cpp` using Emscripten's `EM_JS` macro. This bypasses the standard BSD socket calls (`socket`, `sendto`, `recvfrom`) for the WASM build.
   - **Action:** The client connects to the Go-based WebTransport tunnel (`wasm/wtunnel`) using a relative URL (`/wt-server-certificate-hashes`) to fetch certificate hashes, ensuring compatibility with both HTTP and HTTPS.
   - **Action:** It reads a 4-byte Virtual IP (VIP) from the server's incoming unidirectional stream.
   - **Action:** Outgoing packets are prepended with a custom 10-byte `youdp` header (DestIP, SrcIP=VIP, DestPort) and sent as WebTransport datagrams. Incoming datagrams have this header stripped before being passed to the game engine.

9. **WebTransport Tunnel Server (Go):**
   - **Action:** Fixed critical session management bugs by correctly calling `webtransport.ConfigureHTTP3Server` and using `wtServer.ListenAndServe()`.
   - **Action:** Switched the server to use HTTPS (TLS) for both the static file server and the WebTransport endpoint. This provides the **Secure Context** required by modern browsers to enable the `WebTransport` API.
   - **Action:** Integrated mDNS support via `github.com/hashicorp/mdns` to advertise the server as `openjk.local`, allowing direct connections to the WSL IP and bypassing unreliable UDP port-forwarding on `localhost`.
   - **Action:** Fixed a potential panic by ensuring the internal routing table is properly initialized on startup.
   - **Action:** Optimized the build system (`wasm/CMakeLists.txt`) to ensure the WebTransport test page is updated in the distribution directory on every build.

10. **Asset Loading Strategies:**
    - **Action:** Created two distinct build targets for both Singleplayer and Multiplayer:
        - **Bundled Version:** Includes the full `base/` directory using `--preload-file`. Access is restricted via **Basic Auth** (`jedi` / `academy`) in the Go server to protect copyrighted assets.
        - **Upload Version:** Does not include assets. Instead, it features a custom HTML shell (`wasm/shell_upload.html`) with a file picker that allows users to select their local `base/` folder. The files are then mounted into the WASM virtual filesystem at runtime using the Emscripten `FS` API.
    - **Action:** Fixed Emscripten filesystem initialization issues in the Upload Version by explicitly exporting `callMain` and `FS` methods (`-sEXPORTED_RUNTIME_METHODS=['FS','run','callMain']`). The shell now reads uploaded files into memory, mounts them directly into the virtual filesystem, and invokes `Module.callMain()` to start the engine without re-initializing the runtime and losing the mounted assets.
    - **Action:** Updated `wasm/CMakeLists.txt` to produce `openjk_bundled.html`, `openjk_upload.html`, `openjk_mp_bundled.html`, and `openjk_mp_upload.html`.
