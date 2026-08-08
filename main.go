package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

const (
	screenWidth    = 960
	screenHeight   = 540
	gridWidth      = 56
	gridHeight     = 28
	tileSize       = 16
	version        = "v0.17.2"
	maxHealth      = 10
	maxTowerHP     = 20
	towerRange     = 180.0
	creepHeroRange = 100.0
	towerHealRange = 120.0
	laneY          = 288.0
	blueHeroSpawnX = 150.0
	redHeroSpawnX  = 810.0
	dungeonExitX   = 820.0
)

var ruinWalls = [][4]float64{}

type Cell struct{ x, y int }

type Enemy struct {
	x, y       float64
	name       string
	health     int
	threat     bool
	attackCD   int
	attackAnim int
	targetX    float64
	targetY    float64
	active     bool
}

type Minion struct {
	x, y     float64
	health   int
	attackCD int
	active   bool
}

type Portal struct {
	x, y float64
	name string
}

type AIHero struct {
	x, y             float64
	health           int
	attackCD         int
	active           bool
	threat           bool
	autoAttackPicked bool
	autoAttackRanged bool
	inDungeon        bool
	dungeonX         float64
	dungeonY         float64
	dungeonCreeps    []DungeonCreep
}

type DungeonCreep struct {
	x, y     float64
	health   int
	attackCD int
	active   bool
}

type Game struct {
	playerX, playerY    float64
	targetX, targetY    float64
	hasTarget           bool
	path                []Cell
	pathIndex           int
	enemies             []Enemy
	minions             []Minion
	enemyMinions        []Minion
	minionSpawnTimer    int
	playerTowerHealth   int
	enemyTowerHealth    int
	playerTowerAttackCD int
	enemyTowerAttackCD  int
	won                 bool
	lost                bool
	aiHero              AIHero
	matchFrames         int
	aiHeroRespawnTimer  int
	portals             []Portal
	inDungeon           bool
	dungeonX            float64
	dungeonY            float64
	dungeonTargetX      float64
	dungeonTargetY      float64
	dungeonHasTarget    bool
	dungeonCreeps       []DungeonCreep
	showBuild           bool
	autoAttackPicked    bool
	autoAttackRanged    bool
	attackTarget        int
	attackTargetIsCreep bool
	attackHero          bool
	attackTower         bool
	portalTarget        bool
	attackCooldown      int
	attackAnimation     int
	attackStartX        float64
	attackStartY        float64
	attackEndX          float64
	attackEndY          float64
	attackVisualRanged  bool
	playerHealth        int
	playerActive        bool
	playerRespawnTimer  int
	message             string
}

