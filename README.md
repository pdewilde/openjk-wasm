# OpenJK WebAssembly (`wasm-v2`)

Play **Star Wars Jedi Knight: Jedi Academy** (Singleplayer & Multiplayer) directly in modern web browsers via Emscripten / WebAssembly.

Built cleanly on top of upstream [JACoders/OpenJK](https://github.com/JACoders/OpenJK). See [README_ORIG.md](README_ORIG.md) for the original upstream desktop OpenJK documentation.

> **Game Assets Required:** Like upstream OpenJK, this project does **not** include copyrighted Jedi Academy assets. You must supply your own `GameData/base/assets*.pk3` files from a copy of Jedi Academy ([Steam](https://store.steampowered.com/app/6020/), [GOG](https://www.gog.com/game/star_wars_jedi_knight_jedi_academy), etc.).

---

## Key Features

- **Modular `SIDE_MODULE=2` Architecture:** Compiles both Singleplayer (`openjk_sp.wasm32` + `jagamewasm32.wasm`) and Multiplayer (`openjk.wasm32` + `jampgamewasm32.wasm`, `cgamewasm32.wasm`, `uiwasm32.wasm`) as dynamically loaded Wasm side modules via `dlopen`.
- **WebGL2 Rendering (GL4ES) & WebAudio (OpenAL):** Hardware-accelerated OpenGL translation to WebGL2, OpenAL audio via Emscripten, WebAssembly SIMD (`-msimd128`), and native Wasm exceptions (`-fwasm-exceptions`).
- **Progressive HTTP `Range` Asset Streaming + OPFS Caching:** Instead of downloading ~1.2 GB of `.pk3` files up front, `wasm/tools/gen_asset_manifest.py` splits a tiny `00_boot.pk3` (~25 MB) + `manifest.json` so the menus boot in seconds while map assets stream on demand and persist in the browser's Origin Private File System (OPFS).
- **Hybrid WebTransport Multiplayer Relay (`wasm/wtunnel`):** Browser-to-browser and browser-to-dedicated-server UDP tunneling over HTTP/3 WebTransport datagrams (with automatic framed unidirectional stream fallback for oversized packets), plus per-IP/global session and rate limiting.
- **First-Class Multiplayer Mod Support:**
  - Browser launcher (`wasm/index.html`) with drag-and-drop `.zip` / `.pk3` / folder import, multi-mod checkboxes, nested `.pk3` hoisting, and an in-page **Network Diagnostics** modal.
  - Persistent server auto-download (`+set cl_allowDownload 1`) saved to OPFS.
  - Native C/C++ Wasm code mod SDK ([wasm/mod-sdk/README.md](wasm/mod-sdk/README.md)).
- **Browser Ergonomics:** Default resolution automatically matches your browser viewport (with live in-game resolution switching via `vid_restart`), real-time GPU brightness slider (`r_gamma` SVG filter), fullscreen Keyboard Lock (`Alt + Enter`), and live shareable URL routing (`?map=...`, `?connect=...`, `?mod=...`).

---

## Building

### Prerequisites

- [Emscripten SDK (`emsdk`)](https://emscripten.org/docs/getting_started/downloads.html) (tested with Emscripten 4.x / 5.x)
- CMake 3.16+ and Ninja / Make
- Python 3 (for asset manifest generation)
- Go 1.22+ (for building `wtunnel`)

### 1. Build WebAssembly Binaries

```bash
source /path/to/emsdk/emsdk_env.sh

emcmake cmake -S . -B build-wasm \
  -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_MP_ENGINE=ON \
  -DBUILD_SP_ENGINE=ON \
  -DBUILD_GAME=ON \
  -DBUILD_CGAME=ON \
  -DBUILD_UI=ON \
  -DBUILD_SP_GAME=ON \
  -DBUILD_DEDICATED=OFF

cmake --build build-wasm -j
```

Stage the built artifacts (`openjk.wasm32.*`, `openjk_sp.wasm32.*`, `*wasm32.wasm`, and `wasm/index.html`) into a web root directory (for example `dist/`):

```bash
mkdir -p dist/base
cp build-wasm/openjk.wasm32.{html,js,wasm} dist/
cp build-wasm/openjk_sp.wasm32.{html,js,wasm} dist/
cp build-wasm/codemp/game/jampgamewasm32.wasm dist/
cp build-wasm/codemp/cgame/cgamewasm32.wasm dist/
cp build-wasm/codemp/ui/uiwasm32.wasm dist/
cp build-wasm/code/game/jagamewasm32.wasm dist/
cp wasm/index.html dist/index.html
```

### 2. Prepare Streaming Assets (Optional but Recommended)

Generate `00_boot.pk3` and `manifest.json` from your Jedi Academy `GameData/base/` folder so clients boot quickly and stream maps on demand:

```bash
python3 wasm/tools/gen_asset_manifest.py \
  --base-dir "/path/to/Jedi Academy/GameData/base" \
  --out-dir dist/base
```

*(If you omit hosted `base/` assets, the browser shell automatically displays a local file/folder picker so players can mount their own `GameData/base/` directly into WasmFS.)*

### 3. Build & Run `wtunnel` (HTTPS + WebTransport Relay)

```bash
cd wasm/wtunnel
go build -o ../../dist/wtunnel .
cd ../..

./dist/wtunnel \
  -dir dist \
  -https-addr 0.0.0.0:8080 \
  -quic-addr 0.0.0.0:8443 \
  -udp-allow 127.0.0.1:29070
```

> **Windows Note:** Set `QUIC_GO_DISABLE_GSO=true` and `QUIC_GO_DISABLE_ECN=true` when running `wtunnel.exe` on Windows if `quic-go` logs UDP socket GSO/ECN errors.

Open **`https://localhost:8080/`** in Chrome, Edge, or Firefox (accept the local self-signed certificate for development; pass `-domain your.domain.com` for automatic Let's Encrypt TLS in production).

#### Key `wtunnel` Security & Rate-Limit Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-auth-user` / `-auth-pass` | *(empty — disabled)* | Optional HTTP Basic Auth (or `WTUNNEL_AUTH_USER` / `WTUNNEL_AUTH_PASS`) protecting hosted `/base/*` assets; unauthenticated visitors automatically fall back to the local `base/` folder picker without a login popup, while authorized users sign in at `/base/login` |
| `-udp-allow` | *(empty — deny egress)* | Comma-separated allowlist of `host:port` / `CIDR:port` UDP destinations (e.g. dedicated JKA servers) |
| `-max-sessions-per-ip` | `128` | Maximum concurrent WebTransport sessions per client IP (safe for LAN parties / NAT) |
| `-max-sessions` | `2048` | Global concurrent WebTransport session cap |
| `-rate-limit-per-ip` | `10` | Token-bucket handshake rate limit (requests/sec per IP) |
| `-rate-burst-per-ip` | `40` | Token-bucket burst capacity per IP |

---

## Multiplayer Mods

- **Asset & Map Mods (`.zip`, `.pk3`, folders):** Drop one or more `.zip` archives, `.pk3` files, or mod folders onto `index.html`, check whichever mods you want active, and click **Play Multiplayer**. Checked mods are mounted simultaneously alongside the base game.
- **Native Code Mods (`*wasm32.wasm`):** Open-source C/C++ mods can be compiled to WebAssembly `SIDE_MODULE=2` binaries using the SDK and sample mod in [wasm/mod-sdk/README.md](wasm/mod-sdk/README.md).
- **URL Launch Parameters:** `openjk.wasm32.html` supports `?mod=<fs_game>`, `?map=<mapname>`, `?connect=<host:port>`, `?allowdownload=1`, and `?args=<raw_args>`. Joining a map or remote server automatically updates the browser URL bar for easy sharing.

---

## Controls & Browser Tips

- **`Alt + Enter`**: Toggle fullscreen. In Chrome and Edge, fullscreen activates the Keyboard Lock API so `Ctrl+W`, `Ctrl+R`, `Tab`, and `Esc` are sent directly to the game. Hold **`Esc`** to exit fullscreen.
- **`Ctrl + \``**: Toggle the overlay engine log console.
- **Saves & Configs:** Singleplayer saves, multiplayer configs (`openjk.cfg` / `openjk_sp.cfg`), and downloaded mods persist locally in your browser's OPFS (`openjk-saves` and `openjk-mods`).

---

## Known Issues & Limitations

- **Closed-source desktop code mods (`.dll`, `.so`, `.dylib`) cannot execute in WebAssembly:** Legacy mods that distribute only precompiled x86/macOS native binaries (such as **JA+ / japlus** `cgamex86.dll`, `uix86.dll`, `jampgamex86.dll`, `jampgamei386.so`) cannot run their custom C/C++ game logic in the browser without being compiled from source to `SIDE_MODULE=2` `.wasm` modules. Their `.pk3` assets (maps, sabers, skins, animations, effects) still load, and the launcher flags those packages with an **Assets Only (x86 code)** badge. Emulating or JIT-compiling x86 DLLs inside Wasm is not implemented due to interpreter overhead in per-frame `cgame` logic and the complexity of a full x86-to-Wasm JIT.
- **`Ctrl + W` in windowed mode or Firefox:** Browsers prevent web pages from intercepting `Ctrl+W` outside fullscreen Keyboard Lock (supported in Chromium-based browsers). A "Leave site?" prompt guards against accidental tab closure; for the best experience, press **`Alt + Enter`** for fullscreen in Chrome/Edge or rebind Crouch in-game.
- **Self-signed localhost WebTransport certificates in Firefox:** Firefox may cache stale self-signed TLS/Alt-Svc state across local `wtunnel` restarts; clearing site data for `localhost:8080` or using Chrome avoids this during local testing.

---

## License

OpenJK is licensed under the **GNU General Public License v2 (GPLv2)**. See [LICENSE.txt](LICENSE.txt) for details.
