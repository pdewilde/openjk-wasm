#include "sys_wasm_vfs.h"

#ifdef __EMSCRIPTEN__

#include <emscripten.h>
#include <emscripten/wasmfs.h>
#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <zlib.h>

extern "C" void wasmfs_before_preload(void)
{
	backend_t baseMem = wasmfs_create_memory_backend();
	backend_t baseIcase = baseMem ? wasmfs_create_icase_backend(baseMem) : NULL;
	if (baseIcase) {
		wasmfs_create_directory("/base", 0777, baseIcase);
	} else {
		mkdir("/base", 0777);
	}
}

static void EnsureParentDirs(const char *fullPath)
{
	char tmp[1024];
	size_t len = strlen(fullPath);
	if (len == 0 || len >= sizeof(tmp)) {
		return;
	}
	memcpy(tmp, fullPath, len + 1);
	for (char *p = tmp + 1; *p; p++) {
		if (*p == '/') {
			*p = '\0';
			mkdir(tmp, 0777);
			*p = '/';
		}
	}
}

void Sys_WasmPersistSavePath(const char *osPath)
{
	if (!osPath || strncmp(osPath, "/saves/", 7) != 0) {
		return;
	}
	EM_ASM({
		var p = UTF8ToString($0);
		if (typeof window !== 'undefined' && typeof window.__openjkOpfsSyncFile === 'function') {
			window.__openjkOpfsSyncFile(p);
		}
	}, osPath);
}

void Sys_WasmRemoveSavePath(const char *osPath)
{
	if (!osPath || strncmp(osPath, "/saves/", 7) != 0) {
		return;
	}
	EM_ASM({
		var p = UTF8ToString($0);
		if (typeof window !== 'undefined' && typeof window.__openjkOpfsRemoveFile === 'function') {
			window.__openjkOpfsRemoveFile(p);
		}
	}, osPath);
}

void Sys_WasmSyncBrowserUrl(const char *fsGame, const char *connectAddr, const char *mapName)
{
	EM_ASM({
		if (typeof window !== 'undefined' && typeof window.__openjkSyncUrl === 'function') {
			window.__openjkSyncUrl(UTF8ToString($0), UTF8ToString($1), UTF8ToString($2));
		}
	}, fsGame ? fsGame : "", connectAddr ? connectAddr : "", mapName ? mapName : "");
}

int Sys_WasmEnsureMapAssets(const char *mapName, const char *retryCmd)
{
	if (!mapName || !mapName[0]) {
		return 1;
	}
	return EM_ASM_INT({
		var m = UTF8ToString($0);
		var cmd = $1 ? UTF8ToString($1) : "";
		if (typeof window !== 'undefined' && typeof window.__openjkEnsureMapAssets === 'function') {
			return window.__openjkEnsureMapAssets(m, cmd) ? 1 : 0;
		}
		return 1;
	}, mapName, retryCmd ? retryCmd : "");
}

int Sys_WasmIsMapPrefetchActive(void)
{
	return EM_ASM_INT({
		if (typeof window !== 'undefined' && typeof window.__openjkIsMapPrefetchActive === 'function') {
			return window.__openjkIsMapPrefetchActive() ? 1 : 0;
		}
		return 0;
	});
}

int Sys_WasmConsumePendingCommand(char *outBuf, int outBufSize)
{
	if (!outBuf || outBufSize <= 1) {
		return 0;
	}
	outBuf[0] = '\0';
	return EM_ASM_INT({
		if (typeof window !== 'undefined' && typeof window.__openjkConsumePendingCommand === 'function') {
			var cmd = window.__openjkConsumePendingCommand();
			if (cmd && cmd.length > 0) {
				stringToUTF8(cmd, $0, $1);
				return 1;
			}
		}
		return 0;
	}, outBuf, outBufSize);
}

int Sys_WasmGetManifestPakCount(const char *gameDir)
{
	if (!gameDir) {
		return 0;
	}
	return EM_ASM_INT({
		var g = UTF8ToString($0).toLowerCase();
		if (typeof window !== 'undefined' && window.__openjkManifestApi) {
			return window.__openjkManifestApi.getPakCount(g) | 0;
		}
		return 0;
	}, gameDir);
}

int Sys_WasmGetManifestPakInfo(int pakIdx, char *outBasename, int basenameLen, int *outNumEntries, int *outNumCrcs)
{
	if (!outBasename || basenameLen <= 0 || !outNumEntries || !outNumCrcs) {
		return 0;
	}
	return EM_ASM_INT({
		var api = (typeof window !== 'undefined') ? window.__openjkManifestApi : null;
		if (!api) return 0;
		var info = api.getPakInfo($0);
		if (!info) return 0;
		stringToUTF8(info.basename, $1, $2);
		setValue($3, info.numEntries | 0, 'i32');
		setValue($4, info.numCrcs | 0, 'i32');
		return 1;
	}, pakIdx, outBasename, basenameLen, outNumEntries, outNumCrcs);
}

