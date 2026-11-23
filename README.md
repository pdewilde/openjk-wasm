# Star Wars: Jedi Academy — OpenJK Web

Play **Star Wars Jedi Knight: Jedi Academy** (Singleplayer & Multiplayer) right in your browser, powered by [OpenJK](https://github.com/JACoders/OpenJK) and WebAssembly.

### **[▶ Play Now at openjk.pdewil.de](https://openjk.pdewil.de/)**

> **Game Files Required:** Just like desktop OpenJK, copyrighted Jedi Academy assets are not included. You'll need your own copy of Jedi Academy ([Steam](https://store.steampowered.com/app/6020/), [GOG](https://www.gog.com/game/star_wars_jedi_knight_jedi_academy), or original CD).
> - **Own it on GOG?** Download the installer directly from [GOG's installer link](https://www.gog.com/downloads/star_wars_jedi_knight_jedi_academy/en1installer0) (while signed into GOG) and drop the `.exe` right onto the homepage—the browser unpacks `assets0.pk3`–`assets3.pk3` locally in a few seconds (no need to run the `.exe`, works on macOS/Linux/Chromebooks too!).
> - **Own it on Steam / already installed?** Select your local `GameData/base` folder (`assets0.pk3`–`assets3.pk3`). Everything stays 100% local in your browser and is never uploaded to a server.

---

## Features

- **Multiplayer in the Browser (plus Experimental Singleplayer):** Runs in modern Chrome, Edge, and Firefox with hardware-accelerated WebGL2 graphics and WebAudio sound. *(Singleplayer is included but currently untested and likely has bugs.)*
- **Browser Multiplayer & Dedicated Server Hosting:** Host a full headless dedicated server (`openjkded.wasm32`) directly from your browser tab via **`server.html`** ("Host Dedicated Server") running inside a Web Worker on the `wtunnel` Virtual LAN — complete with a live Control Deck, 1-click Join Links, automatic Local/LAN browser discovery, all 43 JKA bots, live map/gametype/team switching, collapsible **Weapon / Force Power / Physics** rules, and **JSON config export/import**.
- **Easy Mod Support + Built-In [JA++ (`japp-wasm`)](https://github.com/pdewilde/japp-wasm):** Enable **JA++** with a single checkbox (no upload needed!), powered by our WebAssembly fork [pdewilde/japp-wasm](https://github.com/pdewilde/japp-wasm) (forked from [Razish/japp](https://github.com/Razish/japp)), or drag and drop your own `.zip`, `.pk3`, or mod folders right onto the home page. Dedicated JA++ servers support hot-reloadable **JPLua** server scripts + bundled **SQLite**, including built-in plugins for **Auto-Bot Balancing**, **Bot-Only Weapon/Force Restrictions**, **SQLite Stats DB**, **Killstreaks**, and **Vampire**.
- **Saved Locally:** Your singleplayer saves, settings, server Lua scripts, SQLite databases, and installed mods are stored locally in your browser (OPFS).
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

- **[JA++ WebAssembly Fork (`pdewilde/japp-wasm`)](https://github.com/pdewilde/japp-wasm)** — Fork of [Razish/japp](https://github.com/Razish/japp) compiled to OpenJK WebAssembly `SIDE_MODULE=2` binaries (`cgamewasm32.wasm`, `jampgamewasm32.wasm`, `uiwasm32.wasm`) with bundled SQLite and a clean-room GPLv2 `lua/init.lua` JPLua bootstrap. Automatically fetched and version-synced into OPFS via `wasm/mods/catalog.json` for both the browser client and the Web Worker dedicated server (`server.html?mod=japp`).
- **[Building & Self-Hosting Guide (`wasm/BUILDING.md`)](wasm/BUILDING.md)** — Emscripten build instructions, architecture details, progressive asset streaming (`gen_asset_manifest.py`), and running the `wtunnel` WebTransport server.
- **[Native WebAssembly Mod SDK (`wasm/mod-sdk/README.md`)](wasm/mod-sdk/README.md)** — Compiling C/C++ multiplayer mods to WebAssembly (`SIDE_MODULE=2`) and building the sample mod.
- **[Original Upstream OpenJK README (`README_ORIG.md`)](README_ORIG.md)** — Desktop OpenJK documentation and contributor credits.

---

## Contributing & Bug Reports

Found a bug, hit a browser quirk, or want to help test Singleplayer, Multiplayer, or Dedicated Server hosting? Contributions are very welcome!

- **Report bugs or request features:** Open an issue on [openjk-wasm Issues](https://github.com/pdewilde/openjk-wasm/issues) (or [japp-wasm Issues](https://github.com/pdewilde/japp-wasm/issues) for JA++-specific bugs). Including your browser/OS and any console log output (`Ctrl + \``) helps a lot.
- **Submit Pull Requests:** PRs for bug fixes, browser compatibility, UI improvements, Lua server plugins, and documentation are always welcome on both [openjk-wasm](https://github.com/pdewilde/openjk-wasm/pulls) and [japp-wasm](https://github.com/pdewilde/japp-wasm/pulls).

---

## License

OpenJK is licensed under the **GNU General Public License v2 (GPLv2)**. See [LICENSE.txt](LICENSE.txt) for details.
