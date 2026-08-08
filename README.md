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
