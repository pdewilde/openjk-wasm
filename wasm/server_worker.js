'use strict';

// Dedicated OpenJK WebAssembly server Web Worker.
// Runs openjkded.wasm32.js off the main thread so:
//   - The Server Manager UI stays 60fps responsive while maps/assets load.
//   - Switching to another browser tab to PLAY on this server does NOT throttle
//     or pause the server's tick loop or WebTransport connection.

var workerStarted = false;
var statusTimer = null;
var lastVipStr = '';
var pendingAutoBots = 0;
var pendingBotSkill = 2;
var autoBotsAdded = false;

var DEFAULT_BOT_ROSTER = [
	'Kyle Katarn', 'Luke Skywalker', 'Desann', 'Tavion', 'Alora', 'Reborn',
	'Boba Fett', 'Stormtrooper', 'Shadowtrooper', 'Chewbacca', 'Jan Ors', 'Rosh Penin'
];

function postLog(level, text) {
	self.postMessage({ type: 'log', level: level, text: String(text) });
}

function postPhase(text) {
	self.postMessage({ type: 'phase', text: String(text) });
}

function sanitizeModName(name) {
	return String(name || '').trim().replace(/[^a-zA-Z0-9._-]/g, '').replace(/^\.+|\.+$/g, '');
}

function getWasmFs() {
	if (typeof FS !== 'undefined' && FS) return FS;
	if (self.Module && self.Module.FS) return self.Module.FS;
	return null;
}

function ensureDirTree(fs, dirPath) {
	var parts = String(dirPath || '').split('/').filter(Boolean);
	var cur = '';
	for (var i = 0; i < parts.length; i++) {
		cur += '/' + parts[i];
		try { fs.mkdir(cur); } catch (e) { }
	}
}

function writeWasmFile(fs, fullPath, bytes) {
	var norm = String(fullPath || '').replace(/\\/g, '/');
	var idx = norm.lastIndexOf('/');
	if (idx > 0) ensureDirTree(fs, norm.slice(0, idx));
	try { fs.unlink(norm); } catch (e) { }
	fs.writeFile(norm, bytes);
}

async function getOpfsDir(rootName, subParts, create) {
	if (!navigator.storage || typeof navigator.storage.getDirectory !== 'function') return null;
	try {
		var root = await navigator.storage.getDirectory();
		var dir = await root.getDirectoryHandle(rootName, { create: !!create });
		for (var i = 0; i < subParts.length; i++) {
			dir = await dir.getDirectoryHandle(subParts[i], { create: !!create });
		}
		return dir;
	} catch (e) {
		return null;
	}
}

async function inflateRawBytes(compBytes) {
	var stream = new Blob([compBytes]).stream().pipeThrough(new DecompressionStream('deflate-raw'));
	var ab = await new Response(stream).arrayBuffer();
	return new Uint8Array(ab);
}

async function extractModZipEntries(arrayBuffer) {
	var bytes = new Uint8Array(arrayBuffer);
	var view = new DataView(arrayBuffer);
	var minEocd = Math.max(0, bytes.length - 65557);
	var eocd = -1;
	for (var p = bytes.length - 22; p >= minEocd; p--) {
		if (view.getUint32(p, true) === 0x06054b50) { eocd = p; break; }
	}
	if (eocd === -1) throw new Error('Invalid ZIP archive (missing EOCD)');
	var cdCount = view.getUint16(eocd + 10, true);
	var cdOffset = view.getUint32(eocd + 16, true);
	var dec = new TextDecoder('utf-8');
	var out = [];
	var pos = cdOffset;
	for (var i = 0; i < cdCount; i++) {
		if (pos + 46 > bytes.length || view.getUint32(pos, true) !== 0x02014b50) break;
		var method = view.getUint16(pos + 10, true);
		var compSize = view.getUint32(pos + 20, true);
		var nameLen = view.getUint16(pos + 28, true);
		var extraLen = view.getUint16(pos + 30, true);
		var commentLen = view.getUint16(pos + 32, true);
		var localOfs = view.getUint32(pos + 42, true);
		var entryName = dec.decode(bytes.subarray(pos + 46, pos + 46 + nameLen)).replace(/\\/g, '/');
		pos += 46 + nameLen + extraLen + commentLen;
		if (!entryName || entryName.endsWith('/') || entryName.indexOf('..') !== -1) continue;
		if (localOfs + 30 > bytes.length || view.getUint32(localOfs, true) !== 0x04034b50) continue;
		var lNameLen = view.getUint16(localOfs + 26, true);
		var lExtraLen = view.getUint16(localOfs + 28, true);
		var dataStart = localOfs + 30 + lNameLen + lExtraLen;
		var compSlice = bytes.subarray(dataStart, dataStart + compSize);
		var raw;
		if (method === 0) raw = new Uint8Array(compSlice);
		else if (method === 8) raw = await inflateRawBytes(compSlice);
		else continue;
		out.push({ path: entryName, bytes: raw });
	}
	return out;
}