func NewGame() *Game {
	return &Game{
		playerX: blueHeroSpawnX,
		playerY: laneY,
		enemies: []Enemy{},
		minions: []Minion{
			{x: 150, y: 270, health: 2, active: true},
			{x: 150, y: 288, health: 2, active: true},
			{x: 150, y: 306, health: 2, active: true},
		},
		enemyMinions: []Minion{
			{x: 810, y: 270, health: 2, active: true},
			{x: 810, y: 288, health: 2, active: true},
			{x: 810, y: 306, health: 2, active: true},
		},
		minionSpawnTimer:  240,
		playerTowerHealth: maxTowerHP,
		enemyTowerHealth:  maxTowerHP,
		aiHero:            AIHero{x: redHeroSpawnX, y: laneY, health: 10, active: true},
		portals:           []Portal{{x: 470, y: 400, name: "SUNKEN ARCHIVE"}},
		attackTarget:      -1,
		playerHealth:      maxHealth,
		playerActive:      true,
		showBuild:         true,
		message:           "Right-click to move. Open the workbench and choose the path.",
	}
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		*g = *NewGame()
		return nil
	}
	if g.won || g.lost {
		return nil
	}
	if g.inDungeon {
		g.updateDungeon()
		g.updateLaneBackground()
		g.checkMatchEnd()
		g.matchFrames++
		return nil
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		if pointInRect(mouseX, mouseY, 760, 465, 168, 34) {
			g.respec()
		} else if g.showBuild && pointInRect(mouseX, mouseY, 270, 245, 220, 34) {
			g.autoAttackPicked = true
			g.autoAttackRanged = false
			g.chooseAIHeroPath()
			g.message = "Path chosen: close-range auto-attack online. Experiment freely."
		} else if g.showBuild && pointInRect(mouseX, mouseY, 270, 285, 220, 34) {
			g.autoAttackPicked = true
			g.autoAttackRanged = true
			g.chooseAIHeroPath()
			g.message = "Path chosen: ranged auto-attack online. Keep your distance."
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if g.showBuild {
			g.showBuild = false
			g.message = "Workbench closed. Lane simulation resumed."
		} else {
			g.respec()
		}
		return nil
	}
	if g.showBuild {
		return nil
	}
	if !g.playerActive {
		g.updateMinionWave()
		g.updateTowerAttacks()
		g.updateAIHero()
		g.updatePlayerRespawn()
		g.matchFrames++
		return nil
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mouseX, mouseY := ebiten.CursorPosition()
		if portalIndex, ok := g.portalAt(float64(mouseX), float64(mouseY)); ok {
			g.attackHero = false
			g.attackTower = false
			g.attackTargetIsCreep = false
			g.attackTarget = -1
			g.portalTarget = true
			g.targetX, g.targetY = g.portals[portalIndex].x, g.portals[portalIndex].y
			g.path = findPath(worldToCellOrDefault(g.playerX, g.playerY), worldToCellOrDefault(g.targetX, g.targetY))
			g.pathIndex = 1
			g.hasTarget = len(g.path) > 1
			g.message = fmt.Sprintf("Moving to dungeon portal: %s", g.portals[portalIndex].name)
			return nil
		}
		if g.enemyTowerAt(float64(mouseX), float64(mouseY)) {
			if !g.autoAttackPicked {
				g.message = "Choose the path before attacking the enemy tower."
				return nil
			}
			g.attackHero = false
			g.attackTargetIsCreep = false
			g.attackTower = true
			g.attackTarget = -1
			g.targetX, g.targetY = 864, laneY
			g.message = "Targeting enemy tower — auto-attack engaged"
			return nil
		}
		if g.aiHeroAt(float64(mouseX), float64(mouseY)) {
			if !g.autoAttackPicked {
				g.message = "Choose the path before engaging enemies."
				return nil
			}
			g.attackHero = true
			g.attackTower = false
			g.portalTarget = false
			g.attackTarget = -1
			g.targetX, g.targetY = g.aiHero.x, g.aiHero.y
			target := worldToCellOrDefault(g.targetX, g.targetY)
			g.path = findPath(worldToCellOrDefault(g.playerX, g.playerY), target)
			g.pathIndex = 1
			g.hasTarget = len(g.path) > 1
			g.message = "Targeting enemy hero — auto-attack engaged"
			return nil
		}
		if creepIndex, ok := g.enemyMinionAt(float64(mouseX), float64(mouseY)); ok {
			if !g.autoAttackPicked {
				g.message = "Choose the path before engaging enemies."
				return nil
			}
			g.attackHero = false
			g.attackTower = false
			g.portalTarget = false
			g.attackTargetIsCreep = true
			g.attackTarget = creepIndex
			g.targetX, g.targetY = g.enemyMinions[creepIndex].x, g.enemyMinions[creepIndex].y
			g.message = "Targeting enemy creep — auto-attack engaged"
			return nil
		}
		if enemyIndex, ok := g.enemyAt(float64(mouseX), float64(mouseY)); ok {
			if !g.autoAttackPicked {
				g.message = "Choose the path before engaging enemies."
				return nil
			}
			g.attackHero = false
			g.attackTargetIsCreep = false
			g.attackTarget = enemyIndex
			g.targetX, g.targetY = g.enemies[enemyIndex].x, g.enemies[enemyIndex].y
			target := worldToCellOrDefault(g.targetX, g.targetY)
			if isBlocked(target) {
				target = nearestWalkableCell(target)
			}
			g.path = findPath(worldToCellOrDefault(g.playerX, g.playerY), target)
			g.pathIndex = 1
			g.hasTarget = len(g.path) > 1
			g.message = fmt.Sprintf("Targeting %s — auto-attack engaged", g.enemies[enemyIndex].name)
			return nil
		}
		g.attackHero = false
		g.attackTower = false
		g.portalTarget = false
		g.attackTargetIsCreep = false
		g.attackTarget = -1
		g.targetX, g.targetY = float64(mouseX), float64(mouseY)
		if target, ok := worldToCell(g.targetX, g.targetY); ok {
			if isBlocked(target) {
				target = nearestWalkableCell(target)
			}
			g.path = findPath(worldToCellOrDefault(g.playerX, g.playerY), target)
			g.pathIndex = 1
			g.hasTarget = len(g.path) > 1
			g.message = fmt.Sprintf("Path found: %d cells", len(g.path))
		} else {
			g.hasTarget = false
			g.message = "Click inside the ruin floor to move."
		}
	}
	g.refreshAttackPath()
	if g.hasTarget {
		if g.pathIndex >= len(g.path) {
			g.hasTarget = false
		} else {
			nextX, nextY := cellCenter(g.path[g.pathIndex])
			dx, dy := nextX-g.playerX, nextY-g.playerY
			distance := math.Hypot(dx, dy)
			if distance < 3 {
				g.playerX, g.playerY = nextX, nextY
				g.pathIndex++
				if g.pathIndex >= len(g.path) {
					g.hasTarget = false
				}
			} else {
				step := 3.8
				g.playerX += dx / distance * math.Min(step, distance)
				g.playerY += dy / distance * math.Min(step, distance)
			}
		}
	}
	if g.playerActive && g.portalTarget && g.portalAtPlayer() {
		g.enterDungeon()
		return nil
	}
	g.updateAutoAttack()
	g.updateEnemyAttacks()
	g.updateMinionWave()
	g.updateTowerAttacks()
	g.updateAIHero()
	g.checkMatchEnd()
	g.matchFrames++
	if g.attackCooldown > 0 {
		g.attackCooldown--
	}
	if g.attackAnimation > 0 {
		g.attackAnimation--
	}
	for key, label := range map[ebiten.Key]string{
		ebiten.KeyQ: "Q: primary ability slot",
		ebiten.KeyW: "W: primary ability slot",
		ebiten.KeyE: "E: primary ability slot",
		ebiten.KeyR: "R: primary ability slot",
	} {
		if inpututil.IsKeyJustPressed(key) {
			g.message = label + " — component system coming online."
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{7, 11, 18, 255})
	// Ruined facility floor.
	for y := 48; y < 500; y += 32 {
		for x := 32; x < 928; x += 32 {
			shade := uint8(22 + ((x/32+y/32)%3)*4)
			ebitenutil.DrawRect(screen, float64(x), float64(y), 30, 30, color.RGBA{shade, shade + 5, shade + 12, 255})
		}
	}
	// MOBA lane and towers.
	ebitenutil.DrawRect(screen, 32, 238, 896, 100, color.RGBA{28, 38, 48, 255})
	ebitenutil.DrawRect(screen, 32, 286, 896, 4, color.RGBA{63, 87, 96, 255})
	drawTower(screen, 96, 288, color.RGBA{76, 160, 190, 255}, color.RGBA{125, 220, 230, 255}, "ALLY TOWER", g.playerTowerHealth)
	drawTower(screen, 864, 288, color.RGBA{165, 70, 80, 255}, color.RGBA{245, 135, 125, 255}, "ENEMY TOWER", g.enemyTowerHealth)
	for _, portal := range g.portals {
		ebitenutil.DrawRect(screen, portal.x-14, portal.y-14, 28, 28, color.RGBA{105, 50, 155, 230})
		ebitenutil.DrawRect(screen, portal.x-8, portal.y-8, 16, 16, color.RGBA{210, 130, 245, 230})
		text.Draw(screen, "PORTAL", basicfont.Face7x13, int(portal.x)-21, int(portal.y)+28, color.RGBA{210, 160, 245, 255})
	}
	if g.aiHero.active {
		ebitenutil.DrawRect(screen, g.aiHero.x-10, g.aiHero.y-10, 20, 20, color.RGBA{190, 70, 80, 255})
		ebitenutil.DrawRect(screen, g.aiHero.x-5, g.aiHero.y-16, 10, 5, color.RGBA{250, 170, 140, 255})
		aiHeroLabel := "ENEMY HERO // CLOSE"
		if g.aiHero.autoAttackRanged {
			aiHeroLabel = "ENEMY HERO // RANGED"
		}
		text.Draw(screen, aiHeroLabel, basicfont.Face7x13, int(g.aiHero.x)-48, int(g.aiHero.y)-22, color.RGBA{245, 140, 145, 255})
		drawHealthBar(screen, g.aiHero.x, g.aiHero.y-28, g.aiHero.health, maxHealth, color.RGBA{235, 95, 105, 255})
		if g.aiHero.threat {
			ebitenutil.DrawRect(screen, g.aiHero.x-3, g.aiHero.y-34, 6, 2, color.RGBA{255, 115, 90, 255})
		}
	}
	for _, minion := range g.minions {
		if !minion.active {
			continue
		}
		ebitenutil.DrawRect(screen, minion.x-8, minion.y-8, 16, 16, color.RGBA{86, 178, 205, 255})
		ebitenutil.DrawRect(screen, minion.x-5, minion.y-12, 10, 3, color.RGBA{180, 235, 230, 255})
		drawHealthBar(screen, minion.x, minion.y-17, minion.health, 2, color.RGBA{90, 205, 225, 255})
	}
	for _, minion := range g.enemyMinions {
		if !minion.active {
			continue
		}
		ebitenutil.DrawRect(screen, minion.x-8, minion.y-8, 16, 16, color.RGBA{190, 75, 85, 255})
		ebitenutil.DrawRect(screen, minion.x-5, minion.y-12, 10, 3, color.RGBA{250, 170, 135, 255})
		drawHealthBar(screen, minion.x, minion.y-17, minion.health, 2, color.RGBA{235, 95, 105, 255})
	}
	for _, enemy := range g.enemies {
		if !enemy.active {
			continue
		}
		ebitenutil.DrawRect(screen, enemy.x-10, enemy.y-8, 20, 16, color.RGBA{145, 58, 65, 255})
		ebitenutil.DrawRect(screen, enemy.x-4, enemy.y-13, 8, 5, color.RGBA{227, 153, 74, 255})
		if enemy.threat {
			ebitenutil.DrawRect(screen, enemy.x-3, enemy.y-17, 6, 2, color.RGBA{255, 115, 90, 255})
		}
		ebitenutil.DrawRect(screen, enemy.x-14, enemy.y-22, 28, 3, color.RGBA{34, 20, 25, 255})
		ebitenutil.DrawRect(screen, enemy.x-14, enemy.y-22, float64(enemy.health)*9.3, 3, color.RGBA{220, 80, 85, 255})
	}
	g.drawEnemyAttackAnimations(screen)
	if g.hasTarget {
		ebitenutil.DrawRect(screen, g.targetX-5, g.targetY-1, 10, 2, color.RGBA{110, 219, 222, 220})
		ebitenutil.DrawRect(screen, g.targetX-1, g.targetY-5, 2, 10, color.RGBA{110, 219, 222, 220})
	}
	// Player combat chassis.
	if g.playerActive {
		ebitenutil.DrawRect(screen, g.playerX-11, g.playerY-11, 22, 22, color.RGBA{92, 196, 207, 255})
		ebitenutil.DrawRect(screen, g.playerX-5, g.playerY-17, 10, 7, color.RGBA{191, 235, 220, 255})
		ebitenutil.DrawRect(screen, g.playerX+9, g.playerY-3, 13, 5, color.RGBA{231, 177, 86, 255})
		g.drawAttackAnimation(screen)
		drawHealthBar(screen, g.playerX, g.playerY-23, g.playerHealth, maxHealth, color.RGBA{90, 205, 225, 255})
	}

	text.Draw(screen, "RUIN CIRCUIT // COMBAT CHASSIS ONLINE", basicfont.Face7x13, 32, 26, color.RGBA{190, 224, 224, 255})
	text.Draw(screen, version, basicfont.Face7x13, 880, 26, color.RGBA{150, 190, 195, 255})
	text.Draw(screen, "RIGHT-CLICK MOVE", basicfont.Face7x13, 760, 26, color.RGBA{150, 180, 190, 255})
	attackStatus := "AUTO-ATTACK: CHOOSE PATH"
	if g.autoAttackPicked {
		attackStatus = "AUTO-ATTACK: CLOSE-RANGE"
		if g.autoAttackRanged {
			attackStatus = "AUTO-ATTACK: RANGED"
		}
	}
	text.Draw(screen, attackStatus, basicfont.Face7x13, 32, 445, color.RGBA{190, 224, 224, 255})
	text.Draw(screen, fmt.Sprintf("CHASSIS HP: %02d/%02d", g.playerHealth, maxHealth), basicfont.Face7x13, 32, 460, color.RGBA{235, 150, 155, 255})
	text.Draw(screen, fmt.Sprintf("WAVE: %d v %d   TOWERS: %02d/%02d - %02d/%02d", g.activeMinions(), g.activeEnemyMinions(), g.playerTowerHealth, maxTowerHP, g.enemyTowerHealth, maxTowerHP), basicfont.Face7x13, 300, 445, color.RGBA{190, 224, 224, 255})
	if !g.aiHero.active {
		text.Draw(screen, fmt.Sprintf("ENEMY HERO RESPAWN: %ds", int(math.Ceil(float64(g.aiHeroRespawnTimer)/60))), basicfont.Face7x13, 610, 460, color.RGBA{245, 140, 145, 255})
	}
	if !g.playerActive {
		text.Draw(screen, fmt.Sprintf("PLAYER RESPAWN: %ds", int(math.Ceil(float64(g.playerRespawnTimer)/60))), basicfont.Face7x13, 610, 445, color.RGBA{125, 221, 225, 255})
	}
	text.Draw(screen, g.message, basicfont.Face7x13, 32, 520, color.RGBA{220, 200, 150, 255})
	for i, skill := range []string{"Q  COMPONENT", "W  COMPONENT", "E  COMPONENT", "R  COMPONENT"} {
		x := 32 + i*142
		ebitenutil.DrawRect(screen, float64(x), 465, 132, 34, color.RGBA{18, 28, 39, 245})
		text.Draw(screen, skill, basicfont.Face7x13, x+9, 486, color.RGBA{125, 221, 225, 255})
	}
	ebitenutil.DrawRect(screen, 760, 465, 168, 34, color.RGBA{56, 35, 46, 245})
	text.Draw(screen, "RESPEC  [ALWAYS]", basicfont.Face7x13, 772, 486, color.RGBA{245, 170, 180, 255})
	if g.showBuild {
		ebitenutil.DrawRect(screen, 220, 72, 520, 300, color.RGBA{8, 14, 24, 242})
		text.Draw(screen, "BUILD WORKBENCH", basicfont.Face7x13, 400, 105, color.RGBA{240, 216, 150, 255})
		text.Draw(screen, "Five component sockets will define every ability:", basicfont.Face7x13, 270, 145, color.White)
		text.Draw(screen, "TRIGGER   TARGETING   EFFECT   MODIFIER   SCALING", basicfont.Face7x13, 270, 180, color.RGBA{125, 221, 225, 255})
		if g.autoAttackPicked {
			pathName := "PATH: CLOSE-RANGE AUTO-ATTACK"
			if g.autoAttackRanged {
				pathName = "PATH: RANGED AUTO-ATTACK"
			}
			text.Draw(screen, pathName, basicfont.Face7x13, 270, 220, color.RGBA{150, 230, 175, 255})
		} else {
			text.Draw(screen, "PATH: CHOOSE THE PATH", basicfont.Face7x13, 270, 220, color.RGBA{245, 190, 125, 255})
		}
		ebitenutil.DrawRect(screen, 270, 245, 220, 34, color.RGBA{24, 62, 68, 255})
		text.Draw(screen, "CLOSE-RANGE ATTACK", basicfont.Face7x13, 292, 266, color.RGBA{240, 220, 150, 255})
		ebitenutil.DrawRect(screen, 270, 285, 220, 34, color.RGBA{24, 62, 68, 255})
		text.Draw(screen, "RANGED ATTACK", basicfont.Face7x13, 310, 306, color.RGBA{240, 220, 150, 255})
		text.Draw(screen, "TAB to close", basicfont.Face7x13, 430, 330, color.RGBA{240, 216, 150, 255})
	}
	if g.inDungeon {
		g.drawDungeon(screen)
	}
	if g.won || g.lost {
		ebitenutil.DrawRect(screen, 210, 150, 540, 190, color.RGBA{6, 12, 20, 238})
		ebitenutil.DrawRect(screen, 214, 154, 532, 182, color.RGBA{70, 160, 155, 180})
		result := "VICTORY"
		detail := "ENEMY TOWER DESTROYED"
		message := "THE LANE IS YOURS"
		if g.lost {
			result = "DEFEAT"
			detail = "ALLY TOWER DESTROYED"
			message = "THE LANE IS LOST"
		}
		text.Draw(screen, result, basicfont.Face7x13, 448, 205, color.RGBA{255, 230, 150, 255})
		text.Draw(screen, detail, basicfont.Face7x13, 366, 240, color.RGBA{190, 235, 225, 255})
		text.Draw(screen, message, basicfont.Face7x13, 398, 270, color.RGBA{150, 220, 210, 255})
		text.Draw(screen, "PRESS ESC TO START A NEW RUN", basicfont.Face7x13, 354, 310, color.RGBA{240, 216, 150, 255})
	}
}

func (g *Game) drawDungeon(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, 32, 48, 896, 448, color.RGBA{8, 11, 21, 255})
	for y := 80; y < 470; y += 32 {
		for x := 64; x < 896; x += 32 {
			shade := uint8(20 + ((x/32+y/32)%3)*5)
			ebitenutil.DrawRect(screen, float64(x), float64(y), 30, 30, color.RGBA{shade, shade + 2, shade + 10, 255})
		}
	}
	text.Draw(screen, "POP-UP DUNGEON // ESCAPE INSTANCE", basicfont.Face7x13, 330, 72, color.RGBA{210, 160, 245, 255})
	ebitenutil.DrawRect(screen, dungeonExitX-18, 240, 36, 96, color.RGBA{105, 50, 155, 230})
	ebitenutil.DrawRect(screen, dungeonExitX-10, 248, 20, 80, color.RGBA{210, 130, 245, 230})
	text.Draw(screen, "EXIT", basicfont.Face7x13, int(dungeonExitX)-15, 355, color.RGBA{210, 160, 245, 255})
	for _, creep := range g.dungeonCreeps {
		if !creep.active {
			continue
		}
		ebitenutil.DrawRect(screen, creep.x-8, creep.y-8, 16, 16, color.RGBA{205, 95, 75, 255})
		drawHealthBar(screen, creep.x, creep.y-14, creep.health, 2, color.RGBA{240, 105, 90, 255})
	}
	if g.playerActive {
		ebitenutil.DrawRect(screen, g.dungeonX-11, g.dungeonY-11, 22, 22, color.RGBA{92, 196, 207, 255})
		drawHealthBar(screen, g.dungeonX, g.dungeonY-22, g.playerHealth, maxHealth, color.RGBA{90, 205, 225, 255})
	}
	text.Draw(screen, "RIGHT-CLICK TO MOVE // REACH THE EXIT", basicfont.Face7x13, 300, 480, color.RGBA{240, 216, 150, 255})
	text.Draw(screen, "LANE CONTINUES IN BACKGROUND", basicfont.Face7x13, 350, 505, color.RGBA{150, 190, 195, 255})
}

