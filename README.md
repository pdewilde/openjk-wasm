# Star Wars: Jedi Academy — OpenJK Web

Play **Star Wars Jedi Knight: Jedi Academy** (Singleplayer & Multiplayer) right in your browser, powered by [OpenJK](https://github.com/JACoders/OpenJK) and WebAssembly.

### **[▶ Play Now at openjk.pdewil.de](https://openjk.pdewil.de/)**

> **Game Files Required:** Just like desktop OpenJK, copyrighted Jedi Academy assets are not included. You'll need your own copy of Jedi Academy ([Steam](https://store.steampowered.com/app/6020/), [GOG](https://www.gog.com/game/star_wars_jedi_knight_jedi_academy), or original CD).
> - **Own it on GOG?** Download the installer directly from [GOG's installer link](https://www.gog.com/downloads/star_wars_jedi_knight_jedi_academy/en1installer0) (while signed into GOG) and drop the `.exe` right onto the homepage—the browser unpacks `assets0.pk3`–`assets3.pk3` locally in a few seconds (no need to run the `.exe`, works on macOS/Linux/Chromebooks too!).
> - **Own it on Steam / already installed?** Select your local `GameData/base` folder (`assets0.pk3`–`assets3.pk3`). Everything stays 100% local in your browser and is never uploaded to a server.

---

## Features

- **Multiplayer in the Browser (plus Experimental Singleplayer):** Runs in modern Chrome, Edge, and Firefox with hardware-accelerated WebGL2 graphics and WebAudio sound. *(Singleplayer is included but currently untested and likely has bugs.)*
- **Browser Multiplayer:** Host or join multiplayer matches directly from the browser via WebTransport.
- **Easy Mod Support:** Enable built-in FOSS mods like **JA++** with a single checkbox (no upload needed!), or drag and drop your own `.zip`, `.pk3`, or mod folders right onto the home page.
- **Saved Locally:** Your singleplayer saves, settings, and installed mods are stored locally in your browser.
- **Display & Controls:** Defaults to a classic crisp **4:3** aspect ratio fitted to your window, with a pre-launch resolution picker under **Advanced options**, live in-game resolution & brightness sliders, and shareable map/server links.

---

## Tips & Shortcuts

- **`Alt + Enter`**: Toggle fullscreen. In Chrome and Edge, fullscreen locks browser shortcuts so `Ctrl+W`, `Ctrl+R`, `Tab`, and `Esc` go straight to the game. Hold **`Esc`** to exit fullscreen.
- **`Ctrl + \``**: Toggle the engine log console.
- **Change resolution before launching:** Open **Advanced options** on the home page to pick a preset (4:3, 16:9, or full-window widescreen).
- **Share a match:** Joining a map or remote server updates the browser URL bar (`?map=...` or `?connect=...`) so you can share the link directly.

---

## Known Limitations

- **Singleplayer is currently untested:** Testing and polish so far have focused on Multiplayer. Singleplayer launches, but has not been thoroughly tested and likely has bugs.
- **Closed-source desktop code mods (`.dll` / `.so` / `.dylib`):** Mods that ship only precompiled native x86 binaries (such as **JA+ / japlus**) cannot run their custom game code in the browser without being compiled from source to WebAssembly. Their `.pk3` maps, sabers, skins, and animations still load, and the launcher flags them with an **Assets Only (x86 code)** badge.
- **`Ctrl + W` in windowed mode or Firefox:** Browsers don't let web pages block `Ctrl+W` outside Chromium fullscreen mode. A "Leave site?" prompt helps prevent accidental tab closes—use **`Alt + Enter`** fullscreen in Chrome/Edge or rebind Crouch in-game for the smoothest experience.

---

## Technical & Developer Documentation

- **[Building & Self-Hosting Guide (`wasm/BUILDING.md`)](wasm/BUILDING.md)** — Emscripten build instructions, architecture details, progressive asset streaming (`gen_asset_manifest.py`), and running the `wtunnel` WebTransport server.
- **[Native WebAssembly Mod SDK (`wasm/mod-sdk/README.md`)](wasm/mod-sdk/README.md)** — Compiling C/C++ multiplayer mods to WebAssembly (`SIDE_MODULE=2`) and building the sample mod.
- **[Original Upstream OpenJK README (`README_ORIG.md`)](README_ORIG.md)** — Desktop OpenJK documentation and contributor credits.

---

## License

OpenJK is licensed under the **GNU General Public License v2 (GPLv2)**. See [LICENSE.txt](LICENSE.txt) for details.
