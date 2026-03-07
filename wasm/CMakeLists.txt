#============================================================================
# OpenJK WASM Build — Monolithic single-executable for Emscripten
#============================================================================

cmake_minimum_required(VERSION 3.13)
project(openjk_wasm C CXX)

set(CMAKE_CXX_STANDARD 11)
set(CMAKE_C_STANDARD 11)

# ============================================================================
# Paths — this CMakeLists lives in wasm/, so go up one level
# ============================================================================
set(ProjectRoot "${CMAKE_SOURCE_DIR}/..")
set(SPDir "${ProjectRoot}/code")
set(SharedDir "${ProjectRoot}/shared")
set(OpenJKLibDir "${ProjectRoot}/lib")
set(GSLIncludeDirectory "${OpenJKLibDir}/gsl-lite/include")
set(GL4ESDir "${OpenJKLibDir}/gl4es")

# ============================================================================
# Version header (generate a stub so the build doesn't fail)
# ============================================================================
if(NOT EXISTS "${CMAKE_BINARY_DIR}/shared/qcommon/q_version.h")
    file(MAKE_DIRECTORY "${CMAKE_BINARY_DIR}/shared/qcommon")
    file(WRITE "${CMAKE_BINARY_DIR}/shared/qcommon/q_version.h"
        "#pragma once\n"
        "#define SOURCE_DATE \"Mar 2 2026\"\n"
        "#define GIT_HASH \"wasm-build\"\n"
    )
endif()

# ============================================================================
# Compile definitions
# ============================================================================
add_compile_definitions(
    _JK2EXE
    SP_GAME
    __EMSCRIPTEN__
    __linux__
    ARCH_STRING="wasm32"
    RENDERER          # needed for rd-vanilla sources
    DEFAULT_ES=2      # needed for GL4ES to use GLES2 backend
)

# ============================================================================
# Include directories
# ============================================================================
include_directories(
    ${SharedDir}
    ${CMAKE_BINARY_DIR}/shared
    ${SPDir}
    "${SPDir}/game"
    "${SPDir}/rd-common"
    "${SPDir}/rd-vanilla"
    "${GSLIncludeDirectory}"
    "${OpenJKLibDir}"
    # Bundled minizip
    "${OpenJKLibDir}/minizip/include"
    "${OpenJKLibDir}/minizip/include/minizip"
    "${GL4ESDir}/include"
)

# ============================================================================
# Shared common files
# ============================================================================
set(SharedCommonFiles
    "${SharedDir}/qcommon/q_color.h"
    "${SharedDir}/qcommon/q_color.c"
    "${SharedDir}/qcommon/q_math.h"
    "${SharedDir}/qcommon/q_math.c"
    "${SharedDir}/qcommon/q_string.h"
    "${SharedDir}/qcommon/q_string.c"
    "${SharedDir}/qcommon/q_platform.h"
    "${CMAKE_BINARY_DIR}/shared/qcommon/q_version.h"
)

set(SharedCommonSafeFiles
    "${SharedDir}/qcommon/safe/gsl.h"
    "${SharedDir}/qcommon/safe/string.cpp"
    "${SharedDir}/qcommon/safe/string.h"
    "${SharedDir}/qcommon/safe/sscanf.h"
    "${SharedDir}/qcommon/safe/limited_vector.h"
)

# ============================================================================
# ENGINE source files (from code/CMakeLists.txt SPEngineFiles)
# ============================================================================

