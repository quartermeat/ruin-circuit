# Ruin Circuit

Working title for a top-down sci-fi action RPG built with Go, Ebitengine, and WASM.

## Core fantasy

One adaptable combat chassis explores ruined megastructures, fights corrupted machines, gathers components, and builds a highly specialized combat identity. The character is one class; the build is the character.

The tone is Diablo-style action RPG progression in a sci-fi ruins setting, controlled with a MOBA-style layout rather than keyboard movement.

## Controls

- Right-click ground: move
- Right-click enemy: move toward and auto-attack
- Q/W/E/R: primary abilities
- 1–0: learned skills, items, or additional abilities
- Space: center the camera
- Tab: build and loot panel

## Build system

The player has six equipped slots, keeping the active loadout as readable as a MOBA item bar. There is no giant inventory-management game.

Each ability is assembled from modular components:

- Trigger
- Targeting behavior
- Effect
- Modifier
- Scaling behavior

Components are purchased with permanent tech currency. Dungeon runs provide materials, rare catalysts, and intact items used to craft finished equipment and abilities.

## Progression

The character begins by choosing an auto-attack chassis. Later level milestones unlock Q, W, E, and R. Each ability has a deep specialization path, but the choices are assembled from owned components rather than selected from a flat list of passive bonuses.

Primary stats emerge from the player’s chosen components and equipped items. The same ability can become a projectile build, a control build, a mobility build, or a scaling execution build depending on its assembled parts.

## Gameplay loop

```text
dungeon crawl → gather components and materials → return to hub
→ buy tech components → craft abilities/items → equip six slots
→ push into harder ruins
```

## Initial vertical slice

- One playable combat chassis
- One ruined facility
- Three enemy types
- One elite encounter
- One boss
- Auto-attack plus Q/W/E/R
- Six equipment slots
- Component crafting for at least one ability
- Click-to-move navigation

## Current prototype status

The first fidget slice is now scaffolded in Ebitengine:

- Right-click click-to-move movement
- Sci-fi ruin floor, broken walls, and three dormant machine targets
- Visible player combat chassis and movement marker
- Grid collision against ruin walls
- Breadth-first click-to-move pathfinding around obstacles
- Collision uses wall/tile overlap so thin walls and corner geometry agree with the visible map
- Q/W/E/R placeholder ability slots
- Tab toggles a build workbench placeholder
- WASM build script and browser entry point

The current prototype is intentionally a movement-and-presentation slice. Combat, enemy behavior, component purchases, crafting, the tech tree, and the six-slot loadout remain planned work.

## Versioning

Versions follow the project release rule:

- Patch (`v0.0.x`): every new build, including fixes and tuning.
- Feature (`v0.x.0`): a meaningful new feature; reset the patch number to zero.
- Shareable (`v1.0.0` or later): promoted when Jeremy feels the game is ready to share.

Every push is tagged with the matching version number.

## Run the WASM prototype

Build and serve the browser version with one command:

```powershell
.\scripts\start-local.ps1
```

Open `http://127.0.0.1:8080/`. Use `-Address "127.0.0.1:8090"` if the default port is occupied.

The launcher cache-busts the WASM URL from the version in `main.go`, and the same version is visible in the game HUD.

The build workbench now has an always-available RESPEC button. Respec is free, opens the workbench immediately, and returns the build to `CHOOSE THE PATH` until the first auto-attack chassis is selected, keeping experimentation central to the prototype.

The first selected path now has a playable effect: right-click an enemy to target it, move into range, and the auto-attack will fire automatically until that enemy is destroyed. Enemies show simple health bars, and attacking is locked until the path is chosen.

Enemies now create threat when targeted or when the chassis enters their proximity radius. Threatened enemies damage the player at close range, the HUD shows chassis health, and defeat triggers an immediate emergency reboot at full health for continued experimentation.

The workbench now offers two auto-attack paths: close-range and ranged. The ranged path can fire from a larger distance, but enemies still become threatened by the attack and can close in to deal damage.

