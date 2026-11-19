# OpenJK WebAssembly MP Mod SDK (`SIDE_MODULE=2`)

OpenJK WebAssembly loads Multiplayer mods using the exact same `fs_game` search order as native OpenJK:

1. **Asset / script mods (`.pk3`, `.cfg`, `description.txt`)** work out of the box with zero recompilation — drag and drop a `.zip` or `.pk3` in the browser Launcher's **MP Mod Manager** (`index.html`), or enable **server auto-download** (`+set cl_allowDownload 1`).
2. **Native C/C++ code mods (`*wasm32.wasm`)** are compiled with Emscripten as WebAssembly dynamic side modules (`-sSIDE_MODULE=2`) and loaded dynamically by `Sys_LoadGameDll` / `Sys_LoadLegacyGameDll` from `/saves/<fs_game>/` or `/<fs_game>/` before falling back to `/base/`.

---

## Side Module Filenames

| MP Module | Filename | Exported Entry Points |
|-----------|----------|-----------------------|
| Client Game | `cgamewasm32.wasm` | `GetModuleAPI` (OpenJK) or `dllEntry` + `vmMain` (legacy JKA 1.01 SDK) |
| Server Game | `jampgamewasm32.wasm` | `GetModuleAPI` (OpenJK) or `dllEntry` + `vmMain` (legacy JKA 1.01 SDK) |
| User Interface | `uiwasm32.wasm` | `GetModuleAPI` (OpenJK) or `dllEntry` + `vmMain` (legacy JKA 1.01 SDK) |

Partial-module mods are supported: if your mod only ships `cgamewasm32.wasm`, OpenJK automatically falls back to `/base/jampgamewasm32.wasm` and `/base/uiwasm32.wasm`.

---

## Required Emscripten ABI Flags

Your mod's `.wasm` side module **must** enable shared library support in CMake and compile with the same WebAssembly feature & exception ABI as `openjk.wasm32`:

```cmake
set_property(GLOBAL PROPERTY TARGET_SUPPORTS_SHARED_LIBS TRUE)
set(CMAKE_SHARED_LIBRARY_SUFFIX ".wasm")
set(CMAKE_SHARED_LIBRARY_CREATE_C_FLAGS "-sSIDE_MODULE=2 -msimd128 -fwasm-exceptions")
set(CMAKE_SHARED_LIBRARY_CREATE_CXX_FLAGS "-sSIDE_MODULE=2 -msimd128 -fwasm-exceptions")
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

add_library(cgamewasm32 SHARED ...)
set_target_properties(cgamewasm32 PROPERTIES PREFIX "" OUTPUT_NAME "cgamewasm32" SUFFIX ".wasm")

target_compile_options(cgamewasm32 PRIVATE
    -O2
    -fPIC
    -msimd128
    -fwasm-exceptions
    -fvisibility=hidden
)

target_link_options(cgamewasm32 PRIVATE
    -sSIDE_MODULE=2
    -sEXPORTED_FUNCTIONS=_GetModuleAPI,_dllEntry,_vmMain
    -msimd128
    -fwasm-exceptions
)
```

Mark exported entry points with `Q_EXPORT` (`__attribute__((visibility("default")))`):

- **Modern OpenJK API:** `Q_EXPORT cgameExport_t *QDECL GetModuleAPI(int apiVersion, cgameImport_t *import)`
- **Legacy JKA 1.01 SDK:** `Q_EXPORT void QDECL dllEntry(...)` and `Q_EXPORT intptr_t QDECL vmMain(...)` (OpenJK's `VM_CreateLegacy` fallback automatically detects legacy modules when `GetModuleAPI` is absent).

---

## Quick Start: Build & Test the Sample Mod

1. Activate Emscripten (`source /path/to/emsdk/emsdk_env.sh`).
2. Build and package `sample-mod`:
   ```bash
   bash wasm/mod-sdk/build_mod.sh
   ```
   This produces `/tmp/openjk-wasm-samplemod/samplemod/` (`cgamewasm32.wasm`, `samplemod_assets.pk3`, `description.txt`) and `/tmp/openjk-wasm-samplemod/samplemod.zip`.
3. Open `index.html` in your browser, drag and drop `samplemod.zip` into the **Multiplayer Mods** section, enable `samplemod`, and click **Play Multiplayer**.
4. Load any map (e.g. `+map mp/ffa1`) and check the console (`Ctrl + \``):
   - You will see `[SampleMod-WASM] Custom SIDE_MODULE=2 cgamewasm32.wasm initialized!`
   - Run `/samplemod` in the in-game console to execute the native C mod command.

---

## URL Launch Parameters

You can link directly to any installed mod or server via `openjk.wasm32.html`:

- `?mod=<fs_game>` — launches with `+set fs_game <fs_game>`
- `?map=<mapname>` — launches with `+map <mapname>`
- `?connect=<host:port>` — launches with `+connect <host:port>`
- `?allowdownload=1` — enables `+set cl_allowDownload 1` (auto-downloaded `.pk3` files persist in OPFS under `/saves/<fs_game>/`)
- `?args=<raw_args>` — appends arbitrary OpenJK command-line arguments

---

## Known Limitations

- **Closed-source native x86/ARM mods (`.dll`, `.so`, `.dylib`) cannot execute in the browser:** Legacy mods distributed only as precompiled desktop binaries (such as **JA+ / japlus** `jampgamex86.dll` / `cgamex86.dll` / `uix86.dll` / `jampgamei386.so`) cannot run their native C/C++ game logic in WebAssembly without being recompiled from source as `SIDE_MODULE=2` `.wasm` modules.
  - Asset-only `.pk3` files bundled in those archives (maps, skins, sabers, animations, effects) are still hoisted and loaded normally, and the launcher flags such packages with an **Assets Only (x86 code)** badge.
  - Open-source C++ mods (such as JA++ or other OpenJK-compatible forks) can be compiled directly to `*wasm32.wasm` using the ABI flags above.
  - Emulating or JIT-compiling x86-32 `.dll` binaries inside WebAssembly is not currently implemented (an interpreter would be too slow for per-frame `cgame`, and a full x86-to-Wasm JIT is out of scope).
