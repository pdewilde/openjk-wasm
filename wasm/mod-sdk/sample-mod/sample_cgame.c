/*
 * OpenJK WebAssembly Sample Mod — Custom SIDE_MODULE=2 cgamewasm32.wasm
 *
 * Demonstrates building a standalone native C WebAssembly side module that
 * OpenJK dynamically loads from /<fs_game>/cgamewasm32.wasm via dlopen().
 *
 * This sample delegates stock gameplay callbacks to /base/cgamewasm32.wasm
 * while injecting a custom startup banner and a native `/samplemod` console
 * command. Full-source MP mods (JA+, BaseJKA forks, etc.) compile their own
 * complete cgame/game/ui sources with the exact same SIDE_MODULE=2 flags.
 */

#include <dlfcn.h>
#include <stddef.h>
#include <string.h>
#include <strings.h>

#include "qcommon/q_shared.h"
#include "rd-common/tr_types.h"
#include "game/bg_public.h"
#include "cgame/cg_public.h"

static cgameImport_t *s_import = NULL;
static void *s_baseHandle = NULL;
static cgameExport_t s_baseExport;
static cgameExport_t s_modExport;

static void SampleMod_Init( int serverMessageNum, int serverCommandSequence, int clientNum ) {
	if ( s_import && s_import->Print ) {
		s_import->Print( "^3[SampleMod-WASM]^7 Custom SIDE_MODULE=2 cgamewasm32.wasm initialized! Type ^2/samplemod^7 in console.\n" );
	}
	if ( s_baseExport.Init ) {
		s_baseExport.Init( serverMessageNum, serverCommandSequence, clientNum );
	}
	if ( s_import && s_import->AddCommand ) {
		s_import->AddCommand( "samplemod" );
	}
}

static void SampleMod_Shutdown( void ) {
	if ( s_baseExport.Shutdown ) {
		s_baseExport.Shutdown();
	}
	if ( s_baseHandle ) {
		dlclose( s_baseHandle );
		s_baseHandle = NULL;
	}
}

static qboolean SampleMod_ConsoleCommand( void ) {
	if ( s_import && s_import->Cmd_Argv ) {
		char cmd[64];
		cmd[0] = '\0';
		s_import->Cmd_Argv( 0, cmd, (int)sizeof( cmd ) );
		if ( strcasecmp( cmd, "samplemod" ) == 0 ) {
			if ( s_import->Print ) {
				char fsGame[64];
				fsGame[0] = '\0';
				if ( s_import->Cvar_VariableStringBuffer ) {
					s_import->Cvar_VariableStringBuffer( "fs_game", fsGame, (int)sizeof( fsGame ) );
				}
				s_import->Print( "^2[SampleMod-WASM]^7 Native C mod command executed! (fs_game=%s)\n", fsGame[0] ? fsGame : "base" );
			}
			return qtrue;
		}
	}
	if ( s_baseExport.ConsoleCommand ) {
		return s_baseExport.ConsoleCommand();
	}
	return qfalse;
}

Q_EXPORT cgameExport_t *QDECL GetModuleAPI( int apiVersion, cgameImport_t *import ) {
	if ( apiVersion != CGAME_API_VERSION || !import ) {
		return NULL;
	}

	s_import = import;

	if ( s_import->Print ) {
		s_import->Print( "^3[SampleMod-WASM]^7 GetModuleAPI(v%d) invoked from mod cgamewasm32.wasm\n", apiVersion );
	}

	s_baseHandle = dlopen( "/base/cgamewasm32.wasm", RTLD_NOW | RTLD_LOCAL );
	if ( !s_baseHandle ) {
		if ( s_import->Print ) {
			s_import->Print( "^1[SampleMod-WASM]^7 dlopen(/base/cgamewasm32.wasm) failed: %s\n", dlerror() );
		}
		return NULL;
	}

	GetCGameAPI_t baseGetAPI = (GetCGameAPI_t)dlsym( s_baseHandle, "GetModuleAPI" );
	if ( !baseGetAPI ) {
		if ( s_import->Print ) {
			s_import->Print( "^1[SampleMod-WASM]^7 dlsym(GetModuleAPI) failed: %s\n", dlerror() );
		}
		dlclose( s_baseHandle );
		s_baseHandle = NULL;
		return NULL;
	}

	cgameExport_t *baseExport = baseGetAPI( apiVersion, import );
	if ( !baseExport ) {
		dlclose( s_baseHandle );
		s_baseHandle = NULL;
		return NULL;
	}

	s_baseExport = *baseExport;
	s_modExport = *baseExport;
	s_modExport.Init = SampleMod_Init;
	s_modExport.Shutdown = SampleMod_Shutdown;
	s_modExport.ConsoleCommand = SampleMod_ConsoleCommand;

	return &s_modExport;
}