The build model uses attributes as damage multipliers. The selected auto-attack establishes a baseline damage identity, while each Q/W/E/R skill choice will carry its own scaling attribute; choices can reinforce one attribute or create a hybrid build. Once an enemy is within the selected auto-attack's effective range, movement stops and the chassis continues attacking from that position.

Auto-attacks now have readable combat animation: ranged attacks launch a traveling energy bolt, while close-range attacks show a directional melee slash.

Enemy attacks are animated as red energy strikes traveling from a threatened enemy to the chassis, making incoming damage readable alongside the HP change.

Threat now creates pursuit: threatened enemies move toward the chassis while respecting ruin-wall collision, then attack once they close to range. This makes target selection and proximity create an actual combat commitment.

For playtesting, RESPEC also respawns the three starter enemies at their original positions with full health and cleared threat state, allowing repeated build experiments without refreshing the page.

When the workbench is closed, `Tab` has the same behavior as clicking RESPEC: it resets the build, respawns the starter enemies, and opens the workbench. When the workbench is open, `Tab` closes it without resetting, so the menu always has an exit.

The prototype setting is now a first MOBA-style lane instead of a dungeon. An allied tower and enemy tower frame the lane, allied combat drones automatically respawn at our tower and push toward the enemy tower, and the HUD tracks the active wave and enemy-tower health.

The lane now has a single-lane MOBA baseline: allied and enemy waves spawn in matching numbers on the same timer and fight to a natural stalemate. Towers now attack nearby opposing creeps with matching range, damage, and cooldown. The enemy AI hero pushes left toward the blue tower and can break the stalemate by damaging it. Creep attack cooldowns are symmetric during contact, so neither side wins without a hero. The purple lane portal sits off the creep lane and opens a pop-up dungeon instance overlay; `Esc` returns to the lane. The portal loop is intentionally a shell for the next dungeon-combat implementation.

New runs begin paused with the RESPEC/workbench screen open, requiring an auto-attack path before the lane can start. RESPEC pauses the lane whenever it is opened, and all hostile defenders now spawn in the lane rather than in off-lane positions.

The enemy hero is targetable like a creep: right-click it after choosing an auto-attack, and the chassis will path into range and attack it using the selected melee or ranged behavior. When the player chooses a path, the enemy hero also randomly chooses from those same available paths; its current choice is shown beside it and resets with RESPEC. The lane now contains only the mirrored creep waves and the enemy hero; the legacy named defender units have been removed.

The enemy hero now respawns automatically after defeat. The timer starts at five seconds and increases by one second per thirty seconds of match time, up to thirty seconds, with the remaining time shown in the HUD. RESPEC/Esc still resets the whole match timer.

The player now uses that exact same respawn rule: defeat starts the same match-time-scaled timer, the chassis disappears during the timer, and it returns at the blue-side start position with full health. Both timers reset through RESPEC/Esc.

Creeps now prioritize heroes inside a 100-pixel combat radius over opposing creeps. Enemy creep targeting also matches hero targeting: right-click a red creep to follow it, stop at attack range, and keep attacking until another movement command cancels the order.

The enemy hero now prioritizes nearby blue creeps before attacking the blue tower. Creep waves can refill a missing side up to six active units, allowing hero pressure to create an advantage without permanently stopping lane spawns.

All active creeps and heroes now display floating health bars above their heads, using blue or red team colors for quick combat readability.

The enemy hero now responds to threat: once a player attack hits it, the hero abandons tower pressure, pursues the player, and attacks at its own selected melee or ranged distance until that life ends. Respawning clears that threat state.

`Esc` now resets the entire run: towers, waves, heroes, enemies, build choices, and dungeon state all return to the initial paused workbench screen.

Right-clicking an enemy now creates a persistent attack order: the chassis follows the moving target, stops at the selected attack range, and continues auto-attacking. Right-clicking open ground cancels the attack order and returns to movement.