# Client
set(EngineClientFiles
    "${SPDir}/client/keys.h"
    "${SPDir}/client/keycodes.h"
    "${SPDir}/client/cl_cgame.cpp"
    "${SPDir}/client/cl_cin.cpp"
    "${SPDir}/client/cl_console.cpp"
    "${SPDir}/client/cl_input.cpp"
    "${SPDir}/client/cl_keys.cpp"
    "${SPDir}/client/cl_main.cpp"
    "${SPDir}/client/cl_mp3.cpp"
    "${SPDir}/client/cl_parse.cpp"
    "${SPDir}/client/cl_scrn.cpp"
    "${SPDir}/client/cl_ui.cpp"
    "${SPDir}/client/cl_mp3.h"
    "${SPDir}/client/client.h"
    "${SPDir}/client/client_ui.h"
    "${SPDir}/client/snd_ambient.cpp"
    "${SPDir}/client/snd_dma.cpp"
    "${SPDir}/client/snd_mem.cpp"
    "${SPDir}/client/snd_mix.cpp"
    "${SPDir}/client/snd_music.cpp"
    "${SPDir}/client/snd_ambient.h"
    "${SPDir}/client/snd_local.h"
    "${SPDir}/client/snd_music.h"
    "${SPDir}/client/snd_public.h"
    "${SPDir}/client/vmachine.cpp"
    "${SPDir}/client/vmachine.h"
)

# Qcommon
set(EngineCommonFiles
    "${SPDir}/qcommon/cm_load.cpp"
    "${SPDir}/qcommon/cm_patch.cpp"
    "${SPDir}/qcommon/cm_polylib.cpp"
    "${SPDir}/qcommon/cm_test.cpp"
    "${SPDir}/qcommon/cm_trace.cpp"
    "${SPDir}/qcommon/cm_local.h"
    "${SPDir}/qcommon/cm_patch.h"
    "${SPDir}/qcommon/cm_polylib.h"
    "${SPDir}/qcommon/cm_public.h"
    "${SPDir}/qcommon/cmd.cpp"
    "${SPDir}/qcommon/common.cpp"
    "${SPDir}/qcommon/cvar.cpp"
    "${SPDir}/qcommon/files.cpp"
    "${SPDir}/qcommon/md4.cpp"
    "${SPDir}/qcommon/msg.cpp"
    "${SPDir}/qcommon/net_chan.cpp"
    "${SPDir}/qcommon/q_shared.cpp"
    "${SPDir}/qcommon/q_shared.h"
    "${SPDir}/qcommon/sstring.h"
    "${SPDir}/qcommon/stringed_ingame.cpp"
    "${SPDir}/qcommon/stringed_interface.cpp"
    "${SPDir}/qcommon/stringed_ingame.h"
    "${SPDir}/qcommon/stringed_interface.h"
    "${SPDir}/qcommon/strip.cpp"
    "${SPDir}/qcommon/persistence.cpp"
    "${SPDir}/qcommon/tags.h"
    "${SPDir}/qcommon/z_memman_pc.cpp"
    "${SPDir}/qcommon/qfiles.h"
    "${SPDir}/qcommon/qcommon.h"
    "${SPDir}/qcommon/stv_version.h"
    "${SPDir}/qcommon/timing.h"
    "${SPDir}/qcommon/MiniHeap.h"
    "${SPDir}/qcommon/ojk_i_saved_game.h"
    "${SPDir}/qcommon/ojk_saved_game.h"
    "${SPDir}/qcommon/ojk_saved_game.cpp"
    "${SPDir}/qcommon/ojk_saved_game_helper.h"
    "${SPDir}/qcommon/ojk_saved_game_helper_fwd.h"
    "${SPDir}/qcommon/ojk_scope_guard.h"
    ${SharedCommonFiles}
)

set(EngineCommonSafeFiles
    "${SharedDir}/qcommon/safe/files.cpp"
    "${SharedDir}/qcommon/safe/files.h"
    ${SharedCommonSafeFiles}
    "${SPDir}/qcommon/safe/memory.h"
)

# Server
set(EngineServerFiles
    "${SPDir}/server/exe_headers.cpp"
    "${SPDir}/server/sv_ccmds.cpp"
    "${SPDir}/server/sv_client.cpp"
    "${SPDir}/server/sv_game.cpp"
    "${SPDir}/server/sv_init.cpp"
    "${SPDir}/server/sv_main.cpp"
    "${SPDir}/server/sv_savegame.cpp"
    "${SPDir}/server/sv_snapshot.cpp"
    "${SPDir}/server/sv_world.cpp"
    "${SPDir}/server/exe_headers.h"
    "${SPDir}/server/server.h"
)

