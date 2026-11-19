#!/usr/bin/env python3
"""
Generate 00_boot.pk3 and manifest.json from a Jedi Academy base/ directory for
progressive HTTP Range + OPFS asset streaming in OpenJK WebAssembly.

Usage:
  python3 gen_asset_manifest.py --base-dir /path/to/GameData/base --out-dir dist/base
"""

import argparse
import hashlib
import json
import os
import re
import shutil
import struct
import zipfile
from collections import defaultdict

BOOT_PREFIXES = (
    'ui/',
    'menu/',
    'fonts/',
    'ext_data/',
    'shaders/',
    'scripts/',
    'strings/',
    'strip/',
    'botfiles/',
    'gfx/2d/',
    'gfx/menus/',
    'gfx/misc/',
    'gfx/hud/',
    'gfx/colors/',
    'sound/interface/',
    'sound/menus/',
)

BOOT_EXTENSIONS = ('.cfg', '.arena', '.bot', '.jkb', '.menu', '.shader', '.fontdat', '.str', '.sp')
TEX_EXTENSIONS = ('.tga', '.jpg', '.jpeg', '.png', '.dds')
MODEL_EXTENSIONS = ('.glm', '.md3', '.gla')
SOUND_EXTENSIONS = ('.wav', '.mp3', '.ogg')

SHADER_TEX_DIRECTIVE_RE = re.compile(
    r'^\s*(?:map|clampmap|animmap(?:\s+[0-9.]+)?)\s+([^\s{}]+)',
    re.IGNORECASE | re.MULTILINE,
)
SKYPARMS_RE = re.compile(r'^\s*skyparms\s+([^\s{}]+)', re.IGNORECASE | re.MULTILINE)
SHADER_BLOCK_RE = re.compile(r'([^\s{}]+)\s*\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}', re.MULTILINE)
KV_RE = re.compile(r'"([^"]+)"\s+"([^"]*)"')


def is_boot_entry(lower_path: str) -> bool:
    if lower_path.endswith('/'):
        return False
    if '/' not in lower_path and lower_path.endswith('.cfg'):
        return True
    if lower_path.startswith(BOOT_PREFIXES):
        return True
    if lower_path.endswith(BOOT_EXTENSIONS):
        return True
    return False


def get_local_data_offset(fp, header_offset: int) -> int:
    fp.seek(header_offset)
    hdr = fp.read(30)
    if len(hdr) < 30 or hdr[:4] != b'PK\x03\x04':
        raise ValueError(f'Invalid ZIP local file header at offset {header_offset}')
    fname_len, extra_len = struct.unpack('<HH', hdr[26:30])
    return header_offset + 30 + fname_len + extra_len


def strip_ext(path: str) -> str:
    dot = path.rfind('.')
    slash = path.rfind('/')
    if dot > slash:
        return path[:dot]
    return path


def parse_shaders(shader_texts: dict) -> dict:
    """Return mapping: lower(shader_name) -> set of lower(referenced texture/map tokens)."""
    shader_map = defaultdict(set)
    for _path, text in shader_texts.items():
        cleaned = re.sub(r'//[^\r\n]*', '', text)
        for m in SHADER_BLOCK_RE.finditer(cleaned):
            sname = m.group(1).strip().lower().replace('\\', '/')
            body = m.group(2)
            for tm in SHADER_TEX_DIRECTIVE_RE.finditer(body):
                for tok in tm.group(1).split():
                    tok_l = tok.strip().lower().replace('\\', '/')
                    if tok_l and not tok_l.startswith('$'):
                        shader_map[sname].add(tok_l)
            for sm in SKYPARMS_RE.finditer(body):
                base = sm.group(1).strip().lower().replace('\\', '/')
                if base and base != '-':
                    for suffix in ('_up', '_dn', '_lf', '_rt', '_ft', '_bk'):
                        shader_map[sname].add(base + suffix)
    return shader_map


