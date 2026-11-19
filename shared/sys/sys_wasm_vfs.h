#ifndef SYS_WASM_VFS_H
#define SYS_WASM_VFS_H

#ifdef __cplusplus
extern "C" {
#endif

#ifdef __EMSCRIPTEN__

void Sys_WasmPersistSavePath(const char *osPath);
void Sys_WasmRemoveSavePath(const char *osPath);
void Sys_WasmSyncBrowserUrl(const char *fsGame, const char *connectAddr, const char *mapName);

/* Returns 1 if map assets are ready synchronously, 0 if async prefetch started (defer cmd). */
int Sys_WasmEnsureMapAssets(const char *mapName, const char *retryCmd);
int Sys_WasmIsMapPrefetchActive(void);
int Sys_WasmConsumePendingCommand(char *outBuf, int outBufSize);

/* Manifest virtual pak queries for FS_AddGameDirectory / FS_LoadManifestPak. */
int Sys_WasmGetManifestPakCount(const char *gameDir);
int Sys_WasmGetManifestPakInfo(int pakIdx, char *outBasename, int basenameLen, int *outNumEntries, int *outNumCrcs);
int Sys_WasmGetManifestPakCrcs(int pakIdx, int *outCrcs, int maxCrcs);
int Sys_WasmGetManifestPakEntry(int pakIdx, int entryIdx, char *outPath, int pathLen,
	unsigned long *outDataOfs, unsigned long *outCompLen, unsigned long *outUncompLen, int *outMethod);

/* Synchronous HTTP Range fallback + zlib inflate into <baseDir>/<gameDir>/<qpath>. */
int Sys_WasmFetchManifestEntry(const char *pakBasename, const char *qpath,
	unsigned long dataOfs, unsigned long compLen, unsigned long uncompLen, int method,
	const char *baseDir, const char *gameDir);

#else

static inline void Sys_WasmPersistSavePath(const char *osPath) { (void)osPath; }
static inline void Sys_WasmRemoveSavePath(const char *osPath) { (void)osPath; }
static inline void Sys_WasmSyncBrowserUrl(const char *fsGame, const char *connectAddr, const char *mapName) {
	(void)fsGame; (void)connectAddr; (void)mapName;
}
static inline int Sys_WasmEnsureMapAssets(const char *mapName, const char *retryCmd) { (void)mapName; (void)retryCmd; return 1; }
static inline int Sys_WasmIsMapPrefetchActive(void) { return 0; }
static inline int Sys_WasmConsumePendingCommand(char *outBuf, int outBufSize) { (void)outBuf; (void)outBufSize; return 0; }
static inline int Sys_WasmGetManifestPakCount(const char *gameDir) { (void)gameDir; return 0; }
static inline int Sys_WasmGetManifestPakInfo(int pakIdx, char *outBasename, int basenameLen, int *outNumEntries, int *outNumCrcs) {
	(void)pakIdx; (void)outBasename; (void)basenameLen; (void)outNumEntries; (void)outNumCrcs; return 0;
}
static inline int Sys_WasmGetManifestPakCrcs(int pakIdx, int *outCrcs, int maxCrcs) { (void)pakIdx; (void)outCrcs; (void)maxCrcs; return 0; }
static inline int Sys_WasmGetManifestPakEntry(int pakIdx, int entryIdx, char *outPath, int pathLen,
	unsigned long *outDataOfs, unsigned long *outCompLen, unsigned long *outUncompLen, int *outMethod) {
	(void)pakIdx; (void)entryIdx; (void)outPath; (void)pathLen;
	(void)outDataOfs; (void)outCompLen; (void)outUncompLen; (void)outMethod;
	return 0;
}
static inline int Sys_WasmFetchManifestEntry(const char *pakBasename, const char *qpath,
	unsigned long dataOfs, unsigned long compLen, unsigned long uncompLen, int method,
	const char *baseDir, const char *gameDir) {
	(void)pakBasename; (void)qpath; (void)dataOfs; (void)compLen;
	(void)uncompLen; (void)method; (void)baseDir; (void)gameDir;
	return 0;
}

#endif /* __EMSCRIPTEN__ */

#ifdef __cplusplus
}
#endif

#endif /* SYS_WASM_VFS_H */