# Engine game headers (just headers, not the full game DLL)
set(EngineGameFiles
    "${SPDir}/game/genericparser2.cpp"
    "${SPDir}/game/channels.h"
    "${SPDir}/game/anims.h"
    "${SPDir}/game/b_public.h"
    "${SPDir}/game/bg_public.h"
    "${SPDir}/game/bset.h"
    "${SPDir}/game/bstate.h"
    "${SPDir}/game/common_headers.h"
    "${SPDir}/game/g_functions.h"
    "${SPDir}/game/g_items.h"
    "${SPDir}/game/g_local.h"
    "${SPDir}/game/g_nav.h"
    "${SPDir}/game/g_public.h"
    "${SPDir}/game/g_shared.h"
    "${SPDir}/game/genericparser2.h"
    "${SPDir}/game/statindex.h"
    "${SPDir}/game/teams.h"
    "${SPDir}/game/weapons.h"
    "${SPDir}/game/surfaceflags.h"
    "${SPDir}/game/ghoul2_shared.h"
)

# MP3 decoder
set(EngineMP3Files
    "${SPDir}/mp3code/cdct.c"
    "${SPDir}/mp3code/csbt.c"
    "${SPDir}/mp3code/csbtb.c"
    "${SPDir}/mp3code/csbtl3.c"
    "${SPDir}/mp3code/cup.c"
    "${SPDir}/mp3code/cupini.c"
    "${SPDir}/mp3code/cupl1.c"
    "${SPDir}/mp3code/cupl3.c"
    "${SPDir}/mp3code/cwin.c"
    "${SPDir}/mp3code/cwinb.c"
    "${SPDir}/mp3code/cwinm.c"
    "${SPDir}/mp3code/hwin.c"
    "${SPDir}/mp3code/l3dq.c"
    "${SPDir}/mp3code/l3init.c"
    "${SPDir}/mp3code/mdct.c"
    "${SPDir}/mp3code/mhead.c"
    "${SPDir}/mp3code/msis.c"
    "${SPDir}/mp3code/towave.c"
    "${SPDir}/mp3code/uph.c"
    "${SPDir}/mp3code/upsf.c"
    "${SPDir}/mp3code/wavep.c"
)

# Engine renderer headers
set(EngineRendererFiles
    "${SPDir}/rd-common/mdx_format.h"
    "${SPDir}/rd-common/tr_public.h"
    "${SPDir}/rd-common/tr_types.h"
)

# UI
set(EngineUIFiles
    "${SPDir}/ui/ui_atoms.cpp"
    "${SPDir}/ui/ui_connect.cpp"
    "${SPDir}/ui/ui_main.cpp"
    "${SPDir}/ui/ui_saber.cpp"
    "${SPDir}/ui/ui_shared.cpp"
    "${SPDir}/ui/ui_syscalls.cpp"
)

# Ghoul2 headers
set(EngineGhoul2Files
    "${SPDir}/ghoul2/G2.h"
    "${SPDir}/ghoul2/ghoul2_gore.h"
)

# CGame headers (engine only includes these)
set(EngineCGameFiles
    "${SPDir}/cgame/cg_public.h"
    "${SPDir}/cgame/animtable.h"
)

# Sys / SDL Platform files
set(EngineSysFiles
    "${SharedDir}/sys/sys_main.cpp"
    "${SharedDir}/sys/sys_event.cpp"
    "${SharedDir}/sys/sys_local.h"
    "${SharedDir}/sys/sys_public.h"
    "${SharedDir}/sys/sys_loadlib.h"
    "${SharedDir}/sys/con_local.h"
    "${SharedDir}/sys/con_log.cpp"
    "${SharedDir}/sdl/sdl_qgl.h"
    "${SharedDir}/sdl/sdl_sound.cpp"
    "${SharedDir}/sdl/sdl_input.cpp"
    "${SharedDir}/sdl/sdl_window.cpp"
    "${SharedDir}/sdl/sdl_icon.h"
    # Unix-specific
    "${SharedDir}/sys/sys_unix.cpp"
    "${SharedDir}/sys/con_tty.cpp"
)