def resolve_asset_candidates(token: str, known_entries: set, stem_index: dict) -> list:
    tok = token.strip().lower().replace('\\', '/').lstrip('/')
    if not tok or tok.startswith('*') or tok.startswith('$'):
        return []
    out = []
    if tok in known_entries:
        out.append(tok)
    stem = strip_ext(tok)
    for match in stem_index.get(stem, ()):
        if match not in out:
            out.append(match)
    return out


def parse_bsp_dependencies(bsp_bytes: bytes) -> tuple:
    """Extract (shader_names, entity_dict_values) from IBSP/RBSP."""
    if len(bsp_bytes) < 24 or bsp_bytes[:4] not in (b'RBSP', b'IBSP'):
        return [], []
    # Lumps immediately after 8-byte header (magic + version)
    ent_ofs, ent_len = struct.unpack_from('<ii', bsp_bytes, 8 + 0 * 8)
    sh_ofs, sh_len = struct.unpack_from('<ii', bsp_bytes, 8 + 1 * 8)

    shaders = []
    if sh_ofs >= 0 and sh_len >= 72 and sh_ofs + sh_len <= len(bsp_bytes):
        for off in range(sh_ofs, sh_ofs + sh_len - 71, 72):
            raw = bsp_bytes[off:off + 64].split(b'\x00', 1)[0]
            name = raw.decode('latin1', errors='ignore').strip().lower().replace('\\', '/')
            if name:
                shaders.append(name)

    ent_vals = []
    if ent_ofs >= 0 and ent_len > 0 and ent_ofs + ent_len <= len(bsp_bytes):
        ent_text = bsp_bytes[ent_ofs:ent_ofs + ent_len].decode('latin1', errors='ignore')
        for k, v in KV_RE.findall(ent_text):
            kl = k.lower()
            if kl in ('model', 'model2', 'noise', 'soundset', 'music', 'targetshadername', 'sky', 'wait_sound', 'spawnflags'):
                val = v.strip().lower().replace('\\', '/')
                if val and not val.startswith('*'):
                    ent_vals.append(val)
    return shaders, ent_vals