// --- OPFS Save Sync ---
var opfsSaveChain = Promise.resolve();
function enqueueOpfsSave(fn) {
	opfsSaveChain = opfsSaveChain.then(fn).catch(function () { });
}

globalThis.__openjkOpfsSyncFile = function (osPath) {
	var norm = String(osPath || '').replace(/\\/g, '/');
	if (norm.indexOf('/saves/') !== 0) return;
	var rel = norm.slice('/saves/'.length).replace(/^\/+/, '');
	if (!rel || rel.indexOf('..') !== -1) return;
	var fs = getWasmFs();
	var copy;
	try { copy = new Uint8Array(fs.readFile(norm)); } catch (e) { return; }
	var parts = rel.split('/');
	var fn = parts.pop();
	enqueueOpfsSave(async function () {
		var dir = await getOpfsDir('openjk-saves', parts, true);
		if (!dir) return;
		var fh = await dir.getFileHandle(fn, { create: true });
		var w = await fh.createWritable();
		await w.write(copy);
		await w.close();
	});
};

globalThis.__openjkOpfsRemoveFile = function (osPath) {
	var norm = String(osPath || '').replace(/\\/g, '/');
	if (norm.indexOf('/saves/') !== 0) return;
	var rel = norm.slice('/saves/'.length).replace(/^\/+/, '');
	if (!rel || rel.indexOf('..') !== -1) return;
	var parts = rel.split('/');
	var fn = parts.pop();
	enqueueOpfsSave(async function () {
		var dir = await getOpfsDir('openjk-saves', parts, false);
		if (!dir) return;
		await dir.removeEntry(fn);
	});
};

// --- Streaming Manifest Mode (for hosted /base/ assets) ---
var manifestState = {
	data: null,
	pakEntries: [],
	mounted: new Set(),
	activeCount: 0,
	inflight: Object.create(null),
	pendingCmd: ''
};

var opfsCacheChain = Promise.resolve();
function storeOpfsCacheFile(relPath, bytes) {
	var norm = String(relPath || '').replace(/\\/g, '/').replace(/^\/+/, '').toLowerCase();
	if (!norm || norm.indexOf('..') !== -1) return;
	var copy = new Uint8Array(bytes);
	var parts = ['entries'].concat(norm.split('/'));
	var fn = parts.pop();
	opfsCacheChain = opfsCacheChain.then(async function () {
		var dir = await getOpfsDir('openjk-cache', parts, true);
		if (!dir) return;
		var fh = await dir.getFileHandle(fn, { create: true });
		var w = await fh.createWritable();
		await w.write(copy);
		await w.close();
	}).catch(function () { });
}