# ============================================================================
# RENDERER source files (from code/rd-vanilla/CMakeLists.txt)
# ============================================================================
set(RendererG2Files
    "${SPDir}/rd-vanilla/G2_API.cpp"
    "${SPDir}/rd-vanilla/G2_bolts.cpp"
    "${SPDir}/rd-vanilla/G2_bones.cpp"
    "${SPDir}/rd-vanilla/G2_misc.cpp"
    "${SPDir}/rd-vanilla/G2_surfaces.cpp"
)

set(RendererMainFiles
    "${SPDir}/rd-vanilla/glext.h"
    "${SPDir}/rd-vanilla/qgl.h"
    "${SPDir}/rd-vanilla/tr_arb.cpp"
    "${SPDir}/rd-vanilla/tr_backend.cpp"
    "${SPDir}/rd-vanilla/tr_bsp.cpp"
    "${SPDir}/rd-vanilla/tr_cmds.cpp"
    "${SPDir}/rd-vanilla/tr_curve.cpp"
    "${SPDir}/rd-vanilla/tr_draw.cpp"
    "${SPDir}/rd-vanilla/tr_ghoul2.cpp"
    "${SPDir}/rd-vanilla/tr_image.cpp"
    "${SPDir}/rd-vanilla/tr_init.cpp"
    "${SPDir}/rd-vanilla/tr_light.cpp"
    "${SPDir}/rd-vanilla/tr_local.h"
    "${SPDir}/rd-vanilla/tr_main.cpp"
    "${SPDir}/rd-vanilla/tr_marks.cpp"
    "${SPDir}/rd-vanilla/tr_mesh.cpp"
    "${SPDir}/rd-vanilla/tr_model.cpp"
    "${SPDir}/rd-vanilla/tr_quicksprite.cpp"
    "${SPDir}/rd-vanilla/tr_quicksprite.h"
    "${SPDir}/rd-vanilla/tr_scene.cpp"
    "${SPDir}/rd-vanilla/tr_shade.cpp"
    "${SPDir}/rd-vanilla/tr_shade_calc.cpp"
    "${SPDir}/rd-vanilla/tr_shader.cpp"
    "${SPDir}/rd-vanilla/tr_shadows.cpp"
    "${SPDir}/rd-vanilla/tr_skin.cpp"
    "${SPDir}/rd-vanilla/tr_sky.cpp"
    "${SPDir}/rd-vanilla/tr_stl.cpp"
    "${SPDir}/rd-vanilla/tr_stl.h"
    "${SPDir}/rd-vanilla/tr_subs.cpp"
    "${SPDir}/rd-vanilla/tr_surface.cpp"
    "${SPDir}/rd-vanilla/tr_surfacesprites.cpp"
    "${SPDir}/rd-vanilla/tr_world.cpp"
    "${SPDir}/rd-vanilla/tr_WorldEffects.cpp"
    "${SPDir}/rd-vanilla/tr_WorldEffects.h"
)

set(RendererRdCommonFiles
    "${SPDir}/rd-common/tr_common.h"
    "${SPDir}/rd-common/tr_font.cpp"
    "${SPDir}/rd-common/tr_font.h"
    "${SPDir}/rd-common/tr_image_load.cpp"
    "${SPDir}/rd-common/tr_image_jpg.cpp"
    "${SPDir}/rd-common/tr_image_tga.cpp"
    "${SPDir}/rd-common/tr_image_png.cpp"
    "${SPDir}/rd-common/tr_noise.cpp"
)