func (g *Game) respec() {
	g.autoAttackPicked = false
	g.autoAttackRanged = false
	g.attackHero = false
	g.attackTower = false
	g.attackTargetIsCreep = false
	g.attackTarget = -1
	g.hasTarget = false
	g.showBuild = true
	g.message = "Build reset. Choose the path before entering the lane."
}

func (g *Game) chooseAIHeroPath() {
	g.aiHero.autoAttackPicked = true
	g.aiHero.autoAttackRanged = rand.Intn(2) == 1
}

func (g *Game) portalAtPlayer() bool {
	if len(g.portals) == 0 {
		return false
	}
	portal := g.portals[0]
	return math.Hypot(g.playerX-portal.x, g.playerY-portal.y) <= 22
}

func (g *Game) enterDungeon() {
	g.inDungeon = true
	g.portalTarget = false
	g.hasTarget = false
	g.dungeonX, g.dungeonY = 160, 288
	g.dungeonTargetX, g.dungeonTargetY = g.dungeonX, g.dungeonY
	g.dungeonCreeps = newDungeonCreeps(false)
	g.aiHero.inDungeon = true
	g.aiHero.dungeonX, g.aiHero.dungeonY = 800, 288
	g.aiHero.dungeonCreeps = newDungeonCreeps(true)
	g.message = "Entered the dungeon. Reach the exit portal."
}

