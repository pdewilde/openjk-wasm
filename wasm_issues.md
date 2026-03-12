# Known issues

## Singleplayer
1. In tutorial level, if I fail and restart from checkpoint the trees can no longer be cut down, making progress impossible. There also is no longer any dialogue.
2. ~~There is no crosshair in game, despite being enabled in the menu.~~ (FIXED: Resolved as a side effect of fixing the WebGL canvas expansion/resolution bounds).
3. ~~Audio in menu is super choppy. Starts of samples when click button are cut off. This doesn't happen in the old version that used a opengl emulator: https://github.com/lklacar/openjk-emscripten~~ (FIXED: Migrated to Emscripten's OpenAL WebAudio backend for proper hardware mixing, circumventing main thread starvation).
4. ~~No video shows after changing resoultion settings.~~ (WONTFIX: Documented limitation due to WebGL context loss rules, though we stubbed out restart so it is a noop rather than a crash.)
5. Crosshair still disappears after playing for a while.

## Multiplayer
1. **Incompatible with standard (non-WASM) servers:** The WASM client currently uses a reduced `MAX_PACKETLEN` of 1150 bytes to fit within the strict 1200-byte MTU limit of WebTransport Datagrams. Standard servers still send 1400-byte packets, which the WebTransport tunnel drops because they exceed the MTU. This prevents the WASM client from connecting to standard servers until the tunnel is updated to route large packets over WebTransport Streams. This can be fixed by sending larger packets over WebTransport Streams instead of Datagrams. See wasm_port_summary.md for more details.