set(RendererCommonFiles
    "${SPDir}/qcommon/matcomp.cpp"
    "${SPDir}/qcommon/q_shared.cpp"
    "${SPDir}/qcommon/q_shared.h"
    "${SPDir}/qcommon/ojk_i_saved_game.h"
    "${SPDir}/qcommon/ojk_saved_game_class_archivers.h"
    "${SPDir}/qcommon/ojk_saved_game_helper.h"
    "${SPDir}/qcommon/ojk_saved_game_helper_fwd.h"
    "${SPDir}/qcommon/ojk_scope_guard.h"
)

# ============================================================================
# GAME DLL source files (from code/game/CMakeLists.txt)
# ============================================================================
set(GameDLLFiles
    "${SPDir}/game/AI_Animal.cpp"
    "${SPDir}/game/AI_AssassinDroid.cpp"
    "${SPDir}/game/AI_Atst.cpp"
    "${SPDir}/game/AI_BobaFett.cpp"
    "${SPDir}/game/AI_Civilian.cpp"
    "${SPDir}/game/AI_Default.cpp"
    "${SPDir}/game/AI_Droid.cpp"
    "${SPDir}/game/AI_GalakMech.cpp"
    "${SPDir}/game/AI_Grenadier.cpp"
    "${SPDir}/game/AI_HazardTrooper.cpp"
    "${SPDir}/game/AI_Howler.cpp"
    "${SPDir}/game/AI_ImperialProbe.cpp"
    "${SPDir}/game/AI_Interrogator.cpp"
    "${SPDir}/game/AI_Jedi.cpp"
    "${SPDir}/game/AI_Mark1.cpp"
    "${SPDir}/game/AI_Mark2.cpp"
    "${SPDir}/game/AI_MineMonster.cpp"
    "${SPDir}/game/AI_Rancor.cpp"
    "${SPDir}/game/AI_Remote.cpp"
    "${SPDir}/game/AI_RocketTrooper.cpp"
    "${SPDir}/game/AI_SaberDroid.cpp"
    "${SPDir}/game/AI_SandCreature.cpp"
    "${SPDir}/game/AI_Seeker.cpp"
    "${SPDir}/game/AI_Sentry.cpp"
    "${SPDir}/game/AI_Sniper.cpp"
    "${SPDir}/game/AI_Stormtrooper.cpp"
    "${SPDir}/game/AI_Tusken.cpp"
    "${SPDir}/game/AI_Utils.cpp"
    "${SPDir}/game/AI_Wampa.cpp"
    "${SPDir}/game/AnimalNPC.cpp"
    "${SPDir}/game/FighterNPC.cpp"
    "${SPDir}/game/G_Timer.cpp"
    "${SPDir}/game/NPC.cpp"
    "${SPDir}/game/NPC_behavior.cpp"
    "${SPDir}/game/NPC_combat.cpp"
    "${SPDir}/game/NPC_goal.cpp"
    "${SPDir}/game/NPC_misc.cpp"
    "${SPDir}/game/NPC_move.cpp"
    "${SPDir}/game/NPC_reactions.cpp"
    "${SPDir}/game/NPC_senses.cpp"
    "${SPDir}/game/NPC_sounds.cpp"
    "${SPDir}/game/NPC_spawn.cpp"
    "${SPDir}/game/NPC_stats.cpp"
    "${SPDir}/game/NPC_utils.cpp"
    "${SPDir}/game/Q3_Interface.cpp"
    "${SPDir}/game/SpeederNPC.cpp"
    "${SPDir}/game/WalkerNPC.cpp"
    "${SPDir}/game/bg_misc.cpp"
    "${SPDir}/game/bg_pangles.cpp"
    "${SPDir}/game/bg_panimate.cpp"
    "${SPDir}/game/bg_pmove.cpp"
    "${SPDir}/game/bg_slidemove.cpp"
    "${SPDir}/game/bg_vehicleLoad.cpp"
    "${SPDir}/game/g_active.cpp"
    "${SPDir}/game/g_breakable.cpp"
    "${SPDir}/game/g_camera.cpp"
    "${SPDir}/game/g_client.cpp"
    "${SPDir}/game/g_cmds.cpp"
    "${SPDir}/game/g_combat.cpp"
    "${SPDir}/game/g_emplaced.cpp"
    "${SPDir}/game/g_fx.cpp"
    "${SPDir}/game/g_functions.cpp"
    "${SPDir}/game/g_inventory.cpp"
    "${SPDir}/game/g_itemLoad.cpp"
    "${SPDir}/game/g_items.cpp"
    "${SPDir}/game/g_main.cpp"
    "${SPDir}/game/g_mem.cpp"
    "${SPDir}/game/g_misc.cpp"
    "${SPDir}/game/g_misc_model.cpp"
    "${SPDir}/game/g_missile.cpp"
    "${SPDir}/game/g_mover.cpp"
    "${SPDir}/game/g_nav.cpp"
    "${SPDir}/game/g_navigator.cpp"
    "${SPDir}/game/g_navnew.cpp"
    "${SPDir}/game/g_object.cpp"
    "${SPDir}/game/g_objectives.cpp"
    "${SPDir}/game/g_rail.cpp"
    "${SPDir}/game/g_ref.cpp"
    "${SPDir}/game/g_roff.cpp"
    "${SPDir}/game/g_savegame.cpp"
    "${SPDir}/game/g_session.cpp"
    "${SPDir}/game/g_spawn.cpp"
    "${SPDir}/game/g_svcmds.cpp"
    "${SPDir}/game/g_target.cpp"
    "${SPDir}/game/g_trigger.cpp"
    "${SPDir}/game/g_turret.cpp"
    "${SPDir}/game/g_usable.cpp"
    "${SPDir}/game/g_utils.cpp"
    "${SPDir}/game/g_vehicles.cpp"
    "${SPDir}/game/g_weapon.cpp"
    "${SPDir}/game/g_weaponLoad.cpp"
    "${SPDir}/game/genericparser2.cpp"
    "${SPDir}/game/wp_atst.cpp"
    "${SPDir}/game/wp_blaster_pistol.cpp"
    "${SPDir}/game/wp_blaster_rifle.cpp"
    "${SPDir}/game/wp_bot_laser.cpp"
    "${SPDir}/game/wp_bowcaster.cpp"
    "${SPDir}/game/wp_concussion.cpp"
    "${SPDir}/game/wp_demp2.cpp"
    "${SPDir}/game/wp_det_pack.cpp"
    "${SPDir}/game/wp_disruptor.cpp"
    "${SPDir}/game/wp_emplaced_gun.cpp"
    "${SPDir}/game/wp_flechette.cpp"
    "${SPDir}/game/wp_melee.cpp"
    "${SPDir}/game/wp_noghri_stick.cpp"
    "${SPDir}/game/wp_repeater.cpp"
    "${SPDir}/game/wp_rocket_launcher.cpp"
    "${SPDir}/game/wp_saber.cpp"
    "${SPDir}/game/wp_saberLoad.cpp"
    "${SPDir}/game/wp_stun_baton.cpp"
    "${SPDir}/game/wp_thermal.cpp"
    "${SPDir}/game/wp_trip_mine.cpp"
    "${SPDir}/game/wp_tusken.cpp"
)

