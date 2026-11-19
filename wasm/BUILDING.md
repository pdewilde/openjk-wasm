# Building & Self-Hosting OpenJK WebAssembly

This document covers the technical architecture, Emscripten build pipeline, progressive asset streaming, and `wtunnel` WebTransport relay server for OpenJK WebAssembly (`wasm-v2`).

For a player-oriented overview, see the root [README.md](../README.md). For native desktop OpenJK instructions, see [README_ORIG.md](../README_ORIG.md).

---

## Architecture Overview

- **Modular `SIDE_MODULE=2` Layout:** Compiles Singleplayer (`openjk_sp.wasm32` + `jagamewasm32.wasm`) and Multiplayer (`openjk.wasm32` + `jampgamewasm32.wasm`, `cgamewasm32.wasm`, `uiwasm32.wasm`) as dynamically loaded Wasm side modules via `dlopen`.
- **Graphics, Audio, & Codegen:** GL4ES OpenGL-to-WebGL2 translation, OpenAL via Emscripten WebAudio, WebAssembly SIMD (`-msimd128`), and native Wasm exceptions (`-fwasm-exceptions`).
- **Progressive HTTP `Range` Streaming + OPFS Caching:** `wasm/tools/gen_asset_manifest.py` extracts a small `00_boot.pk3` + `manifest.json` so menus boot quickly while map assets stream on demand via HTTP `Range` requests and cache in the browser's Origin Private File System (OPFS).
- **Hybrid WebTransport Relay (`wasm/wtunnel`):** Tunnels Quake 3 / JKA UDP packets over HTTP/3 WebTransport datagrams with automatic framed unidirectional stream fallback for oversized packets, plus per-IP/global session & rate limiting and optional HTTP Basic Auth for `/base/*`.
- **URL Launch Parameters:** `openjk.wasm32.html` and `openjk_sp.wasm32.html` support `?res=<auto43|window|WxH>`, `?mod=<fs_game>`, `?map=<mapname>`, `?connect=<host:port>`, `?allowdownload=1`, and `?args=<raw_args>`.

---

## Prerequisites

- [Emscripten SDK (`emsdk`)](https://emscripten.org/docs/getting_started/downloads.html) (tested with Emscripten 4.x / 5.x)
- CMake 3.16+ and Ninja / Make
- Python 3 (for asset manifest generation)
- Go 1.22+ (for building `wtunnel`)

---

## 1. Build WebAssembly Binaries

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

Stage the built artifacts, launcher, and icons into a web root (for example `dist/`):

```bash
mkdir -p dist/base dist/icons
cp build-wasm/openjk.wasm32.{html,js,wasm} dist/
cp build-wasm/openjk_sp.wasm32.{html,js,wasm} dist/
cp build-wasm/codemp/game/jampgamewasm32.wasm dist/
cp build-wasm/codemp/cgame/cgamewasm32.wasm dist/
cp build-wasm/codemp/ui/uiwasm32.wasm dist/
cp build-wasm/code/game/jagamewasm32.wasm dist/
cp wasm/index.html dist/index.html
cp wasm/icons/mp128.png wasm/icons/sp128.png dist/icons/
```

---

## 2. Generate Streaming Assets (Optional)

If you want to host streaming base assets for personal use (recommended behind Basic Auth), generate `00_boot.pk3` and `manifest.json` from your `GameData/base/` directory:

```bash
python3 wasm/tools/gen_asset_manifest.py \
  --base-dir "/path/to/Jedi Academy/GameData/base" \
  --out-dir dist/base
```

*(If `dist/base/` is omitted or protected by Basic Auth, unauthenticated players automatically see the local `GameData/base/` folder picker with no login popup.)*

---

## 3. Build & Run `wtunnel` (HTTPS + WebTransport Relay)

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

> **Windows / Linux Note:** Set `QUIC_GO_DISABLE_GSO=true` and `QUIC_GO_DISABLE_ECN=true` if `quic-go` reports UDP GSO/ECN socket errors on your host.

Open **`https://localhost:8080/`** for local development (accept the self-signed certificate), or pass `-domain your.domain.com` in production for automatic Let's Encrypt TLS on port `443` (grant `setcap cap_net_bind_service=+ep wtunnel` if running as a non-root user).

> **Firefox Localhost Note:** Firefox may cache stale self-signed TLS/Alt-Svc state across local `wtunnel` restarts; clearing site data for `localhost:8080` or using Chrome avoids this during local testing.

### `wtunnel` Security & Rate-Limit Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-auth-user` / `-auth-pass` | *(empty — disabled)* | Optional HTTP Basic Auth (or `WTUNNEL_AUTH_USER` / `WTUNNEL_AUTH_PASS`) protecting `/base/*`; unauthenticated visitors silently fall back to the local `base/` picker, while authorized users sign in at `/base/login` |
| `-udp-allow` | *(empty — deny egress)* | Comma-separated allowlist of `host:port` / `CIDR:port` UDP destinations (e.g. dedicated JKA servers) |
| `-max-sessions-per-ip` | `128` | Maximum concurrent WebTransport sessions per client IP (safe for LAN parties / NAT) |
| `-max-sessions` | `2048` | Global concurrent WebTransport session cap |
| `-rate-limit-per-ip` | `10` | Token-bucket handshake rate limit (requests/sec per IP) |
| `-rate-burst-per-ip` | `40` | Token-bucket burst capacity per IP |

---

## Native C/C++ Code Mods

See [wasm/mod-sdk/README.md](mod-sdk/README.md) for CMake presets, ABI flags (`-sSIDE_MODULE=2 -msimd128 -fwasm-exceptions`), and the working sample `cgamewasm32.wasm` mod.