func newDungeonCreeps(reverse bool) []DungeonCreep {
	positions := [][2]float64{{330, 190}, {430, 360}, {540, 220}, {630, 350}, {700, 180}, {760, 390}}
	creeps := make([]DungeonCreep, len(positions))
	for index, position := range positions {
		x, y := position[0], position[1]
		if reverse {
			x = 960 - x
		}
		creeps[index] = DungeonCreep{x: x, y: y, health: 2, active: true}
	}
	return creeps
}

func (g *Game) updateDungeon() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mouseX, mouseY := ebiten.CursorPosition()
		g.dungeonTargetX = math.Max(120, math.Min(dungeonExitX, float64(mouseX)))
		g.dungeonTargetY = math.Max(110, math.Min(430, float64(mouseY)))
		g.dungeonHasTarget = true
	}
	if g.dungeonHasTarget {
		dx, dy := g.dungeonTargetX-g.dungeonX, g.dungeonTargetY-g.dungeonY
		distance := math.Hypot(dx, dy)
		if distance < 4 {
			g.dungeonX, g.dungeonY = g.dungeonTargetX, g.dungeonTargetY
			g.dungeonHasTarget = false
		} else {
			step := math.Min(3.2, distance)
			g.dungeonX += dx / distance * step
			g.dungeonY += dy / distance * step
		}
	}
	hits := updateDungeonCreeps(g.dungeonCreeps, g.dungeonX, g.dungeonY)
	for index := 0; index < hits; index++ {
		g.playerHealth--
	}
	if g.playerHealth <= 0 {
		g.playerActive = false
		g.inDungeon = false
		g.playerRespawnTimer = g.enemyHeroRespawnDelay()
		g.message = "Chassis destroyed in the dungeon. Respawn timer started."
		return
	}
	if g.dungeonX >= dungeonExitX-20 {
		g.inDungeon = false
		g.playerX, g.playerY = blueHeroSpawnX, laneY
		g.message = "Dungeon exit reached. Returned to the blue tower."
	}
}

func updateDungeonCreeps(creeps []DungeonCreep, heroX, heroY float64) int {
	hits := 0
	for index := range creeps {
		creep := &creeps[index]
		if !creep.active {
			continue
		}
		if creep.attackCD > 0 {
			creep.attackCD--
		}
		dx, dy := heroX-creep.x, heroY-creep.y
		distance := math.Hypot(dx, dy)
		if distance > 38 {
			step := math.Min(0.9, distance-38)
			if distance > 0 {
				creep.x += dx / distance * step
				creep.y += dy / distance * step
			}
		} else if creep.attackCD == 0 {
			creep.attackCD = 30
			hits++
		}
	}
	return hits
}

func (g *Game) resetEnemies() {
	g.enemies = []Enemy{}
	g.enemyTowerHealth = maxTowerHP
	g.playerTowerHealth = maxTowerHP
	g.minions = []Minion{
		{x: 150, y: 270, health: 2, active: true},
		{x: 150, y: 288, health: 2, active: true},
		{x: 150, y: 306, health: 2, active: true},
	}
	g.enemyMinions = []Minion{
		{x: 810, y: 270, health: 2, active: true},
		{x: 810, y: 288, health: 2, active: true},
		{x: 810, y: 306, health: 2, active: true},
	}
	g.minionSpawnTimer = 240
	g.playerTowerAttackCD = 0
	g.enemyTowerAttackCD = 0
	g.aiHero = AIHero{x: redHeroSpawnX, y: laneY, health: 10, active: true}
	g.matchFrames = 0
	g.aiHeroRespawnTimer = 0
	g.playerX, g.playerY = blueHeroSpawnX, laneY
	g.playerHealth = maxHealth
	g.playerActive = true
	g.playerRespawnTimer = 0
}