# CGame files (from the game DLL build)
set(GameCGameFiles
    "${SPDir}/cgame/cg_camera.cpp"
    "${SPDir}/cgame/cg_consolecmds.cpp"
    "${SPDir}/cgame/cg_credits.cpp"
    "${SPDir}/cgame/cg_draw.cpp"
    "${SPDir}/cgame/cg_drawtools.cpp"
    "${SPDir}/cgame/cg_effects.cpp"
    "${SPDir}/cgame/cg_ents.cpp"
    "${SPDir}/cgame/cg_event.cpp"
    "${SPDir}/cgame/cg_headers.cpp"
    "${SPDir}/cgame/cg_info.cpp"
    "${SPDir}/cgame/cg_lights.cpp"
    "${SPDir}/cgame/cg_localents.cpp"
    "${SPDir}/cgame/cg_main.cpp"
    "${SPDir}/cgame/cg_marks.cpp"
    "${SPDir}/cgame/cg_players.cpp"
    "${SPDir}/cgame/cg_playerstate.cpp"
    "${SPDir}/cgame/cg_predict.cpp"
    "${SPDir}/cgame/cg_scoreboard.cpp"
    "${SPDir}/cgame/cg_servercmds.cpp"
    "${SPDir}/cgame/cg_snapshot.cpp"
    "${SPDir}/cgame/cg_syscalls.cpp"
    "${SPDir}/cgame/cg_text.cpp"
    "${SPDir}/cgame/cg_view.cpp"
    "${SPDir}/cgame/cg_weapons.cpp"
    "${SPDir}/cgame/FxPrimitives.cpp"
    "${SPDir}/cgame/FxScheduler.cpp"
    "${SPDir}/cgame/FxSystem.cpp"
    "${SPDir}/cgame/FxTemplate.cpp"
    "${SPDir}/cgame/FxUtil.cpp"
    "${SPDir}/cgame/FX_ATSTMain.cpp"
    "${SPDir}/cgame/FX_Blaster.cpp"
    "${SPDir}/cgame/FX_Bowcaster.cpp"
    "${SPDir}/cgame/FX_BryarPistol.cpp"
    "${SPDir}/cgame/FX_Concussion.cpp"
    "${SPDir}/cgame/FX_DEMP2.cpp"
    "${SPDir}/cgame/FX_Disruptor.cpp"
    "${SPDir}/cgame/FX_Emplaced.cpp"
    "${SPDir}/cgame/FX_Flechette.cpp"
    "${SPDir}/cgame/FX_HeavyRepeater.cpp"
    "${SPDir}/cgame/FX_NoghriShot.cpp"
    "${SPDir}/cgame/FX_RocketLauncher.cpp"
    "${SPDir}/cgame/FX_TuskenShot.cpp"
)

