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
file(MAKE_DIRECTORY "${CMAKE_BINARY_DIR}/shared/qcommon")
file(WRITE "${CMAKE_BINARY_DIR}/shared/qcommon/q_version.h"
    "#pragma once\n"
    "#define SOURCE_DATE \"Mar 2 2026\"\n"
    "#define GIT_HASH \"wasm-build\"\n"
    "#define GIT_TAG \"wasm-build\"\n"
)

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
# Include directories (Global common ones only)
# ============================================================================
include_directories(
    ${SharedDir}
    ${CMAKE_BINARY_DIR}/shared
    "${GSLIncludeDirectory}"
    "${OpenJKLibDir}"
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
    "${SPDir}/qcommon/game_version.h"
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
    #"${SPDir}/rd-vanilla/tr_subs.cpp"
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

target_include_directories(openjk_wasm PRIVATE
    ${SPDir}
    "${SPDir}/game"
    "${SPDir}/rd-common"
    "${SPDir}/rd-vanilla"
)

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
    -sUSE_SDL=2
    -sUSE_ZLIB=1
    -sUSE_LIBPNG=1
    -sUSE_LIBJPEG=1
)

set_target_properties(openjk_wasm PROPERTIES
    SUFFIX ".html"
    OUTPUT_NAME "openjk"
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/dist"
)
# ============================================================================
# MULTIPLAYER
# ============================================================================
set(MPDir "${ProjectRoot}/codemp")

# MP Engine files
set(MPEngineCommonFiles
    "${MPDir}/qcommon/cm_load.cpp"
    "${MPDir}/qcommon/cm_patch.cpp"
    "${MPDir}/qcommon/cm_polylib.cpp"
    "${MPDir}/qcommon/cm_test.cpp"
    "${MPDir}/qcommon/cm_trace.cpp"
    "${MPDir}/qcommon/cmd.cpp"
    "${MPDir}/qcommon/common.cpp"
    "${MPDir}/qcommon/cvar.cpp"
    "${MPDir}/qcommon/files.cpp"
    "${MPDir}/qcommon/GenericParser2.cpp"
    "${MPDir}/qcommon/huffman.cpp"
    "${MPDir}/qcommon/md4.cpp"
    "${MPDir}/qcommon/md5.cpp"
    "${MPDir}/qcommon/msg.cpp"
    "${MPDir}/qcommon/matcomp.cpp"
    "${MPDir}/qcommon/net_chan.cpp"
    "${MPDir}/qcommon/net_ip.cpp"
    "${MPDir}/qcommon/persistence.cpp"
    "${MPDir}/qcommon/q_shared.cpp"
    "${MPDir}/qcommon/RoffSystem.cpp"
    "${MPDir}/qcommon/stringed_ingame.cpp"
    "${MPDir}/qcommon/stringed_interface.cpp"
    "${MPDir}/qcommon/vm.cpp"
    "${MPDir}/qcommon/z_memman_pc.cpp"
    "${MPDir}/ghoul2/G2_gore.cpp"
    ${SharedCommonFiles}
)

set(MPEngineClientFiles
    "${MPDir}/client/cl_avi.cpp"
    "${MPDir}/client/cl_cgame.cpp"
    "${MPDir}/client/cl_cgameapi.cpp"
    "${MPDir}/client/cl_cin.cpp"
    "${MPDir}/client/cl_console.cpp"
    "${MPDir}/client/cl_input.cpp"
    "${MPDir}/client/cl_keys.cpp"
    "${MPDir}/client/cl_lan.cpp"
    "${MPDir}/client/cl_main.cpp"
    "${MPDir}/client/cl_net_chan.cpp"
    "${MPDir}/client/cl_parse.cpp"
    "${MPDir}/client/cl_scrn.cpp"
    "${MPDir}/client/cl_ui.cpp"
    "${MPDir}/client/cl_uiapi.cpp"
    "${MPDir}/client/FXExport.cpp"
    "${MPDir}/client/FxPrimitives.cpp"
    "${MPDir}/client/FxScheduler.cpp"
    "${MPDir}/client/FxSystem.cpp"
    "${MPDir}/client/FxTemplate.cpp"
    "${MPDir}/client/FxUtil.cpp"
    "${MPDir}/client/snd_ambient.cpp"
    "${MPDir}/client/snd_dma.cpp"
    "${MPDir}/client/snd_mem.cpp"
    "${MPDir}/client/snd_mix.cpp"
    "${MPDir}/client/snd_mp3.cpp"
    "${MPDir}/client/snd_music.cpp"
)

