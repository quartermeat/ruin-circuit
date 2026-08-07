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