# Game-side UI
set(GameUIFiles
    "${SPDir}/ui/gameinfo.cpp"
)

# Game common files
set(GameCommonFiles
    "${SPDir}/qcommon/tri_coll_test.cpp"
    "${SPDir}/qcommon/q_shared.cpp"
    "${SPDir}/qcommon/q_shared.h"
    ${SharedCommonFiles}
)

set(GameCommonSafeFiles
    ${SharedCommonSafeFiles}
    "${SharedDir}/qcommon/safe/files.h"
    "${SharedDir}/qcommon/safe/files.cpp"
    "${SPDir}/qcommon/safe/memory.h"
)

# Game ICARUS files
set(GameIcarusFiles
    "${SPDir}/icarus/BlockStream.cpp"
    "${SPDir}/icarus/IcarusImplementation.cpp"
    "${SPDir}/icarus/Sequence.cpp"
    "${SPDir}/icarus/Sequencer.cpp"
    "${SPDir}/icarus/TaskManager.cpp"
)

# Game Ratl files
set(GameRatlFiles
    "${SPDir}/Ratl/ratl.cpp"
)

# Game Ravl files
set(GameRavlFiles
    "${SPDir}/Ravl/CBounds.cpp"
    "${SPDir}/Ravl/CVec.cpp"
)

# Game Rufl files
set(GameRuflFiles
    "${SPDir}/Rufl/hfile.cpp"
    "${SPDir}/Rufl/hstring.cpp"
)

# ============================================================================
# Bundled minizip (always used, modified to use Z_Malloc)
# ============================================================================
set(MinizipFiles
    "${OpenJKLibDir}/minizip/unzip.c"
    "${OpenJKLibDir}/minizip/ioapi.c"
)