func (g *Game) portalAt(x, y float64) (int, bool) {
	for index, portal := range g.portals {
		if math.Hypot(portal.x-x, portal.y-y) <= 24 {
			return index, true
		}
	}
	return -1, false
}

func (g *Game) updateAIHero() {
	if g.aiHero.inDungeon {
		g.updateAIHeroDungeon()
		return
	}
	if !g.aiHero.active {
		if g.aiHeroRespawnTimer > 0 {
			g.aiHeroRespawnTimer--
			return
		}
		g.aiHero.x, g.aiHero.y = redHeroSpawnX, laneY
		g.aiHero.health = 10
		g.aiHero.threat = false
		g.aiHero.active = true
		if g.inDungeon {
			g.aiHero.inDungeon = true
			g.aiHero.dungeonX, g.aiHero.dungeonY = 800, laneY
			g.aiHero.dungeonCreeps = newDungeonCreeps(true)
			g.message = "Enemy hero respawned into its dungeon layer."
		} else {
			g.aiHero.inDungeon = false
			g.message = "Enemy hero respawned at the red tower."
		}
		return
	}
	if g.aiHero.attackCD > 0 {
		g.aiHero.attackCD--
	}
	if g.aiHero.threat && g.playerActive {
		attackRange := 58.0
		if g.aiHero.autoAttackRanged {
			attackRange = 190
		}
		distance := math.Hypot(g.playerX-g.aiHero.x, g.playerY-g.aiHero.y)
		if distance > attackRange {
			step := math.Min(0.85, distance-attackRange)
			if distance > 0 {
				g.aiHero.x += (g.playerX - g.aiHero.x) / distance * step
				g.aiHero.y += (g.playerY - g.aiHero.y) / distance * step
			}
			return
		}
		if g.aiHero.attackCD == 0 {
			g.aiHero.attackCD = 30
			g.damagePlayerFromEnemyHero()
		}
		return
	}
	creepIndex, creepDistance := g.closestActiveAllyMinionToHero()
	if creepIndex >= 0 && creepDistance <= creepHeroRange {
		if g.aiHero.attackCD == 0 {
			g.aiHero.attackCD = 30
			g.minions[creepIndex].health--
			if g.minions[creepIndex].health <= 0 {
				g.minions[creepIndex].active = false
			}
		}
		return
	}
	if g.aiHero.x > 150 {
		g.aiHero.x -= 0.55
		return
	}
	if g.playerTowerHealth > 0 && g.aiHero.attackCD == 0 {
		g.aiHero.attackCD = 45
		g.playerTowerHealth--
	}
}

func (g *Game) updateAIHeroDungeon() {
	hits := updateDungeonCreeps(g.aiHero.dungeonCreeps, g.aiHero.dungeonX, g.aiHero.dungeonY)
	for index := 0; index < hits; index++ {
		g.aiHero.health--
	}
	if g.aiHero.health <= 0 {
		g.aiHero.active = false
		g.aiHero.inDungeon = false
		g.aiHeroRespawnTimer = g.enemyHeroRespawnDelay()
		g.message = "Enemy hero was defeated in the dungeon."
		return
	}
	if g.aiHero.dungeonX > 140 {
		g.aiHero.dungeonX -= 2.4
		return
	}
	g.aiHero.inDungeon = false
	g.aiHero.x, g.aiHero.y = redHeroSpawnX, laneY
}

func (g *Game) updatePlayerRespawn() {
	if g.playerRespawnTimer > 0 {
		g.playerRespawnTimer--
		return
	}
	if g.aiHero.inDungeon {
		g.inDungeon = true
		g.dungeonX, g.dungeonY = 160, laneY
		g.dungeonCreeps = newDungeonCreeps(false)
		g.message = "Player chassis respawned into the dungeon layer."
	} else {
		g.inDungeon = false
		g.playerX, g.playerY = blueHeroSpawnX, laneY
		g.message = "Player chassis respawned at the blue tower."
	}
	g.playerHealth = maxHealth
	g.playerActive = true
}

func (g *Game) updateLaneBackground() {
	g.updateMinionWave()
	g.updateTowerAttacks()
	g.updateAIHero()
}

func (g *Game) checkMatchEnd() {
	if g.won || g.lost {
		return
	}
	if g.enemyTowerHealth <= 0 {
		g.enemyTowerHealth = 0
		g.won = true
		g.message = "Enemy tower destroyed. Victory achieved."
	} else if g.playerTowerHealth <= 0 {
		g.playerTowerHealth = 0
		g.lost = true
		g.message = "Ally tower destroyed. Defeat."
	} else {
		return
	}
	g.attackTower = false
	g.attackHero = false
	g.attackTargetIsCreep = false
	g.portalTarget = false
	g.hasTarget = false
}

func (g *Game) damageAIHeroFromCreep() {
	g.aiHero.health--
	if g.aiHero.health > 0 {
		return
	}
	g.aiHero.active = false
	g.aiHeroRespawnTimer = g.enemyHeroRespawnDelay()
	g.attackHero = false
	g.hasTarget = false
	g.message = "Enemy hero defeated by allied creeps."
}

func (g *Game) damagePlayerFromCreep() {
	g.playerHealth--
	if g.playerHealth > 0 {
		return
	}
	g.playerActive = false
	g.playerRespawnTimer = g.enemyHeroRespawnDelay()
	g.attackHero = false
	g.attackTower = false
	g.attackTarget = -1
	g.attackTargetIsCreep = false
	g.hasTarget = false
	g.message = "Chassis destroyed by enemy creeps. Respawn timer started."
}

func (g *Game) damagePlayerFromEnemyHero() {
	g.playerHealth--
	if g.playerHealth > 0 {
		g.message = fmt.Sprintf("Enemy hero hit the chassis (%d/%d HP)", g.playerHealth, maxHealth)
		return
	}
	g.playerActive = false
	g.playerRespawnTimer = g.enemyHeroRespawnDelay()
	g.attackHero = false
	g.attackTarget = -1
	g.attackTargetIsCreep = false
	g.hasTarget = false
	g.message = "Chassis destroyed by enemy hero. Respawn timer started."
}

