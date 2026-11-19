(function (root) {
	'use strict';

	// Marker for Inno Setup zlib chunks: "zlb\x1a" + standard zlib header 0x78 0x9c
	var ZLB_HDR = [0x7a, 0x6c, 0x62, 0x1a, 0x78, 0x9c];

	// Known asset signatures inside the first local file header of Jedi Academy assets0..3.pk3
	var ASSET_SPECS = [
		{ name: 'assets0.pk3', firstEntry: 'scripts/academy1/', fastOffset: 0x64492ca },
		{ name: 'assets1.pk3', firstEntry: 'textures/bespin/', fastOffset: 0x27911d9f },
		{ name: 'assets2.pk3', firstEntry: 'productid.txt', fastOffset: 0x4e6a6cc4 },
		{ name: 'assets3.pk3', firstEntry: 'scripts/t1_fatal/', fastOffset: 0x4e7b4916 }
	];

	function isGogExeFile(file) {
		return !!(file && /\.exe$/i.test(String(file.name || '')));
	}

	function asciiIndexOf(bytes, str) {
		var n = bytes.length;
		var m = str.length;
		if (m === 0 || m > n) return -1;
		outer: for (var i = 0; i <= n - m; i++) {
			for (var j = 0; j < m; j++) {
				if (bytes[i + j] !== str.charCodeAt(j)) continue outer;
			}
			return i;
		}
		return -1;
	}

	function checkPk3FirstEntry(headBytes, expectedPrefix) {
		if (!headBytes || headBytes.length < 30 + expectedPrefix.length) return false;
		if (headBytes[0] !== 0x50 || headBytes[1] !== 0x4b || headBytes[2] !== 0x03 || headBytes[3] !== 0x04) {
			return false;
		}
		var nameLen = headBytes[26] | (headBytes[27] << 8);
		if (nameLen < expectedPrefix.length || 30 + expectedPrefix.length > headBytes.length) {
			return false;
		}
		for (var i = 0; i < expectedPrefix.length; i++) {
			if (headBytes[30 + i] !== expectedPrefix.charCodeAt(i)) return false;
		}
		return true;
	}

	function hasZipEocdNearEnd(tailBytes) {
		if (!tailBytes || tailBytes.length < 22) return false;
		var start = Math.max(0, tailBytes.length - 65557);
		for (var p = tailBytes.length - 22; p >= start; p--) {
			if (tailBytes[p] === 0x50 && tailBytes[p + 1] === 0x4b &&
				tailBytes[p + 2] === 0x05 && tailBytes[p + 3] === 0x06) {
				return true;
			}
		}
		return false;
	}

	async function peekZlibHead(blob, offset, maxSlice) {
		var slice = blob.slice(offset + 4, Math.min(blob.size, offset + 4 + (maxSlice || 4096)));
		var ds = new DecompressionStream('deflate');
		var writer = ds.writable.getWriter();
		var reader = ds.readable.getReader();
		var out = new Uint8Array(0);
		var writePromise = slice.arrayBuffer().then(function (ab) {
			return writer.write(new Uint8Array(ab));
		}).catch(function () { });

		try {
			while (out.length < 96) {
				var res = await reader.read();
				if (res.done || !res.value) break;
				var next = new Uint8Array(out.length + res.value.length);
				next.set(out, 0);
				next.set(res.value, out.length);
				out = next;
			}
		} catch (e) {
			// Truncated stream after first chunk is expected when peeking
		} finally {
			reader.cancel().catch(function () { });
			writer.abort().catch(function () { });
			await writePromise;
		}
		return out;
	}

	async function inflateZlibSlice(blob, start, end) {
		var slice = blob.slice(start, end);
		var stream = slice.stream().pipeThrough(new DecompressionStream('deflate'));
		var ab = await new Response(stream).arrayBuffer();
		return new Uint8Array(ab);
	}

	async function scanZlbOffsets(file, onProgress) {
		var CHUNK = 16 * 1024 * 1024;
		var OVERLAP = 16;
		var offsets = [];
		var pos = 0;
		var total = file.size;

		while (pos < total) {
			var readStart = Math.max(0, pos - OVERLAP);
			var readEnd = Math.min(total, pos + CHUNK);
			var buf = new Uint8Array(await file.slice(readStart, readEnd).arrayBuffer());
			var minIdx = pos === 0 ? 0 : (OVERLAP - 5);
			var limit = buf.length - 6;

			for (var i = Math.max(0, minIdx); i <= limit; i++) {
				if (buf[i] === ZLB_HDR[0] &&
					buf[i + 1] === ZLB_HDR[1] &&
					buf[i + 2] === ZLB_HDR[2] &&
					buf[i + 3] === ZLB_HDR[3] &&
					buf[i + 4] === ZLB_HDR[4] &&
					buf[i + 5] === ZLB_HDR[5]) {
					var absOff = readStart + i;
					if (offsets.length === 0 || offsets[offsets.length - 1] !== absOff) {
						offsets.push(absOff);
					}
					i += 5;
				}
			}

			pos = readEnd;
			if (onProgress) {
				var pct = Math.round((pos * 100) / total);
				onProgress('Scanning GOG installer (' + pct + '%)...');
			}
		}
		return offsets;
	}

	async function locateAssetStartIndices(file, offsets, onProgress) {
		var found = new Array(ASSET_SPECS.length).fill(-1);
		var offsetToIdx = new Map();
		for (var i = 0; i < offsets.length; i++) {
			offsetToIdx.set(offsets[i], i);
		}

		// Fast path: check known GOG Jedi Academy 1.01 installer offsets first (<5ms)
		var allFast = true;
		for (var s = 0; s < ASSET_SPECS.length; s++) {
			var idx = offsetToIdx.get(ASSET_SPECS[s].fastOffset);
			if (idx === undefined) {
				allFast = false;
				break;
			}
			var head = await peekZlibHead(file, offsets[idx], 4096);
			if (!checkPk3FirstEntry(head, ASSET_SPECS[s].firstEntry)) {
				allFast = false;
				break;
			}
			found[s] = idx;
		}
		if (allFast) return found;

		// General fallback: scan all zlb\x1a chunks for the 4 Jedi Academy PK3 headers
		found.fill(-1);
		var remaining = ASSET_SPECS.length;
		for (var k = 0; k < offsets.length && remaining > 0; k++) {
			if (onProgress && (k % 25 === 0)) {
				onProgress('Locating game archives (' + k + '/' + offsets.length + ')...');
			}
			var peek = await peekZlibHead(file, offsets[k], 4096);
			if (!peek || peek.length < 34 || peek[0] !== 0x50 || peek[1] !== 0x4b || peek[2] !== 0x03 || peek[3] !== 0x04) {
				continue;
			}
			for (var j = 0; j < ASSET_SPECS.length; j++) {
				if (found[j] === -1 && checkPk3FirstEntry(peek, ASSET_SPECS[j].firstEntry)) {
					found[j] = k;
					remaining--;
					break;
				}
			}
		}
		return found;
	}

	/**
	 * Extract assets0.pk3..assets3.pk3 from the GOG Jedi Academy Inno Setup .exe.
	 *
	 * @param {File|Blob} file - Selected GOG installer .exe
	 * @param {Object} handlers
	 * @param {function(string):void} [handlers.onProgress]
	 * @param {function(string):Promise<{write:function(Uint8Array):Promise<void>, close:function():Promise<void>}>} handlers.createSink
	 * @returns {Promise<{files: Array<{name:string, size:number}>}>}
	 */
	async function extractGogInstaller(file, handlers) {
		if (typeof DecompressionStream !== 'function') {
			throw new Error('This browser does not support native zlib DecompressionStream required to unpack the GOG installer.');
		}
		var onProgress = handlers && handlers.onProgress;
		var createSink = handlers && handlers.createSink;
		if (typeof createSink !== 'function') {
			throw new Error('Missing createSink handler.');
		}

		// Quick sanity check: PE header ("MZ") + Inno Setup signature in first 1 MB
		if (onProgress) onProgress('Verifying GOG installer...');
		var probeHead = new Uint8Array(await file.slice(0, Math.min(file.size, 1024 * 1024)).arrayBuffer());
		if (probeHead.length < 64 || probeHead[0] !== 0x4d || probeHead[1] !== 0x5a) {
			throw new Error('Selected .exe is not a valid Windows installer.');
		}
		if (asciiIndexOf(probeHead, 'Inno Setup') === -1) {
			throw new Error('This .exe does not appear to be the GOG Jedi Academy Inno Setup installer.');
		}

		var offsets = await scanZlbOffsets(file, onProgress);
		if (offsets.length < 4) {
			throw new Error('Could not find compressed game data inside this installer.');
		}

		var starts = await locateAssetStartIndices(file, offsets, onProgress);
		for (var i = 0; i < ASSET_SPECS.length; i++) {
			if (starts[i] === -1) {
				throw new Error('Could not locate ' + ASSET_SPECS[i].name + ' inside this installer. Make sure you selected the GOG Jedi Academy installer.');
			}
		}

		// Build ordered extraction ranges [startIdx, endIdx)
		var ordered = ASSET_SPECS.map(function (spec, idx) {
			return { name: spec.name, startIdx: starts[idx], endIdx: -1 };
		}).sort(function (a, b) { return a.startIdx - b.startIdx; });

		for (var oi = 0; oi < ordered.length - 1; oi++) {
			ordered[oi].endIdx = ordered[oi + 1].startIdx;
		}
		// Last archive (assets3.pk3): stop when a chunk ends with ZIP EOCD or is not a 10 MiB full block
		ordered[ordered.length - 1].endIdx = offsets.length;

		var results = [];
		for (var fi = 0; fi < ordered.length; fi++) {
			var item = ordered[fi];
			var isLastArchive = (fi === ordered.length - 1);
			var sink = await createSink(item.name);
			var totalBytes = 0;
			var lastChunkTail = new Uint8Array(0);
			var c = item.startIdx;
			var chunkTotal = isLastArchive ? 1 : (item.endIdx - item.startIdx);
			var chunkDone = 0;

			try {
				while (c < item.endIdx) {
					// Defensive merge: if a random 6-byte zlb\x1a\x78\x9c ever occurred inside a compressed chunk,
					// try advancing nextIdx until inflate succeeds cleanly.
					var decompressed = null;
					var nextIdx = c + 1;
					var maxLookahead = Math.min(item.endIdx, c + 4);
					while (nextIdx <= maxLookahead) {
						var sliceEnd = (nextIdx < offsets.length) ? offsets[nextIdx] : file.size;
						try {
							decompressed = await inflateZlibSlice(file, offsets[c] + 4, sliceEnd);
							break;
						} catch (inflateErr) {
							nextIdx++;
						}
					}
					if (!decompressed) {
						throw new Error('Failed to decompress chunk ' + c + ' while extracting ' + item.name);
					}

					await sink.write(decompressed);
					totalBytes += decompressed.length;
					chunkDone++;

					// Keep trailing 64KB to verify ZIP End-Of-Central-Directory
					if (decompressed.length >= 65560) {
						lastChunkTail = decompressed.subarray(decompressed.length - 65560);
					} else {
						var combined = new Uint8Array(Math.min(65560, lastChunkTail.length + decompressed.length));
						var keepPrev = combined.length - decompressed.length;
						if (keepPrev > 0) {
							combined.set(lastChunkTail.subarray(lastChunkTail.length - keepPrev), 0);
						}
						combined.set(decompressed, keepPrev);
						lastChunkTail = combined;
					}

					if (onProgress) {
						var pct = isLastArchive ? 100 : Math.min(100, Math.round((chunkDone * 100) / Math.max(1, chunkTotal)));
						onProgress('Unpacking ' + item.name + ' (' + (fi + 1) + '/' + ordered.length + ') — ' + pct + '%...');
					}

					c = nextIdx;
					if (isLastArchive && (decompressed.length !== 10485760 || hasZipEocdNearEnd(lastChunkTail))) {
						break;
					}
				}
			} finally {
				await sink.close();
			}

			if (!hasZipEocdNearEnd(lastChunkTail)) {
				throw new Error('Extracted ' + item.name + ' appears incomplete (missing ZIP directory).');
			}
			results.push({ name: item.name, size: totalBytes });
		}

		return { files: results };
	}

	var api = {
		isGogExeFile: isGogExeFile,
		extractGogInstaller: extractGogInstaller
	};

	if (typeof module !== 'undefined' && module.exports) {
		module.exports = api;
	}
	if (root) {
		root.OpenJKGogExtractor = api;
	}
})(typeof window !== 'undefined' ? window : (typeof globalThis !== 'undefined' ? globalThis : null));