async function prefetchMapDependencies(mapKey, deps) {
	var fs = getWasmFs();
	var missing = [];
	for (var i = 0; i < deps.length; i++) {
		var p = deps[i];
		if (!manifestState.mounted.has(p) && manifestState.data.entries[p]) missing.push(p);
	}
	if (missing.length === 0) return;

	var itemsByPak = Object.create(null);
	var totalItems = 0;
	for (var j = 0; j < missing.length; j++) {
		var path = missing[j];
		var e = manifestState.data.entries[path];
		var pakIdx = e[0];
		var compLen = e[2];
		var uncompLen = e[3];
		if (compLen === 0 && uncompLen === 0) {
			writeWasmFile(fs, '/base/' + path, new Uint8Array(0));
			manifestState.mounted.add(path);
			continue;
		}
		if (!itemsByPak[pakIdx]) itemsByPak[pakIdx] = [];
		itemsByPak[pakIdx].push({
			path: path,
			pakName: manifestState.data.paks[pakIdx].name,
			start: e[1],
			compLen: compLen,
			uncompLen: uncompLen,
			method: e[4]
		});
		totalItems++;
	}
	if (totalItems === 0) return;

	var GAP_LIMIT = 65536;
	var MAX_BATCH = 2 * 1024 * 1024;
	var batches = [];
	Object.keys(itemsByPak).forEach(function (pk) {
		var list = itemsByPak[pk];
		list.sort(function (a, b) { return a.start - b.start; });
		var cur = null;
		for (var li = 0; li < list.length; li++) {
			var it = list[li];
			var itEnd = it.start + it.compLen;
			if (!cur || (it.start - cur.end) > GAP_LIMIT || (itEnd - cur.start) > MAX_BATCH) {
				cur = { pakName: it.pakName, start: it.start, end: itEnd, items: [it] };
				batches.push(cur);
			} else {
				if (itEnd > cur.end) cur.end = itEnd;
				cur.items.push(it);
			}
		}
	});

	var completed = 0;
	postPhase('Streaming map "' + mapKey + '" assets (0/' + totalItems + ')...');
	var cursor = 0;
	async function runBatchWorker() {
		while (cursor < batches.length) {
			var batch = batches[cursor++];
			var resp = await fetch('base/' + batch.pakName, {
				headers: { 'Range': 'bytes=' + batch.start + '-' + (batch.end - 1) }
			});
			if (!resp.ok) throw new Error('Range fetch failed (' + resp.status + ') for ' + batch.pakName);
			var isPartial = (resp.status === 206);
			var buf = new Uint8Array(await resp.arrayBuffer());
			for (var bi = 0; bi < batch.items.length; bi++) {
				var item = batch.items[bi];
				var relOfs = isPartial ? (item.start - batch.start) : item.start;
				var compSlice = buf.subarray(relOfs, relOfs + item.compLen);
				var outBytes = (item.method === 8) ? await inflateRawBytes(compSlice) : new Uint8Array(compSlice);
				writeWasmFile(fs, '/base/' + item.path, outBytes);
				manifestState.mounted.add(item.path);
				storeOpfsCacheFile(item.path, outBytes);
				completed++;
				if (completed % 25 === 0 || completed === totalItems) {
					postPhase('Streaming map "' + mapKey + '" (' + completed + '/' + totalItems + ')...');
				}
			}
		}
	}
	var workers = [];
	for (var w = 0; w < Math.min(4, batches.length); w++) workers.push(runBatchWorker());
	await Promise.all(workers);
	postPhase('Online');
}