int Sys_WasmGetManifestPakCrcs(int pakIdx, int *outCrcs, int maxCrcs)
{
	if (!outCrcs || maxCrcs <= 0) {
		return 0;
	}
	return EM_ASM_INT({
		var api = (typeof window !== 'undefined') ? window.__openjkManifestApi : null;
		if (!api) return 0;
		var crcs = api.getPakCrcs($0);
		if (!crcs) return 0;
		var n = Math.min(crcs.length, $2);
		var idx = ($1 >>> 2);
		for (var i = 0; i < n; i++) {
			HEAP32[idx + i] = crcs[i] | 0;
		}
		return n;
	}, pakIdx, outCrcs, maxCrcs);
}

int Sys_WasmGetManifestPakEntry(int pakIdx, int entryIdx, char *outPath, int pathLen,
	unsigned long *outDataOfs, unsigned long *outCompLen, unsigned long *outUncompLen, int *outMethod)
{
	if (!outPath || pathLen <= 0 || !outDataOfs || !outCompLen || !outUncompLen || !outMethod) {
		return 0;
	}
	return EM_ASM_INT({
		var api = (typeof window !== 'undefined') ? window.__openjkManifestApi : null;
		if (!api) return 0;
		var e = api.getPakEntry($0, $1);
		if (!e) return 0;
		stringToUTF8(e[0], $2, $3);
		setValue($4, e[1] >>> 0, 'i32');
		setValue($5, e[2] >>> 0, 'i32');
		setValue($6, e[3] >>> 0, 'i32');
		setValue($7, e[4] | 0, 'i32');
		return 1;
	}, pakIdx, entryIdx, outPath, pathLen, outDataOfs, outCompLen, outUncompLen, outMethod);
}

int Sys_WasmFetchManifestEntry(const char *pakBasename, const char *qpath,
	unsigned long dataOfs, unsigned long compLen, unsigned long uncompLen, int method,
	const char *baseDir, const char *gameDir)
{
	if (!pakBasename || !qpath || !baseDir || !gameDir) {
		return 0;
	}
	if (strstr(qpath, "..") != NULL) {
		return 0;
	}

	char destPath[1024];
	const char *cleanQ = (qpath[0] == '/' || qpath[0] == '\\') ? qpath + 1 : qpath;
	if (strcmp(baseDir, "/") == 0) {
		snprintf(destPath, sizeof(destPath), "/%s/%s", gameDir, cleanQ);
	} else {
		snprintf(destPath, sizeof(destPath), "%s/%s/%s", baseDir, gameDir, cleanQ);
	}
	for (char *p = destPath; *p; p++) {
		if (*p == '\\') {
			*p = '/';
		}
	}

	EnsureParentDirs(destPath);

	struct stat existingSt;
	if (stat(destPath, &existingSt) == 0 && S_ISREG(existingSt.st_mode) &&
		(unsigned long)existingSt.st_size == uncompLen) {
		return 1;
	}

	if (compLen == 0 && uncompLen == 0) {
		FILE *fp = fopen(destPath, "wb");
		if (!fp) {
			return 0;
		}
		fclose(fp);
		return 1;
	}

	unsigned char *compBuf = (unsigned char *)malloc(compLen > 0 ? compLen : 1);
	if (!compBuf) {
		return 0;
	}

	int fetched = EM_ASM_INT({
		var api = (typeof window !== 'undefined') ? window.__openjkManifestApi : null;
		if (!api) return 0;
		var bytes = api.fetchRangeSync(UTF8ToString($0), $1 >>> 0, $2 >>> 0);
		if (!bytes || bytes.length !== ($2 >>> 0)) return 0;
		HEAPU8.set(bytes, $3);
		return 1;
	}, pakBasename, (unsigned int)dataOfs, (unsigned int)compLen, compBuf);

	if (!fetched) {
		free(compBuf);
		return 0;
	}

	const unsigned char *writeBuf = compBuf;
	size_t writeLen = compLen;
	unsigned char *decompBuf = NULL;

	if (method == 8) {
		decompBuf = (unsigned char *)malloc(uncompLen > 0 ? uncompLen : 1);
		if (!decompBuf) {
			free(compBuf);
			return 0;
		}
		z_stream zs;
		memset(&zs, 0, sizeof(zs));
		zs.next_in = compBuf;
		zs.avail_in = (uInt)compLen;
		zs.next_out = decompBuf;
		zs.avail_out = (uInt)uncompLen;
		if (inflateInit2(&zs, -MAX_WBITS) != Z_OK) {
			free(decompBuf);
			free(compBuf);
			return 0;
		}
		int zret = inflate(&zs, Z_FINISH);
		inflateEnd(&zs);
		if (zret != Z_STREAM_END && zs.total_out != uncompLen) {
			free(decompBuf);
			free(compBuf);
			return 0;
		}
		writeBuf = decompBuf;
		writeLen = uncompLen;
	} else if (method != 0) {
		free(compBuf);
		return 0;
	}

	FILE *fp = fopen(destPath, "wb");
	int ok = 0;
	if (fp) {
		ok = (writeLen == 0 || fwrite(writeBuf, 1, writeLen, fp) == writeLen) ? 1 : 0;
		fclose(fp);
	}

	if (ok) {
		EM_ASM({
			var api = (typeof window !== 'undefined') ? window.__openjkManifestApi : null;
			if (api && typeof api.cachePutFile === 'function') {
				api.cachePutFile(UTF8ToString($0), UTF8ToString($1));
			}
		}, cleanQ, destPath);
	}

	free(decompBuf);
	free(compBuf);
	return ok;
}

#endif /* __EMSCRIPTEN__ */