func (g *Game) updateMinionWave() {
	if g.minionSpawnTimer > 0 {
		g.minionSpawnTimer--
	} else if g.activeMinions() < 6 || g.activeEnemyMinions() < 6 {
		spawnY := 270 + float64((g.activeMinions()%3)*18)
		if g.activeMinions() < 6 {
			g.minions = append(g.minions, Minion{x: 150, y: spawnY, health: 2, active: true})
		}
		if g.activeEnemyMinions() < 6 {
			g.enemyMinions = append(g.enemyMinions, Minion{x: 810, y: spawnY, health: 2, active: true})
		}
		g.minionSpawnTimer = 240
	}
	for index := range g.minions {
		minion := &g.minions[index]
		if !minion.active {
			continue
		}
		if minion.attackCD > 0 {
			minion.attackCD--
		}
		if g.aiHero.active && math.Hypot(g.aiHero.x-minion.x, g.aiHero.y-minion.y) <= creepHeroRange {
			if minion.attackCD == 0 {
				minion.attackCD = 30
				g.damageAIHeroFromCreep()
			}
			continue
		}
		opponentIndex := g.closestEnemyMinion(index)
		if opponentIndex >= 0 && math.Abs(g.enemyMinions[opponentIndex].x-minion.x) <= 28 {
			opponent := &g.enemyMinions[opponentIndex]
			if minion.attackCD == 0 {
				minion.attackCD = 30
				opponent.health--
			}
			if opponent.attackCD == 0 {
				opponent.attackCD = 30
				minion.health--
			}
			if minion.health <= 0 {
				minion.active = false
			}
			if opponent.health <= 0 {
				opponent.active = false
			}
			continue
		}
		if g.enemyTowerHealth <= 0 {
			continue
		}
		if minion.x < 830 {
			minion.x += 1.1
		} else if minion.attackCD == 0 {
			minion.attackCD = 30
			g.enemyTowerHealth--
		}
	}
	for index := range g.enemyMinions {
		minion := &g.enemyMinions[index]
		if !minion.active {
			continue
		}
		if minion.attackCD > 0 {
			minion.attackCD--
		}
		if g.playerActive && math.Hypot(g.playerX-minion.x, g.playerY-minion.y) <= creepHeroRange {
			if minion.attackCD == 0 {
				minion.attackCD = 30
				g.damagePlayerFromCreep()
			}
			continue
		}
		if g.closestAllyMinion(index) >= 0 && math.Abs(g.minions[g.closestAllyMinion(index)].x-minion.x) <= 28 {
			continue
		}
		if g.playerTowerHealth <= 0 {
			continue
		}
		if minion.x > 130 {
			minion.x -= 1.1
		} else if minion.attackCD == 0 {
			minion.attackCD = 30
			g.playerTowerHealth--
		}
	}
	if g.enemyTowerHealth <= 0 {
		g.enemyTowerHealth = 0
		g.message = "Enemy tower destroyed. Lane victory achieved."
	}
	if g.playerTowerHealth <= 0 {
		g.playerTowerHealth = 0
		g.message = "Ally tower destroyed. The lane is lost."
	}
}

func (g *Game) updateTowerAttacks() {
	if g.playerTowerAttackCD > 0 {
		g.playerTowerAttackCD--
	}
	if g.enemyTowerAttackCD > 0 {
		g.enemyTowerAttackCD--
	}
	if g.playerTowerHealth > 0 && g.playerTowerAttackCD == 0 && attackNearestMinion(g.enemyMinions, 96) {
		g.playerTowerAttackCD = 30
	}
	if g.enemyTowerHealth > 0 && g.enemyTowerAttackCD == 0 && attackNearestMinion(g.minions, 864) {
		g.enemyTowerAttackCD = 30
	}
	if g.matchFrames%30 == 0 {
		if !g.inDungeon && g.playerActive && g.playerTowerHealth > 0 && g.playerHealth < maxHealth && math.Hypot(g.playerX-blueHeroSpawnX, g.playerY-laneY) <= towerHealRange {
			g.playerHealth++
		}
		if !g.aiHero.inDungeon && g.aiHero.active && g.enemyTowerHealth > 0 && g.aiHero.health < maxHealth && math.Hypot(g.aiHero.x-redHeroSpawnX, g.aiHero.y-laneY) <= towerHealRange {
			g.aiHero.health++
		}
	}
}

func attackNearestMinion(minions []Minion, towerX float64) bool {
	target := -1
	bestDistance := towerRange
	for index, minion := range minions {
		if !minion.active {
			continue
		}
		distance := math.Abs(minion.x - towerX)
		if distance <= bestDistance {
			target, bestDistance = index, distance
		}
	}
	if target < 0 {
		return false
	}
	minions[target].health--
	if minions[target].health <= 0 {
		minions[target].active = false
	}
	return true
}

func (g *Game) activeMinions() int {
	count := 0
	for _, minion := range g.minions {
		if minion.active {
			count++
		}
	}
	return count
}

func (g *Game) activeEnemyMinions() int {
	count := 0
	for _, minion := range g.enemyMinions {
		if minion.active {
			count++
		}
	}
	return count
}

func (g *Game) closestActiveAllyMinionToHero() (int, float64) {
	best, distance := -1, math.MaxFloat64
	for index, minion := range g.minions {
		if !minion.active {
			continue
		}
		current := math.Hypot(minion.x-g.aiHero.x, minion.y-g.aiHero.y)
		if current < distance {
			best, distance = index, current
		}
	}
	return best, distance
}

func (g *Game) closestEnemyMinion(index int) int {
	best, distance := -1, math.MaxFloat64
	for enemyIndex, enemy := range g.enemyMinions {
		if !enemy.active {
			continue
		}
		current := math.Abs(enemy.x - g.minions[index].x)
		if current < distance {
			best, distance = enemyIndex, current
		}
	}
	return best
}

func (g *Game) closestAllyMinion(index int) int {
	best, distance := -1, math.MaxFloat64
	for allyIndex, ally := range g.minions {
		if !ally.active {
			continue
		}
		current := math.Abs(ally.x - g.enemyMinions[index].x)
		if current < distance {
			best, distance = allyIndex, current
		}
	}
	return best
}

func drawTower(screen *ebiten.Image, x, y float64, body, core color.Color, label string, health int) {
	ebitenutil.DrawRect(screen, x-24, y-42, 48, 84, body)
	ebitenutil.DrawRect(screen, x-12, y-28, 24, 56, core)
	ebitenutil.DrawRect(screen, x-32, y-57, 64, 5, color.RGBA{30, 20, 28, 255})
	if health > 0 {
		ebitenutil.DrawRect(screen, x-32, y-57, float64(health)*3.2, 5, core)
	}
	text.Draw(screen, label, basicfont.Face7x13, int(x)-35, int(y)+62, body)
}

func drawHealthBar(screen *ebiten.Image, x, y float64, health, maximum int, fill color.Color) {
	if maximum <= 0 {
		return
	}
	if health < 0 {
		health = 0
	}
	if health > maximum {
		health = maximum
	}
	const width = 24.0
	ebitenutil.DrawRect(screen, x-width/2, y, width, 3, color.RGBA{25, 18, 26, 240})
	ebitenutil.DrawRect(screen, x-width/2, y, width*float64(health)/float64(maximum), 3, fill)
}

func (g *Game) enemyAt(x, y float64) (int, bool) {
	for index, enemy := range g.enemies {
		if enemy.active && math.Hypot(enemy.x-x, enemy.y-y) <= 26 {
			return index, true
		}
	}
	return -1, false
}

func (g *Game) aiHeroAt(x, y float64) bool {
	return g.aiHero.active && math.Hypot(g.aiHero.x-x, g.aiHero.y-y) <= 28
}

func (g *Game) enemyMinionAt(x, y float64) (int, bool) {
	for index, minion := range g.enemyMinions {
		if minion.active && math.Hypot(minion.x-x, minion.y-y) <= 22 {
			return index, true
		}
	}
	return -1, false
}

