#!/usr/bin/env bash
set -euo pipefail

# Build an OpenJK WebAssembly MP mod (SIDE_MODULE=2) and package a ready-to-import mod directory + .zip.
# Usage:
#   ./wasm/mod-sdk/build_mod.sh [mod_source_dir] [output_dir] [mod_name]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MOD_SRC="${1:-${SCRIPT_DIR}/sample-mod}"
OUT_DIR="${2:-/tmp/openjk-wasm-samplemod}"
MOD_NAME="${3:-samplemod}"
BUILD_DIR="${OUT_DIR}/build"
STAGE_DIR="${OUT_DIR}/${MOD_NAME}"

if ! command -v emcmake >/dev/null 2>&1; then
	echo "ERROR: emcmake not found in PATH. Source your emsdk_env.sh first." >&2
	exit 1
fi

mkdir -p "${BUILD_DIR}" "${STAGE_DIR}"

emcmake cmake -S "${MOD_SRC}" -B "${BUILD_DIR}" \
	-DCMAKE_BUILD_TYPE=Release

cmake --build "${BUILD_DIR}" -j"$(nproc 2>/dev/null || echo 4)"

# Collect built *.wasm side modules
find "${BUILD_DIR}" -maxdepth 2 -name "*wasm32.wasm" -exec cp -v {} "${STAGE_DIR}/" \;

if [ -f "${MOD_SRC}/description.txt" ]; then
	cp -v "${MOD_SRC}/description.txt" "${STAGE_DIR}/description.txt"
fi

# Package a tiny sample .pk3 asset pack alongside the .wasm if building sample-mod
if [ "$(basename "${MOD_SRC}")" = "sample-mod" ] && command -v python3 >/dev/null 2>&1; then
	python3 - "${STAGE_DIR}" "${MOD_NAME}" <<'PY'
import os, sys, zipfile
stage_dir, mod_name = sys.argv[1], sys.argv[2]
pk3_path = os.path.join(stage_dir, f"{mod_name}_assets.pk3")
with zipfile.ZipFile(pk3_path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
    zf.writestr("samplemod.cfg", 'echo "^2[SampleMod-WASM]^7 samplemod.cfg loaded from samplemod_assets.pk3"\n')
zip_path = os.path.join(os.path.dirname(stage_dir), f"{mod_name}.zip")
with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
    for fn in sorted(os.listdir(stage_dir)):
        full = os.path.join(stage_dir, fn)
        if os.path.isfile(full):
            zf.write(full, arcname=f"{mod_name}/{fn}")
print(f"Packaged {pk3_path} and {zip_path}")
PY
fi

echo "Done! Mod staged in: ${STAGE_DIR}"