# ============================================================================
# Combine ALL source files
# ============================================================================
set(ALL_SOURCES
    # Engine
    ${EngineClientFiles}
    ${EngineCommonFiles}
    ${EngineCommonSafeFiles}
    ${EngineServerFiles}
    ${EngineGameFiles}
    ${EngineMP3Files}
    ${EngineRendererFiles}
    ${EngineUIFiles}
    ${EngineGhoul2Files}
    ${EngineCGameFiles}
    ${EngineSysFiles}

    # Renderer (built into executable instead of shared lib)
    ${RendererG2Files}
    ${RendererMainFiles}
    ${RendererRdCommonFiles}
    # Note: RendererCommonFiles has duplicates with engine — we'll handle below

    # Game DLL files (built into executable instead of shared lib)
    ${GameDLLFiles}
    ${GameCGameFiles}
    ${GameUIFiles}
    ${GameIcarusFiles}
    ${GameRatlFiles}
    ${GameRavlFiles}
    ${GameRuflFiles}

    # Renderer common files (matcomp needed by renderer)
    "${SPDir}/qcommon/matcomp.cpp"

    # Game common files (tri_coll_test needed by game)
    "${SPDir}/qcommon/tri_coll_test.cpp"

    # Bundled minizip
    ${MinizipFiles}
)

# ============================================================================
# Create the executable
# ============================================================================
add_executable(openjk_wasm ${ALL_SOURCES})

# Mark header files so CMake doesn't try to compile them
foreach(_src ${ALL_SOURCES})
    get_filename_component(_ext "${_src}" EXT)
    if(_ext STREQUAL ".h")
        set_source_files_properties("${_src}" PROPERTIES HEADER_FILE_ONLY TRUE)
    endif()
endforeach()

# Set .c files to compile as C
set_source_files_properties(
    ${EngineMP3Files}
    "${SharedDir}/qcommon/q_color.c"
    "${SharedDir}/qcommon/q_math.c"
    "${SharedDir}/qcommon/q_string.c"
    "${OpenJKLibDir}/minizip/unzip.c"
    "${OpenJKLibDir}/minizip/ioapi.c"
    PROPERTIES LANGUAGE C
)

# ============================================================================
# Emscripten compile and link flags
# ============================================================================
target_compile_options(openjk_wasm PRIVATE
    -DUSE_OPENAL=1
    -Wno-write-strings
    -Wno-comment
    -Wno-invalid-offsetof
    -fsigned-char
    -O2
)

target_link_options(openjk_wasm PRIVATE
    -lopenal
    -sUSE_SDL=2
    -sUSE_ZLIB=1
    -sUSE_LIBPNG=1
    -sUSE_LIBJPEG=1
    -sALLOW_MEMORY_GROWTH=1
    -sINITIAL_MEMORY=67108864
    -sFORCE_FILESYSTEM=1
    -sWASM=1
    -sFULL_ES2=1
    -sSTACK_SIZE=8388608
    -sERROR_ON_UNDEFINED_SYMBOLS=0
    -O0 -g
    -sASSERTIONS=1
    # Preload game assets
    --preload-file "${ProjectRoot}/GameData/base@base"
    # Output HTML page
    --shell-file "${CMAKE_SOURCE_DIR}/shell.html"
)

target_link_libraries(openjk_wasm PRIVATE
    "${GL4ESDir}/lib/libGL.a"
    "-lopenal"
)

# Also set compile flags for libraries Emscripten provides
target_compile_options(openjk_wasm PRIVATE
    "SHELL:-sUSE_SDL=2"
    "SHELL:-sUSE_ZLIB=1"
    "SHELL:-sUSE_LIBPNG=1"
    "SHELL:-sUSE_LIBJPEG=1"
)

set_target_properties(openjk_wasm PROPERTIES
    SUFFIX ".html"
    OUTPUT_NAME "openjk"
)