set(MPEngineServerFiles
    "${MPDir}/server/NPCNav/navigator.cpp"
    "${MPDir}/server/sv_bot.cpp"
    "${MPDir}/server/sv_ccmds.cpp"
    "${MPDir}/server/sv_challenge.cpp"
    "${MPDir}/server/sv_client.cpp"
    "${MPDir}/server/sv_game.cpp"
    "${MPDir}/server/sv_init.cpp"
    "${MPDir}/server/sv_main.cpp"
    "${MPDir}/server/sv_net_chan.cpp"
    "${MPDir}/server/sv_snapshot.cpp"
    "${MPDir}/server/sv_world.cpp"
    "${MPDir}/server/sv_gameapi.cpp"
)

set(MPIcarusFiles
    "${MPDir}/icarus/BlockStream.cpp"
    "${MPDir}/icarus/GameInterface.cpp"
    "${MPDir}/icarus/Instance.cpp"
    "${MPDir}/icarus/Interface.cpp"
    "${MPDir}/icarus/Memory.cpp"
    "${MPDir}/icarus/Q3_Interface.cpp"
    "${MPDir}/icarus/Q3_Registers.cpp"
    "${MPDir}/icarus/Sequence.cpp"
    "${MPDir}/icarus/Sequencer.cpp"
    "${MPDir}/icarus/TaskManager.cpp"
)

set(MPEngineSysFiles
    "${SharedDir}/sys/sys_main.cpp"
    "${SharedDir}/sys/sys_event.cpp"
    "${SharedDir}/sys/con_log.cpp"
    "${SharedDir}/sys/sys_unix.cpp"
    "${SharedDir}/sys/con_tty.cpp"
    "${SharedDir}/sys/snapvector.cpp"
    "${SharedDir}/sdl/sdl_sound.cpp"
    "${SharedDir}/sdl/sdl_input.cpp"
    "${SharedDir}/sdl/sdl_window.cpp"
)

# MP Redirected/Stub Renderer (Monolithic monolithic)
set(MPRendererFiles
    "${MPDir}/rd-vanilla/G2_API.cpp"
    "${MPDir}/rd-vanilla/G2_bolts.cpp"
    "${MPDir}/rd-vanilla/G2_bones.cpp"
    "${MPDir}/rd-vanilla/G2_misc.cpp"
    "${MPDir}/rd-vanilla/G2_surfaces.cpp"
    "${MPDir}/rd-vanilla/tr_arb.cpp"
    "${MPDir}/rd-vanilla/tr_backend.cpp"
    "${MPDir}/rd-vanilla/tr_bsp.cpp"
    "${MPDir}/rd-vanilla/tr_cmds.cpp"
    "${MPDir}/rd-vanilla/tr_curve.cpp"
    "${MPDir}/rd-vanilla/tr_decals.cpp"
    "${MPDir}/rd-vanilla/tr_ghoul2.cpp"
    "${MPDir}/rd-vanilla/tr_image.cpp"
    "${MPDir}/rd-vanilla/tr_init.cpp"
    "${MPDir}/rd-vanilla/tr_light.cpp"
    "${MPDir}/rd-vanilla/tr_main.cpp"
    "${MPDir}/rd-vanilla/tr_marks.cpp"
    "${MPDir}/rd-vanilla/tr_mesh.cpp"
    "${MPDir}/rd-vanilla/tr_model.cpp"
    "${MPDir}/rd-vanilla/tr_quicksprite.cpp"
    "${MPDir}/rd-vanilla/tr_scene.cpp"
    "${MPDir}/rd-vanilla/tr_shade.cpp"
    "${MPDir}/rd-vanilla/tr_shade_calc.cpp"
    "${MPDir}/rd-vanilla/tr_shader.cpp"
    "${MPDir}/rd-vanilla/tr_shadows.cpp"
    "${MPDir}/rd-vanilla/tr_skin.cpp"
    "${MPDir}/rd-vanilla/tr_sky.cpp"
    #"${MPDir}/rd-vanilla/tr_subs.cpp"
    "${MPDir}/rd-vanilla/tr_surface.cpp"
    "${MPDir}/rd-vanilla/tr_surfacesprites.cpp"
    "${MPDir}/rd-vanilla/tr_world.cpp"
    "${MPDir}/rd-vanilla/tr_WorldEffects.cpp"
    "${MPDir}/rd-common/tr_font.cpp"
    "${MPDir}/rd-common/tr_image_load.cpp"
    "${MPDir}/rd-common/tr_image_jpg.cpp"
    "${MPDir}/rd-common/tr_image_tga.cpp"
    "${MPDir}/rd-common/tr_image_png.cpp"
    "${MPDir}/rd-common/tr_noise.cpp"
)