func (g *Game) enemyTowerAt(x, y float64) bool {
	return math.Abs(x-864) <= 32 && math.Abs(y-laneY) <= 50 && g.enemyTowerHealth > 0
}

func (g *Game) updateAutoAttack() {
	if g.attackTower {
		if g.enemyTowerHealth <= 0 {
			g.attackTower = false
			g.hasTarget = false
			return
		}
		attackRange := g.selectedAttackRange()
		distance := math.Hypot(864-g.playerX, laneY-g.playerY)
		if distance <= attackRange {
			g.hasTarget = false
		}
		if distance > attackRange || g.attackCooldown > 0 {
			return
		}
		g.enemyTowerHealth--
		g.attackCooldown = 30
		g.attackAnimation = 10
		g.attackStartX, g.attackStartY = g.playerX, g.playerY
		g.attackEndX, g.attackEndY = 864, laneY
		g.attackVisualRanged = g.autoAttackRanged
		if g.enemyTowerHealth <= 0 {
			g.enemyTowerHealth = 0
			g.attackTower = false
			g.hasTarget = false
			g.checkMatchEnd()
			return
		}
		g.message = fmt.Sprintf("Auto-attack hit enemy tower (%d health)", g.enemyTowerHealth)
		return
	}
	if g.attackHero {
		if !g.aiHero.active {
			g.attackHero = false
			g.hasTarget = false
			return
		}
		attackRange := g.selectedAttackRange()
		distance := math.Hypot(g.aiHero.x-g.playerX, g.aiHero.y-g.playerY)
		if distance <= attackRange {
			g.hasTarget = false
		}
		if distance > attackRange || g.attackCooldown > 0 {
			return
		}
		g.aiHero.health--
		g.aiHero.threat = true
		g.attackCooldown = 30
		g.attackAnimation = 10
		g.attackStartX, g.attackStartY = g.playerX, g.playerY
		g.attackEndX, g.attackEndY = g.aiHero.x, g.aiHero.y
		g.attackVisualRanged = g.autoAttackRanged
		if g.aiHero.health <= 0 {
			g.aiHero.active = false
			g.aiHeroRespawnTimer = g.enemyHeroRespawnDelay()
			g.attackHero = false
			g.hasTarget = false
			g.message = "Enemy hero defeated."
			return
		}
		g.message = fmt.Sprintf("Auto-attack hit enemy hero (%d health)", g.aiHero.health)
		return
	}
	if g.attackTargetIsCreep {
		if g.attackTarget < 0 || g.attackTarget >= len(g.enemyMinions) || !g.enemyMinions[g.attackTarget].active {
			g.attackTarget = -1
			g.attackTargetIsCreep = false
			g.hasTarget = false
			return
		}
		creep := &g.enemyMinions[g.attackTarget]
		attackRange := g.selectedAttackRange()
		if math.Hypot(creep.x-g.playerX, creep.y-g.playerY) <= attackRange {
			g.hasTarget = false
		}
		if math.Hypot(creep.x-g.playerX, creep.y-g.playerY) > attackRange || g.attackCooldown > 0 {
			return
		}
		creep.health--
		g.attackCooldown = 30
		g.attackAnimation = 10
		g.attackStartX, g.attackStartY = g.playerX, g.playerY
		g.attackEndX, g.attackEndY = creep.x, creep.y
		g.attackVisualRanged = g.autoAttackRanged
		if creep.health <= 0 {
			creep.active = false
			g.attackTarget = -1
			g.attackTargetIsCreep = false
			g.hasTarget = false
			g.message = "Enemy creep destroyed."
			return
		}
		g.message = fmt.Sprintf("Auto-attack hit enemy creep (%d health)", creep.health)
		return
	}
	if g.attackTarget < 0 || g.attackTarget >= len(g.enemies) || !g.enemies[g.attackTarget].active {
		g.attackTarget = -1
		return
	}
	enemy := &g.enemies[g.attackTarget]
	enemy.threat = true
	attackRange := g.selectedAttackRange()
	if math.Hypot(enemy.x-g.playerX, enemy.y-g.playerY) <= attackRange {
		g.hasTarget = false
	}
	if math.Hypot(enemy.x-g.playerX, enemy.y-g.playerY) > attackRange || g.attackCooldown > 0 {
		return
	}
	enemy.health--
	g.attackCooldown = 30
	g.attackAnimation = 10
	g.attackStartX, g.attackStartY = g.playerX, g.playerY
	g.attackEndX, g.attackEndY = enemy.x, enemy.y
	g.attackVisualRanged = g.autoAttackRanged
	if enemy.health <= 0 {
		enemy.active = false
		g.attackTarget = -1
		g.hasTarget = false
		g.message = fmt.Sprintf("%s destroyed. Choose another target.", enemy.name)
		return
	}
	g.message = fmt.Sprintf("Auto-attack hit %s (%d health)", enemy.name, enemy.health)
}

func (g *Game) enemyHeroRespawnDelay() int {
	matchSeconds := g.matchFrames / 60
	additionalSeconds := matchSeconds / 30
	if additionalSeconds > 25 {
		additionalSeconds = 25
	}
	return (5 + additionalSeconds) * 60
}

func (g *Game) selectedAttackRange() float64 {
	if g.autoAttackRanged {
		return 190
	}
	return 58
}

func (g *Game) refreshAttackPath() {
	var targetX, targetY float64
	var active bool
	if g.attackTower && g.enemyTowerHealth > 0 {
		targetX, targetY = 864, laneY
		active = true
	} else if g.attackHero && g.aiHero.active {
		targetX, targetY = g.aiHero.x, g.aiHero.y
		active = true
	} else if g.attackTargetIsCreep && g.attackTarget >= 0 && g.attackTarget < len(g.enemyMinions) && g.enemyMinions[g.attackTarget].active {
		targetX, targetY = g.enemyMinions[g.attackTarget].x, g.enemyMinions[g.attackTarget].y
		active = true
	} else if g.attackTarget >= 0 && g.attackTarget < len(g.enemies) && g.enemies[g.attackTarget].active {
		targetX, targetY = g.enemies[g.attackTarget].x, g.enemies[g.attackTarget].y
		active = true
	}
	if !active {
		return
	}
	g.targetX, g.targetY = targetX, targetY
	if math.Hypot(targetX-g.playerX, targetY-g.playerY) <= g.selectedAttackRange() {
		g.hasTarget = false
		return
	}
	target := worldToCellOrDefault(targetX, targetY)
	g.path = findPath(worldToCellOrDefault(g.playerX, g.playerY), target)
	g.pathIndex = 1
	g.hasTarget = len(g.path) > 1
}