function installManifestBindings() {
	globalThis.__openjkManifestApi = {
		getPakCount: function (gameDir) {
			if (!manifestState.data || gameDir !== 'base') return 0;
			return manifestState.data.paks.length;
		},
		getPakInfo: function (pakIdx) {
			if (!manifestState.data || pakIdx < 0 || pakIdx >= manifestState.data.paks.length) return null;
			var p = manifestState.data.paks[pakIdx];
			return { basename: p.basename, numEntries: manifestState.pakEntries[pakIdx].length, numCrcs: p.crcs ? p.crcs.length : 0 };
		},
		getPakCrcs: function (pakIdx) {
			if (!manifestState.data || pakIdx < 0 || pakIdx >= manifestState.data.paks.length) return null;
			return manifestState.data.paks[pakIdx].crcs || [];
		},
		getPakEntry: function (pakIdx, entryIdx) {
			var list = manifestState.pakEntries[pakIdx];
			return (list && entryIdx >= 0 && entryIdx < list.length) ? list[entryIdx] : null;
		},
		fetchRangeSync: function (pakBasename, dataOfs, compLen) {
			if (!manifestState.data || compLen <= 0) return new Uint8Array(0);
			var pak = null;
			for (var i = 0; i < manifestState.data.paks.length; i++) {
				if (manifestState.data.paks[i].basename === pakBasename) { pak = manifestState.data.paks[i]; break; }
			}
			if (!pak) return null;
			var xhr = new XMLHttpRequest();
			xhr.open('GET', 'base/' + pak.name, false);
			xhr.setRequestHeader('Range', 'bytes=' + dataOfs + '-' + (dataOfs + compLen - 1));
			xhr.overrideMimeType('text/plain; charset=x-user-defined');
			xhr.send(null);
			if (xhr.status !== 206 && xhr.status !== 200) return null;
			var text = xhr.responseText || '';
			var startAt = (xhr.status === 200) ? dataOfs : 0;
			if (text.length < startAt + compLen) return null;
			var out = new Uint8Array(compLen);
			for (var j = 0; j < compLen; j++) out[j] = text.charCodeAt(startAt + j) & 0xff;
			return out;
		},
		cachePutFile: function (cleanQ, destPath) {
			var key = String(cleanQ || '').replace(/\\/g, '/').replace(/^\/+/, '').toLowerCase();
			if (!key) return;
			manifestState.mounted.add(key);
			try { storeOpfsCacheFile(key, getWasmFs().readFile(destPath)); } catch (e) { }
		}
	};

	globalThis.__openjkIsMapPrefetchActive = function () { return manifestState.activeCount > 0 ? 1 : 0; };
	globalThis.__openjkConsumePendingCommand = function () {
		if (manifestState.activeCount > 0 || !manifestState.pendingCmd) return '';
		var cmd = manifestState.pendingCmd;
		manifestState.pendingCmd = '';
		return cmd;
	};
	globalThis.__openjkEnsureMapAssets = function (mapName, retryCmd) {
		if (!manifestState.data) return 1;
		var m = String(mapName || '').trim().toLowerCase().replace(/^maps\//, '').replace(/\.bsp$/, '');
		var deps = (manifestState.data.maps && manifestState.data.maps[m]) ? manifestState.data.maps[m] : [];
		if (deps.length === 0) return 1;
		var need = false;
		for (var i = 0; i < deps.length; i++) {
			if (!manifestState.mounted.has(deps[i]) && manifestState.data.entries[deps[i]]) { need = true; break; }
		}
		if (!need) return 1;
		if (manifestState.inflight[m]) {
			if (retryCmd) manifestState.inflight[m].retryCmd = retryCmd;
			return 0;
		}
		var st = { retryCmd: retryCmd || '' };
		manifestState.activeCount++;
		manifestState.inflight[m] = st;
		prefetchMapDependencies(m, deps)
			.catch(function (err) { postLog('err', 'Map prefetch error (' + m + '): ' + err.message); })
			.finally(function () {
				delete manifestState.inflight[m];
				manifestState.activeCount--;
				if (st.retryCmd) manifestState.pendingCmd = st.retryCmd;
			});
		return 0;
	};
}

async function hydrateOpfsDir(dirHandle, targetPrefix, fs) {
	var count = 0;
	for await (var entry of dirHandle.entries()) {
		var name = entry[0];
		var handle = entry[1];
		if (name === '.catalog-version') continue;
		var dest = targetPrefix + '/' + name;
		if (handle.kind === 'directory') {
			ensureDirTree(fs, dest);
			count += await hydrateOpfsDir(handle, dest, fs);
		} else if (handle.kind === 'file') {
			var f = await handle.getFile();
			var bytes = new Uint8Array(await f.arrayBuffer());
			writeWasmFile(fs, dest, bytes);
			count++;
		}
	}
	return count;
}

async function hydrateOpfsManifestCache(dirHandle, relPrefix, fs, manifestEntries) {
	for await (var entry of dirHandle.entries()) {
		var name = entry[0];
		var handle = entry[1];
		var rel = relPrefix ? (relPrefix + '/' + name) : name;
		if (handle.kind === 'directory') {
			await hydrateOpfsManifestCache(handle, rel, fs, manifestEntries);
		} else if (handle.kind === 'file') {
			var key = rel.toLowerCase();
			var meta = manifestEntries[key];
			if (!meta) continue;
			var f = await handle.getFile();
			if (f.size !== meta[3]) {
				try { await dirHandle.removeEntry(name); } catch (e) { }
				continue;
			}
			var bytes = new Uint8Array(await f.arrayBuffer());
			writeWasmFile(fs, '/base/' + rel, bytes);
			manifestState.mounted.add(key);
		}
	}
}

async function initHostedManifest(fs) {
	var resp;
	try {
		resp = await fetch('base/manifest.json', { cache: 'no-cache', headers: { 'X-No-Auth-Prompt': '1' } });
	} catch (e) { return false; }
	if (!resp || !resp.ok) return false;
	var manifest = await resp.json();
	if (!manifest || !manifest.boot || !Array.isArray(manifest.paks) || !manifest.entries) return false;

	postPhase('Downloading boot pack...');
	var bootResp = await fetch('base/' + manifest.boot.name, { headers: { 'X-No-Auth-Prompt': '1' } });
	if (!bootResp || !bootResp.ok) return false;
	var bootBytes = new Uint8Array(await bootResp.arrayBuffer());
	writeWasmFile(fs, '/base/' + manifest.boot.name, bootBytes);

	manifestState.data = manifest;
	manifestState.pakEntries = manifest.paks.map(function () { return []; });
	Object.keys(manifest.entries).forEach(function (k) {
		var e = manifest.entries[k];
		if (manifestState.pakEntries[e[0]]) {
			manifestState.pakEntries[e[0]].push([k, e[1], e[2], e[3], e[4]]);
		}
	});
	installManifestBindings();

	var cacheDir = await getOpfsDir('openjk-cache', ['entries'], false);
	if (cacheDir) {
		postPhase('Restoring cached map assets...');
		await hydrateOpfsManifestCache(cacheDir, '', fs, manifest.entries);
	}
	return true;
}

async function hydrateBaseAssets(fs, sessionBaseFiles) {
	ensureDirTree(fs, '/base');

	// 1. Private-mode transferred File objects
	if (Array.isArray(sessionBaseFiles) && sessionBaseFiles.length > 0) {
		postPhase('Mounting session base files (' + sessionBaseFiles.length + ')...');
		for (var i = 0; i < sessionBaseFiles.length; i++) {
			var f = sessionBaseFiles[i];
			var bytes = new Uint8Array(await f.arrayBuffer());
			writeWasmFile(fs, '/base/' + f.name, bytes);
		}
		return 'session';
	}

	// 2. Local OPFS openjk-base/ (assets0-3.pk3)
	var baseDir = await getOpfsDir('openjk-base', [], false);
	if (baseDir) {
		var hasAssets0 = false;
		try {
			await baseDir.getFileHandle('assets0.pk3');
			hasAssets0 = true;
		} catch (e) { }
		if (hasAssets0) {
			postPhase('Loading local GameData/base from browser storage...');
			await hydrateOpfsDir(baseDir, '/base', fs);
			return 'opfs';
		}
	}

	// 3. Hosted streaming manifest (/base/manifest.json)
	if (await initHostedManifest(fs)) {
		return 'hosted';
	}

	return null;
}

async function hydrateMod(fs, modId) {
	var clean = sanitizeModName(modId);
	if (!clean || clean.toLowerCase() === 'base') return;

	var opfsDir = await getOpfsDir('openjk-mods', [clean], false);
	var catalogEntry = null;
	try {
		var cResp = await fetch('mods/catalog.json', { cache: 'no-cache' });
		if (cResp && cResp.ok) {
			var cData = await cResp.json();
			if (cData && Array.isArray(cData.mods)) {
				for (var i = 0; i < cData.mods.length; i++) {
					if (sanitizeModName(cData.mods[i].id).toLowerCase() === clean.toLowerCase()) {
						catalogEntry = cData.mods[i];
						break;
					}
				}
			}
		}
	} catch (e) { }

	var needCatalogFetch = false;
	if (catalogEntry && catalogEntry.url) {
		if (!opfsDir) {
			needCatalogFetch = true;
		} else if (catalogEntry.version) {
			var cachedVer = '';
			try {
				var vfh = await opfsDir.getFileHandle('.catalog-version');
				cachedVer = (await (await vfh.getFile()).text()).trim();
			} catch (e) { cachedVer = ''; }
			if (cachedVer !== String(catalogEntry.version)) needCatalogFetch = true;
		}
	}

	if (needCatalogFetch) {
		postPhase('Downloading built-in mod "' + (catalogEntry.name || clean) + '"...');
		var fetchUrl = catalogEntry.url + (catalogEntry.version ? ((catalogEntry.url.indexOf('?') === -1 ? '?' : '&') + 'v=' + encodeURIComponent(catalogEntry.version)) : '');
		var zResp = await fetch(fetchUrl);
		if (!zResp || !zResp.ok) throw new Error('Failed to fetch mod archive: ' + catalogEntry.url);
		var items = await extractModZipEntries(await zResp.arrayBuffer());
		ensureDirTree(fs, '/' + clean);
		var prefixLower = clean.toLowerCase() + '/';
		var writeOpfsDir = await getOpfsDir('openjk-mods', [clean], true);
		for (var ii = 0; ii < items.length; ii++) {
			var rel = items[ii].path.replace(/^\/+/, '');
			if (rel.toLowerCase().indexOf(prefixLower) === 0) rel = rel.slice(prefixLower.length);
			if (!rel || rel.indexOf('..') !== -1) continue;
			writeWasmFile(fs, '/' + clean + '/' + rel, items[ii].bytes);
			if (writeOpfsDir) {
				try {
					var parts = rel.split('/').filter(Boolean);
					var fn = parts.pop();
					var sub = writeOpfsDir;
					for (var pi = 0; pi < parts.length; pi++) sub = await sub.getDirectoryHandle(parts[pi], { create: true });
					var fh = await sub.getFileHandle(fn, { create: true });
					var w = await fh.createWritable();
					await w.write(items[ii].bytes);
					await w.close();
				} catch (e) { }
			}
		}
		if (writeOpfsDir && catalogEntry.version) {
			try {
				var vfhW = await writeOpfsDir.getFileHandle('.catalog-version', { create: true });
				var vw = await vfhW.createWritable();
				await vw.write(String(catalogEntry.version));
				await vw.close();
			} catch (e) { }
		}
		return;
	}

	if (opfsDir) {
		postPhase('Mounting mod "' + clean + '"...');
		ensureDirTree(fs, '/' + clean);
		await hydrateOpfsDir(opfsDir, '/' + clean, fs);
	}
}

function buildServerArgs(cfg) {
	var args = [
		'+set', 'dedicated', '2',
		'+set', 'fs_basepath', '/',
		'+set', 'fs_homepath', '/saves',
		'+set', 'net_port', String(cfg.port || 29070),
		'+set', 'sv_hostname', String(cfg.hostname || 'OpenJK Web Server'),
		'+set', 'g_gametype', String(cfg.gametype != null ? cfg.gametype : 0),
		'+set', 'sv_maxclients', String(cfg.maxClients || 32),
		'+set', 'fraglimit', String(cfg.fraglimit != null ? cfg.fraglimit : 20),
		'+set', 'timelimit', String(cfg.timelimit != null ? cfg.timelimit : 15),
		'+set', 'sv_pure', cfg.pure ? '1' : '0',
		'+set', 'sv_fps', String(cfg.svFps || 20),
		'+set', 'sv_allowDownload', '0'
	];
	if (cfg.capturelimit != null && cfg.capturelimit > 0) {
		args.push('+set', 'capturelimit', String(cfg.capturelimit));
	}
	if (cfg.password) {
		args.push('+set', 'g_needpass', '1', '+set', 'g_password', String(cfg.password));
	} else {
		args.push('+set', 'g_needpass', '0', '+set', 'g_password', '');
	}
	var modId = sanitizeModName(cfg.mod);
	if (modId && modId.toLowerCase() !== 'base') {
		args.push('+set', 'fs_game', modId);
	}
	if (modId.toLowerCase() === 'japp' && cfg.jappOpts) {
		var jo = cfg.jappOpts;
		args.push(
			'+set', 'japp_promode', jo.promode ? '1' : '0',
			'+set', 'japp_unlagged', jo.unlagged ? '1' : '0',
			'+set', 'japp_allowHook', jo.hook ? '1' : '0',
			'+set', 'japp_allowJetpack', jo.jetpack ? '1' : '0',
			'+set', 'japp_instagib', jo.instagib ? '1' : '0',
			'+set', 'japp_vq3physics', jo.vq3 ? '1' : '0'
		);
	}
	if (cfg.extraArgs) {
		var tokens = String(cfg.extraArgs).match(/(?:[^\s"]+|"[^"]*")+/g) || [];
		for (var i = 0; i < tokens.length; i++) {
			args.push(tokens[i].replace(/^"|"$/g, ''));
		}
	}
	var startMap = String(cfg.map || 'mp/ffa1').trim() || 'mp/ffa1';
	args.push('+map', startMap);
	return args;
}

function pollServerStatus() {
	var M = self.Module;
	if (!M) return;

	if (M.wtVIP && M.wtVIP.length >= 4) {
		var vipStr = M.wtVIP[0] + '.' + M.wtVIP[1] + '.' + M.wtVIP[2] + '.' + M.wtVIP[3];
		if (vipStr !== lastVipStr) {
			lastVipStr = vipStr;
			self.postMessage({ type: 'vip', ip: vipStr, port: M.wtSrcPort || 29070 });
		}
	}

	if (typeof M._OpenJKDed_GetStatusJson === 'function') {
		try {
			var ptr = M._OpenJKDed_GetStatusJson();
			if (ptr) {
				var raw = M.UTF8ToString(ptr);
				var state = JSON.parse(raw);
				self.postMessage({ type: 'status', vip: lastVipStr, port: (M.wtSrcPort || state.netPort || 29070), state: state });

				// Auto-add requested bots once the map has finished loading
				if (!autoBotsAdded && pendingAutoBots > 0 && state.running && state.mapname) {
					autoBotsAdded = true;
					var count = Math.min(pendingAutoBots, Math.max(0, (state.maxClients || 16) - 1));
					for (var bi = 0; bi < count; bi++) {
						var botName = DEFAULT_BOT_ROSTER[bi % DEFAULT_BOT_ROSTER.length];
						M.ccall('OpenJKDed_ExecCommand', null, ['string'], ['addbot "' + botName + '" ' + pendingBotSkill]);
					}
				}
			}
		} catch (e) { }
	}
}

async function bootServer(cfg) {
	if (workerStarted) return;
	workerStarted = true;
	pendingAutoBots = Math.max(0, parseInt(cfg.botCount, 10) || 0);
	pendingBotSkill = Math.max(1, Math.min(5, parseInt(cfg.botSkill, 10) || 2));
	autoBotsAdded = false;

	postPhase('Initializing dedicated server Wasm runtime...');

	self.Module = {
		noInitialRun: true,
		print: function (text) { postLog('out', text); },
		printErr: function (text) { postLog('err', text); },
		onAbort: function (what) {
			postLog('err', 'Server aborted: ' + what);
			self.postMessage({ type: 'error', message: 'Dedicated server aborted: ' + what });
		},
		onRuntimeInitialized: async function () {
			try {
				var fs = getWasmFs();
				ensureDirTree(fs, '/base');
				ensureDirTree(fs, '/saves');

				// Restore saved configs/databases from OPFS
				var savesDir = await getOpfsDir('openjk-saves', [], false);
				if (savesDir) {
					postPhase('Restoring server saves...');
					await hydrateOpfsDir(savesDir, '/saves', fs);
				}

				// Fetch base jampgamewasm32.wasm
				postPhase('Fetching base server game module...');
				var gameResp = await fetch('jampgamewasm32.wasm');
				if (!gameResp || !gameResp.ok) {
					throw new Error('Failed to fetch jampgamewasm32.wasm (' + (gameResp ? gameResp.status : 'network') + ')');
				}
				writeWasmFile(fs, '/base/jampgamewasm32.wasm', new Uint8Array(await gameResp.arrayBuffer()));

				// Hydrate base assets (session / OPFS / hosted manifest)
				var source = await hydrateBaseAssets(fs, cfg.sessionBaseFiles);
				if (!source) {
					self.postMessage({
						type: 'need_assets',
						message: 'No Jedi Academy base assets found. Load your GameData/base folder (or GOG installer) on the main page or below first.'
					});
					return;
				}
				postLog('sys', 'Base assets ready (' + source + ').');

				// Hydrate selected mod (e.g. JA++)
				if (cfg.mod) {
					await hydrateMod(fs, cfg.mod);
				}

				// Hydrate any extra add-on mods into /base
				if (Array.isArray(cfg.enabledBaseMods)) {
					for (var ei = 0; ei < cfg.enabledBaseMods.length; ei++) {
						var extra = sanitizeModName(cfg.enabledBaseMods[ei]);
						if (!extra || extra.toLowerCase() === sanitizeModName(cfg.mod).toLowerCase()) continue;
						var extraDir = await getOpfsDir('openjk-mods', [extra], false);
						if (extraDir) {
							await hydrateOpfsDir(extraDir, '/base', fs);
						}
					}
				}

				var args = buildServerArgs(cfg);
				postPhase('Starting OpenJK Dedicated Server...');
				postLog('sys', 'Launching openjkded ' + args.join(' '));
				self.Module.callMain(args);

				postPhase('Online');
				self.postMessage({ type: 'started' });
				statusTimer = setInterval(pollServerStatus, 750);
				pollServerStatus();
			} catch (err) {
				postLog('err', 'Boot error: ' + (err && err.stack ? err.stack : err));
				self.postMessage({ type: 'error', message: String(err && err.message ? err.message : err) });
			}
		}
	};

	try {
		importScripts('openjkded.wasm32.js');
	} catch (err) {
		self.postMessage({ type: 'error', message: 'Failed to load openjkded.wasm32.js: ' + err.message });
	}
}

self.onmessage = function (ev) {
	var msg = ev.data || {};
	if (msg.type === 'start') {
		bootServer(msg.config || {});
	} else if (msg.type === 'exec') {
		var M = self.Module;
		if (M && typeof M._OpenJKDed_ExecCommand === 'function' && msg.cmd) {
			M.ccall('OpenJKDed_ExecCommand', null, ['string'], [String(msg.cmd)]);
			setTimeout(pollServerStatus, 120);
		}
	} else if (msg.type === 'poll') {
		pollServerStatus();
	} else if (msg.type === 'stop') {
		if (statusTimer) { clearInterval(statusTimer); statusTimer = null; }
		if (self.Module && self.Module.wt) {
			try { self.Module.wt.close(); } catch (e) { }
		}
		self.postMessage({ type: 'stopped' });
		self.close();
	}
};