# Static Libraries for MP Modules (to use -include headers)
add_library(mp_cgame STATIC
    "${MPDir}/cgame/cg_consolecmds.c"
    "${MPDir}/cgame/cg_cvar.c"
    "${MPDir}/cgame/cg_draw.c"
    "${MPDir}/cgame/cg_drawtools.c"
    "${MPDir}/cgame/cg_effects.c"
    "${MPDir}/cgame/cg_ents.c"
    "${MPDir}/cgame/cg_event.c"
    "${MPDir}/cgame/cg_info.c"
    "${MPDir}/cgame/cg_light.c"
    "${MPDir}/cgame/cg_localents.c"
    "${MPDir}/cgame/cg_main.c"
    "${MPDir}/cgame/cg_marks.c"
    "${MPDir}/cgame/cg_newDraw.c"
    "${MPDir}/cgame/cg_players.c"
    "${MPDir}/cgame/cg_playerstate.c"
    "${MPDir}/cgame/cg_predict.c"
    "${MPDir}/cgame/cg_saga.c"
    "${MPDir}/cgame/cg_scoreboard.c"
    "${MPDir}/cgame/cg_servercmds.c"
    "${MPDir}/cgame/cg_snapshot.c"
    "${MPDir}/cgame/cg_spawn.c"
    "${MPDir}/cgame/cg_syscalls.c"
    "${MPDir}/cgame/cg_turret.c"
    "${MPDir}/cgame/cg_view.c"
    "${MPDir}/ui/ui_shared.c"
    "${MPDir}/cgame/cg_weaponinit.c"
    "${MPDir}/cgame/cg_weapons.c"
    "${MPDir}/cgame/fx_blaster.c"
    "${MPDir}/cgame/fx_bowcaster.c"
    "${MPDir}/cgame/fx_bryarpistol.c"
    "${MPDir}/cgame/fx_demp2.c"
    "${MPDir}/cgame/fx_disruptor.c"
    "${MPDir}/cgame/fx_flechette.c"
    "${MPDir}/cgame/fx_force.c"
    "${MPDir}/cgame/fx_heavyrepeater.c"
    "${MPDir}/cgame/fx_rocketlauncher.c"
    "${MPDir}/game/bg_g2_utils.c"
    "${MPDir}/game/bg_misc.c"
    "${MPDir}/game/bg_panimate.c"
    "${MPDir}/game/bg_pmove.c"
    "${MPDir}/game/bg_saber.c"
    "${MPDir}/game/bg_saberLoad.c"
    "${MPDir}/game/bg_saga.c"
    "${MPDir}/game/bg_slidemove.c"
    "${MPDir}/game/bg_vehicleLoad.c"
    "${MPDir}/game/bg_weapons.c"
    "${MPDir}/game/FighterNPC.c"
    "${MPDir}/game/SpeederNPC.c"
    "${MPDir}/game/WalkerNPC.c"
    "${MPDir}/game/AnimalNPC.c"
)
target_include_directories(mp_cgame PRIVATE ${MPDir} ${SharedDir} ${CMAKE_BINARY_DIR}/shared)
target_compile_definitions(mp_cgame PRIVATE __EMSCRIPTEN__ MP_GAME _CGAME)
target_compile_options(mp_cgame PRIVATE -O2 -sUSE_SDL=2 "-include" "${MPDir}/cgame/cg_rename.h")

