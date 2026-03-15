# OpenJK WebAssembly (Emscripten) Port

This directory contains the CMake configuration and HTML shell for compiling the OpenJK Jedi Academy single-player engine into WebAssembly (WASM).

## Prerequisites

1.  **Emscripten SDK (emsdk):** You must have Emscripten installed and activated in your environment.
    ```bash
    git clone https://github.com/emscripten-core/emsdk.git
    cd emsdk
    ./emsdk install latest
    ./emsdk activate latest
    source ./emsdk_env.sh
    ```
2.  **Python:** You will need a modern version of Python 3.12 or higher installed. This is required for the Emscripten SDK.
3.  **Jedi Academy Game Assets:** You must own a legal copy of *Star Wars: Jedi Knight: Jedi Academy* (Steam, GOG, etc.). The engine requires the original game assets to run.

## Setup & Compilation

### 1. GameData Setup
The build process uses Emscripten's `--preload-file` packager to bundle the game assets into an `openjk.data` file that the browser can load.

You need to place or symlink your `base` folder into a `GameData` directory at the root of the OpenJK repository.

```bash
cd openjk-wasm
mkdir -p GameData
# EITHER copy your assets:
cp -r /path/to/your/Jedi\ Academy/GameData/base ./GameData/
# OR symlink your assets (recommended to save space):
ln -s /path/to/your/Jedi\ Academy/GameData/base ./GameData/base
```

### 2. Prepare GL4ES
The renderer uses a translation layer called GL4ES to convert legacy OpenGL 1.x calls into WebGL 2.0.
Make sure the submodule is cloned and built (if required by your workflow), though the internal CMake should statically link against it. Ensure you have run:
```bash
git submodule update --init --recursive
```

### 3. Build the Engine
Create a build directory for the WASM port and run the Emscripten CMake wrappers:

```bash
mkdir build-wasm
cd build-wasm

# Configure the project using the wasm-specific CMakeLists
emcmake cmake ../wasm

# Compile (adjust -j4 based on your CPU cores)
emmake make -j4
```

This will produce several files in your `build-wasm/dist` directory, including:
*   `openjk_bundled.html` / `openjk_mp_bundled.html` (Bundled versions with preloaded assets)
*   `openjk_upload.html` / `openjk_mp_upload.html` (Lightweight versions requiring manual asset upload)
*   `*.js`, `*.wasm`, and `*.data` files for the respective targets.

*Note: Packaging the `.data` files for the bundled versions can take a few minutes and will result in very large files (1GB+ depending on mods).*

## Running Locally

Because browsers restrict loading local files via `file://` protocols for security (CORS), you must serve the files over a local HTTP server. We have a built-in Go server that handles this and also provides WebTransport tunneling for multiplayer.

```bash
# From within your build-wasm directory:
emmake make serve
```

To run the server in production with a real Let's Encrypt certificate, you can pass the `-domain` flag to the `wtunnel` binary. This will automatically provision a certificate and bind the server to port 443:
```bash
./wtunnel -dir ./dist -domain play.yourdomain.com
```

1.  Open your web browser (Chrome or Firefox recommended).
2.  Navigate to `https://localhost:8080/` (or your domain if using the `-domain` flag).
    *   *Note: The server uses HTTPS to satisfy WebTransport secure context requirements. You may need to accept the self-signed certificate warning if running locally.*
    *   *Note: The bundled versions are protected by Basic Auth. Use username `jedi` and password `academy`.*
3.  Click anywhere on the cinematic or menu to grant the browser permission to play audio and capture your mouse (pointer lock).
4.  To release your mouse cursor, press the `Escape` key (or whatever key your browser binds to exiting pointer lock).

## Known Limitations

* **Video Resolution Changes:** The in-game `Setup -> Video` menu option to change resolutions does not work. Changing resolutions natively requires destroying the SDL window and recreating it with a new OpenGL context. In WebAssembly, destroying the window permanently drops the HTML Canvas WebGL context, causing a black screen. Because of this, the engine is forced to stretch the underlying canvas to fit the CSS window size, and changing the internal resolution variables is currently a no-op.