def main() -> None:
    ap = argparse.ArgumentParser(description='Generate OpenJK-WASM 00_boot.pk3 + manifest.json')
    ap.add_argument('--base-dir', required=True, help='Source base/ directory containing .pk3 files')
    ap.add_argument('--out-dir', required=True, help='Output directory (e.g. dist/base)')
    ap.add_argument('--copy-paks', action='store_true', help='Copy source .pk3 files into --out-dir if different')
    args = ap.parse_args()

    base_dir = os.path.abspath(args.base_dir)
    out_dir = os.path.abspath(args.out_dir)
    os.makedirs(out_dir, exist_ok=True)

    pk3_names = sorted(
        f for f in os.listdir(base_dir)
        if f.lower().endswith('.pk3') and f.lower() != '00_boot.pk3' and os.path.isfile(os.path.join(base_dir, f))
    )
    if not pk3_names:
        raise SystemExit(f'No .pk3 files found in {base_dir}')

    # Highest precedence pak wins when populating 00_boot.pk3 and entries map.
    boot_blobs = {}
    entries = {}
    pak_meta = []
    shader_texts = {}
    stem_index = defaultdict(list)
    bsp_locations = {}

    for pak_idx, pak_name in enumerate(pk3_names):
        pak_path = os.path.join(base_dir, pak_name)
        size = os.path.getsize(pak_path)
        sha = hashlib.sha256()
        pak_crcs = []
        with open(pak_path, 'rb') as fp:
            sha.update(fp.read(min(size, 65536)))
            with zipfile.ZipFile(fp, 'r') as zf:
                for info in zf.infolist():
                    if info.is_dir():
                        continue
                    lpath = info.filename.replace('\\', '/').lstrip('/').lower()
                    if not lpath:
                        continue
                    crc_val = info.CRC & 0xFFFFFFFF
                    if info.file_size > 0:
                        pak_crcs.append(crc_val if crc_val < 0x80000000 else crc_val - 0x100000000)
                    data_ofs = get_local_data_offset(fp, info.header_offset)
                    entries[lpath] = [
                        pak_idx,
                        data_ofs,
                        info.compress_size,
                        info.file_size,
                        info.compress_type,
                    ]
                    stem = strip_ext(lpath)
                    if lpath not in stem_index[stem]:
                        stem_index[stem].append(lpath)
                    if is_boot_entry(lpath):
                        boot_blobs[lpath] = zf.read(info)
                        if lpath.endswith('.shader'):
                            shader_texts[lpath] = boot_blobs[lpath].decode('latin1', errors='ignore')
                    if lpath.startswith('maps/') and lpath.endswith('.bsp'):
                        bsp_locations[lpath] = (pak_path, info)
            pak_meta.append({
                'name': pak_name,
                'basename': pak_name[:-4],
                'size': size,
                'sha256_head': sha.hexdigest()[:16],
                'crcs': pak_crcs,
            })

    # Write 00_boot.pk3 using DEFLATE
    boot_path = os.path.join(out_dir, '00_boot.pk3')
    boot_sha = hashlib.sha256()
    with zipfile.ZipFile(boot_path, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=6) as bf:
        for lpath in sorted(boot_blobs):
            bf.writestr(lpath, boot_blobs[lpath])
    with open(boot_path, 'rb') as bf:
        while True:
            chunk = bf.read(1024 * 1024)
            if not chunk:
                break
            boot_sha.update(chunk)
    boot_size = os.path.getsize(boot_path)

    known_entries = set(entries.keys())
    shader_map = parse_shaders(shader_texts)

    # Collect per-map dependencies
    maps_deps = {}
    for bsp_path, (pak_path, info) in sorted(bsp_locations.items()):
        map_name = bsp_path[len('maps/'):-len('.bsp')]
        deps = {bsp_path}
        for ls_candidate in (f'levelshots/{map_name}.tga', f'levelshots/{map_name}.jpg'):
            if ls_candidate in known_entries:
                deps.add(ls_candidate)
        with zipfile.ZipFile(pak_path, 'r') as zf:
            bsp_bytes = zf.read(info)
        bsp_shaders, ent_vals = parse_bsp_dependencies(bsp_bytes)
        for sh in bsp_shaders:
            for c in resolve_asset_candidates(sh, known_entries, stem_index):
                deps.add(c)
            for tex_tok in shader_map.get(sh, ()):
                for c in resolve_asset_candidates(tex_tok, known_entries, stem_index):
                    deps.add(c)
        for val in ent_vals:
            for c in resolve_asset_candidates(val, known_entries, stem_index):
                deps.add(c)
            # If entity points to a sound/music dir or model, add sibling skins/animations
            if val.endswith('.glm'):
                mdir = val.rsplit('/', 1)[0]
                for ep in known_entries:
                    if ep.startswith(mdir + '/') and ep.endswith(('.skin', '.cfg', '.gla')):
                        deps.add(ep)
        # Keep only files not already shipped inside 00_boot.pk3
        maps_deps[map_name] = sorted(d for d in deps if d in entries and d not in boot_blobs)

    if args.copy_paks and base_dir != out_dir:
        for pak_name in pk3_names:
            shutil.copy2(os.path.join(base_dir, pak_name), os.path.join(out_dir, pak_name))

    manifest = {
        'version': 1,
        'boot': {
            'name': '00_boot.pk3',
            'size': boot_size,
            'sha256': boot_sha.hexdigest(),
            'files': len(boot_blobs),
        },
        'paks': pak_meta,
        'entries': entries,
        'maps': maps_deps,
    }
    manifest_path = os.path.join(out_dir, 'manifest.json')
    with open(manifest_path, 'w', encoding='utf-8') as mf:
        json.dump(manifest, mf, separators=(',', ':'))

    print(
        f'Generated {boot_path} ({boot_size} bytes, {len(boot_blobs)} boot files) '
        f'and {manifest_path} ({len(entries)} total entries, {len(maps_deps)} maps)'
    )


if __name__ == '__main__':
    main()