add_library(mp_game STATIC
    "${MPDir}/game/ai_main.c"
    "${MPDir}/game/ai_util.c"
    "${MPDir}/game/ai_wpnav.c"
    "${MPDir}/game/AnimalNPC.c"
    "${MPDir}/game/bg_g2_utils.c"
    "${MPDir}/game/bg_misc.c"
    "${MPDir}/game/bg_panimate.c"
    "${MPDir}/game/bg_pmove.c"
    "${MPDir}/game/bg_saber.c"
    "${MPDir}/game/bg_saberLoad.c"
    "${MPDir}/game/bg_saga.c"
    "${MPDir}/game/bg_slidemove.c"
    "${MPDir}/game/bg_vehicleLoad.c"
    "${MPDir}/game/bg_weapons.c"
    "${MPDir}/game/FighterNPC.c"
    "${MPDir}/game/g_active.c"
    "${MPDir}/game/g_bot.c"
    "${MPDir}/game/g_client.c"
    "${MPDir}/game/g_cmds.c"
    "${MPDir}/game/g_combat.c"
    "${MPDir}/game/g_cvar.c"
    "${MPDir}/game/g_exphysics.c"
    "${MPDir}/game/g_ICARUScb.c"
    "${MPDir}/game/g_items.c"
    "${MPDir}/game/g_log.c"
    "${MPDir}/game/g_main.c"
    "${MPDir}/game/g_mem.c"
    "${MPDir}/game/g_misc.c"
    "${MPDir}/game/g_missile.c"
    "${MPDir}/game/g_mover.c"
    "${MPDir}/game/g_nav.c"
    "${MPDir}/game/g_navnew.c"
    "${MPDir}/game/g_object.c"
    "${MPDir}/game/g_saga.c"
    "${MPDir}/game/g_session.c"
    "${MPDir}/game/g_spawn.c"
    "${MPDir}/game/g_svcmds.c"
    "${MPDir}/game/g_syscalls.c"
    "${MPDir}/game/g_target.c"
    "${MPDir}/game/g_team.c"
    "${MPDir}/game/g_timer.c"
    "${MPDir}/game/g_trigger.c"
    "${MPDir}/game/g_turret.c"
    "${MPDir}/game/g_turret_G2.c"
    "${MPDir}/game/g_utils.c"
    "${MPDir}/game/g_vehicles.c"
    "${MPDir}/game/g_vehicleTurret.c"
    "${MPDir}/game/g_weapon.c"
    "${MPDir}/game/NPC.c"
    "${MPDir}/game/NPC_AI_Atst.c"
    "${MPDir}/game/NPC_AI_Default.c"
    "${MPDir}/game/NPC_AI_Droid.c"
    "${MPDir}/game/NPC_AI_GalakMech.c"
    "${MPDir}/game/NPC_AI_Grenadier.c"
    "${MPDir}/game/NPC_AI_Howler.c"
    "${MPDir}/game/NPC_AI_ImperialProbe.c"
    "${MPDir}/game/NPC_AI_Interrogator.c"
    "${MPDir}/game/NPC_AI_Jedi.c"
    "${MPDir}/game/NPC_AI_Mark1.c"
    "${MPDir}/game/NPC_AI_Mark2.c"
    "${MPDir}/game/NPC_AI_MineMonster.c"
    "${MPDir}/game/NPC_AI_Rancor.c"
    "${MPDir}/game/NPC_AI_Remote.c"
    "${MPDir}/game/NPC_AI_Seeker.c"
    "${MPDir}/game/NPC_AI_Sentry.c"
    "${MPDir}/game/NPC_AI_Sniper.c"
    "${MPDir}/game/NPC_AI_Stormtrooper.c"
    "${MPDir}/game/NPC_AI_Utils.c"
    "${MPDir}/game/NPC_AI_Wampa.c"
    "${MPDir}/game/NPC_behavior.c"
    "${MPDir}/game/NPC_combat.c"
    "${MPDir}/game/NPC_goal.c"
    "${MPDir}/game/NPC_misc.c"
    "${MPDir}/game/NPC_move.c"
    "${MPDir}/game/NPC_reactions.c"
    "${MPDir}/game/NPC_senses.c"
    "${MPDir}/game/NPC_sounds.c"
    "${MPDir}/game/NPC_spawn.c"
    "${MPDir}/game/NPC_stats.c"
    "${MPDir}/game/NPC_utils.c"
    "${MPDir}/game/SpeederNPC.c"
    "${MPDir}/game/tri_coll_test.c"
    "${MPDir}/game/w_force.c"
    "${MPDir}/game/w_saber.c"
    "${MPDir}/game/WalkerNPC.c"
)
target_include_directories(mp_game PRIVATE ${MPDir} ${SharedDir} ${CMAKE_BINARY_DIR}/shared)
target_compile_definitions(mp_game PRIVATE __EMSCRIPTEN__ MP_GAME _GAME)
target_compile_options(mp_game PRIVATE -O2 -sUSE_SDL=2 "-include" "${MPDir}/game/g_rename.h")