func (g *Game) drawAttackAnimation(screen *ebiten.Image) {
	if g.attackAnimation <= 0 {
		return
	}
	progress := 1 - float64(g.attackAnimation)/10
	if g.attackVisualRanged {
		x := g.attackStartX + (g.attackEndX-g.attackStartX)*progress
		y := g.attackStartY + (g.attackEndY-g.attackStartY)*progress
		ebitenutil.DrawLine(screen, g.attackStartX, g.attackStartY, x, y, color.RGBA{110, 219, 222, 120})
		ebitenutil.DrawRect(screen, x-5, y-2, 10, 4, color.RGBA{175, 245, 236, 255})
		return
	}
	dx, dy := g.attackEndX-g.attackStartX, g.attackEndY-g.attackStartY
	distance := math.Hypot(dx, dy)
	if distance == 0 {
		return
	}
	perpendicularX, perpendicularY := -dy/distance*12, dx/distance*12
	centerX := g.attackStartX + dx/distance*30
	centerY := g.attackStartY + dy/distance*30
	slashColor := color.RGBA{245, 220, 150, 230}
	ebitenutil.DrawLine(screen, g.attackStartX, g.attackStartY, centerX+perpendicularX, centerY+perpendicularY, slashColor)
	ebitenutil.DrawLine(screen, centerX+perpendicularX, centerY+perpendicularY, centerX-perpendicularX, centerY-perpendicularY, slashColor)
}

func (g *Game) updateEnemyAttacks() {
	for index := range g.enemies {
		enemy := &g.enemies[index]
		if enemy.attackAnim > 0 {
			enemy.attackAnim--
		}
		if !enemy.active {
			continue
		}
		if enemy.attackCD > 0 {
			enemy.attackCD--
		}
		distance := math.Hypot(enemy.x-g.playerX, enemy.y-g.playerY)
		if g.attackTarget == index || distance <= 135 {
			enemy.threat = true
		}
		if enemy.threat && distance > 58 {
			g.moveEnemyTowardPlayer(enemy)
			distance = math.Hypot(enemy.x-g.playerX, enemy.y-g.playerY)
		}
		if !enemy.threat || distance > 58 || enemy.attackCD > 0 {
			continue
		}
		enemy.attackCD = 45
		enemy.attackAnim = 9
		enemy.targetX, enemy.targetY = g.playerX, g.playerY
		g.playerHealth--
		g.message = fmt.Sprintf("%s hit the chassis (%d/%d HP)", enemy.name, g.playerHealth, maxHealth)
		if g.playerHealth <= 0 {
			g.playerActive = false
			g.playerRespawnTimer = g.enemyHeroRespawnDelay()
			g.hasTarget = false
			g.attackHero = false
			g.attackTarget = -1
			g.message = "Chassis destroyed. Respawn timer started."
		}
	}
}

func (g *Game) moveEnemyTowardPlayer(enemy *Enemy) {
	dx, dy := g.playerX-enemy.x, g.playerY-enemy.y
	distance := math.Hypot(dx, dy)
	if distance == 0 {
		return
	}
	step := math.Min(1.25, distance-58)
	if step <= 0 {
		return
	}
	if canOccupy(enemy.x+dx/distance*step, enemy.y, 20, 16) {
		enemy.x += dx / distance * step
	}
	if canOccupy(enemy.x, enemy.y+dy/distance*step, 20, 16) {
		enemy.y += dy / distance * step
	}
}

func (g *Game) drawEnemyAttackAnimations(screen *ebiten.Image) {
	for _, enemy := range g.enemies {
		if enemy.attackAnim <= 0 {
			continue
		}
		progress := 1 - float64(enemy.attackAnim)/9
		x := enemy.x + (enemy.targetX-enemy.x)*progress
		y := enemy.y + (enemy.targetY-enemy.y)*progress
		ebitenutil.DrawLine(screen, enemy.x, enemy.y, x, y, color.RGBA{240, 80, 95, 150})
		ebitenutil.DrawRect(screen, x-5, y-2, 10, 4, color.RGBA{255, 130, 110, 255})
	}
}

func pointInRect(x, y, left, top, width, height int) bool {
	return x >= left && x < left+width && y >= top && y < top+height
}

func canOccupy(centerX, centerY float64, width, height float64) bool {
	left, top := centerX-width/2, centerY-height/2
	right, bottom := centerX+width/2, centerY+height/2
	if left < 32 || top < 48 || right > 928 || bottom > 496 {
		return false
	}
	for _, wall := range ruinWalls {
		wallLeft, wallTop := wall[0]-2, wall[1]-2
		wallRight, wallBottom := wall[0]+wall[2]+2, wall[1]+wall[3]+2
		if left < wallRight && right > wallLeft && top < wallBottom && bottom > wallTop {
			return false
		}
	}
	return true
}

func worldToCell(x, y float64) (Cell, bool) {
	cell := Cell{x: int((x - 32) / tileSize), y: int((y - 48) / tileSize)}
	return cell, cell.x >= 0 && cell.x < gridWidth && cell.y >= 0 && cell.y < gridHeight
}

func worldToCellOrDefault(x, y float64) Cell {
	cell, ok := worldToCell(x, y)
	if !ok {
		return Cell{x: gridWidth / 2, y: gridHeight / 2}
	}
	return cell
}

func cellCenter(cell Cell) (float64, float64) {
	return float64(32 + cell.x*tileSize + tileSize/2), float64(48 + cell.y*tileSize + tileSize/2)
}

func isBlocked(cell Cell) bool {
	if cell.x < 0 || cell.x >= gridWidth || cell.y < 0 || cell.y >= gridHeight {
		return true
	}
	cellLeft := float64(32 + cell.x*tileSize)
	cellTop := float64(48 + cell.y*tileSize)
	cellRight := cellLeft + tileSize
	cellBottom := cellTop + tileSize
	for _, wall := range ruinWalls {
		wallLeft, wallTop := wall[0]-2, wall[1]-2
		wallRight, wallBottom := wall[0]+wall[2]+2, wall[1]+wall[3]+2
		if cellLeft < wallRight && cellRight > wallLeft && cellTop < wallBottom && cellBottom > wallTop {
			return true
		}
	}
	return false
}

func nearestWalkableCell(target Cell) Cell {
	best := target
	bestDistance := 9999
	for y := 0; y < gridHeight; y++ {
		for x := 0; x < gridWidth; x++ {
			candidate := Cell{x: x, y: y}
			if isBlocked(candidate) {
				continue
			}
			distance := absInt(candidate.x-target.x) + absInt(candidate.y-target.y)
			if distance < bestDistance {
				best, bestDistance = candidate, distance
			}
		}
	}
	return best
}

func findPath(start, target Cell) []Cell {
	if isBlocked(start) || isBlocked(target) {
		return nil
	}
	visited := [gridHeight][gridWidth]bool{}
	previous := [gridHeight][gridWidth]Cell{}
	queue := []Cell{start}
	visited[start.y][start.x] = true
	previous[start.y][start.x] = start
	directions := [...]Cell{{x: 1}, {x: -1}, {y: 1}, {y: -1}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == target {
			break
		}
		for _, direction := range directions {
			next := Cell{x: current.x + direction.x, y: current.y + direction.y}
			if isBlocked(next) || visited[next.y][next.x] {
				continue
			}
			visited[next.y][next.x] = true
			previous[next.y][next.x] = current
			queue = append(queue, next)
		}
	}
	if !visited[target.y][target.x] {
		return nil
	}
	path := []Cell{}
	for current := target; ; current = previous[current.y][current.x] {
		path = append(path, current)
		if current == start {
			break
		}
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (g *Game) Layout(_, _ int) (int, int) { return screenWidth, screenHeight }

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Ruin Circuit")
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}