add_library(mp_ui STATIC
    "${MPDir}/ui/ui_atoms.c"
    "${MPDir}/ui/ui_cvar.c"
    "${MPDir}/ui/ui_force.c"
    "${MPDir}/ui/ui_gameinfo.c"
    "${MPDir}/ui/ui_main.c"
    "${MPDir}/ui/ui_saber.c"
    "${MPDir}/ui/ui_shared.c"
    "${MPDir}/ui/ui_syscalls.c"
    "${MPDir}/game/bg_g2_utils.c"
    "${MPDir}/game/bg_misc.c"
    "${MPDir}/game/bg_panimate.c"
    "${MPDir}/game/bg_pmove.c"
    "${MPDir}/game/bg_saber.c"
    "${MPDir}/game/bg_saberLoad.c"
    "${MPDir}/game/bg_saga.c"
    "${MPDir}/game/bg_slidemove.c"
    "${MPDir}/game/bg_vehicleLoad.c"
    "${MPDir}/game/bg_weapons.c"
)
target_include_directories(mp_ui PRIVATE ${MPDir} ${SharedDir} ${CMAKE_BINARY_DIR}/shared)
target_compile_definitions(mp_ui PRIVATE __EMSCRIPTEN__ MP_GAME UI_BUILD)
target_compile_options(mp_ui PRIVATE -O2 -sUSE_SDL=2 "-include" "${MPDir}/ui/ui_rename.h")

# Botlib
add_library(mp_botlib STATIC
    "${MPDir}/botlib/be_aas_bspq3.cpp"
    "${MPDir}/botlib/be_aas_cluster.cpp"
    "${MPDir}/botlib/be_aas_debug.cpp"
    "${MPDir}/botlib/be_aas_entity.cpp"
    "${MPDir}/botlib/be_aas_file.cpp"
    "${MPDir}/botlib/be_aas_main.cpp"
    "${MPDir}/botlib/be_aas_move.cpp"
    "${MPDir}/botlib/be_aas_optimize.cpp"
    "${MPDir}/botlib/be_aas_reach.cpp"
    "${MPDir}/botlib/be_aas_route.cpp"
    "${MPDir}/botlib/be_aas_routealt.cpp"
    "${MPDir}/botlib/be_aas_sample.cpp"
    "${MPDir}/botlib/be_ai_char.cpp"
    "${MPDir}/botlib/be_ai_chat.cpp"
    "${MPDir}/botlib/be_ai_gen.cpp"
    "${MPDir}/botlib/be_ai_goal.cpp"
    "${MPDir}/botlib/be_ai_move.cpp"
    "${MPDir}/botlib/be_ai_weap.cpp"
    "${MPDir}/botlib/be_ai_weight.cpp"
    "${MPDir}/botlib/be_ea.cpp"
    "${MPDir}/botlib/be_interface.cpp"
    "${MPDir}/botlib/l_crc.cpp"
    "${MPDir}/botlib/l_libvar.cpp"
    "${MPDir}/botlib/l_log.cpp"
    "${MPDir}/botlib/l_memory.cpp"
    "${MPDir}/botlib/l_precomp.cpp"
    "${MPDir}/botlib/l_script.cpp"
    "${MPDir}/botlib/l_struct.cpp"
)
target_include_directories(mp_botlib PRIVATE ${MPDir} ${SharedDir} ${CMAKE_BINARY_DIR}/shared)
target_compile_definitions(mp_botlib PRIVATE __EMSCRIPTEN__ MP_GAME BOTLIB)
target_compile_options(mp_botlib PRIVATE -O2 -sUSE_SDL=2 "-include" "${MPDir}/botlib/bot_rename.h")

# MONOLITHIC MP EXECUTABLE
set(MP_ALL_SOURCES
    ${MPEngineCommonFiles}
    ${MPEngineClientFiles}
    ${MPEngineServerFiles}
    ${MPEngineSysFiles}
    ${MPRendererFiles}
    ${EngineMP3Files}
    ${MinizipFiles}
    ${MPIcarusFiles}
)

add_executable(openjk_mp_wasm ${MP_ALL_SOURCES})
target_include_directories(openjk_mp_wasm PRIVATE
    ${SharedDir}
    ${CMAKE_BINARY_DIR}/shared
    ${MPDir}
    "${MPDir}/rd-common"
    "${MPDir}/rd-vanilla"
    "${GSLIncludeDirectory}"
    "${OpenJKLibDir}"
    "${GL4ESDir}/include"
)
target_compile_definitions(openjk_mp_wasm PRIVATE
    _JK2EXE
    MP_GAME
    __EMSCRIPTEN__
    __linux__
    ARCH_STRING="wasm32"
    RENDERER          # needed for rd-vanilla sources
    DEFAULT_ES=2
)
target_compile_options(openjk_mp_wasm PRIVATE
    -DUSE_OPENAL=1
    -sUSE_SDL=2
    -Wno-write-strings
    -Wno-comment
    -Wno-invalid-offsetof
    -fsigned-char
    -O2
)

target_link_libraries(openjk_mp_wasm PRIVATE
    mp_cgame
    mp_game
    mp_ui
    mp_botlib
    "${GL4ESDir}/lib/libGL.a"
    "-lopenal"
)

target_link_options(openjk_mp_wasm PRIVATE
    -lopenal
    -sUSE_SDL=2
    -sUSE_ZLIB=1
    -sUSE_LIBPNG=1
    -sUSE_LIBJPEG=1
    -sALLOW_MEMORY_GROWTH=1
    -sINITIAL_MEMORY=134217728
    -sFORCE_FILESYSTEM=1
    -sWASM=1
    -sFULL_ES2=1
    -sSTACK_SIZE=8388608
    -sERROR_ON_UNDEFINED_SYMBOLS=0
    -O2
    # Preload game assets (base)
    --preload-file "${ProjectRoot}/GameData/base@base"
    # Output HTML page
    --shell-file "${CMAKE_SOURCE_DIR}/shell.html"
)

set_target_properties(openjk_mp_wasm PROPERTIES
    SUFFIX ".html"
    OUTPUT_NAME "openjk_mp"
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/dist"
)

# Force common.cpp to recompile on every build to update __DATE__ and __TIME__
add_custom_target(ForceUpdateVersion_MP ALL
    COMMAND ${CMAKE_COMMAND} -E touch "${MPDir}/qcommon/common.cpp"
)
add_dependencies(openjk_mp_wasm ForceUpdateVersion_MP)

# ============================================================================
# WebTransport Tunnel Server (Golang)
# ============================================================================
add_custom_target(wtunnel ALL
    COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=0 go build -o "${CMAKE_BINARY_DIR}/wtunnel"
    WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}/wtunnel"
    COMMENT "Building WebTransport Tunnel Server"
)

add_custom_target(serve
    COMMAND "${CMAKE_BINARY_DIR}/wtunnel" -dir "${CMAKE_BINARY_DIR}/dist"
    WORKING_DIRECTORY "${CMAKE_BINARY_DIR}"
    DEPENDS wtunnel openjk_mp_wasm openjk_wasm
    COMMENT "Running WebTransport Tunnel Server"
)
