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
	screenWidth  = 960
	screenHeight = 540
	gridWidth    = 28
	gridHeight   = 14
	tileSize     = 32
	version      = "v0.13.0"
	maxHealth    = 10
	maxTowerHP   = 20
	towerRange   = 180.0
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
	autoAttackPicked bool
	autoAttackRanged bool
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
	aiHero              AIHero
	portals             []Portal
	inDungeon           bool
	showBuild           bool
	autoAttackPicked    bool
	autoAttackRanged    bool
	attackTarget        int
	attackHero          bool
	attackCooldown      int
	attackAnimation     int
	attackStartX        float64
	attackStartY        float64
	attackEndX          float64
	attackEndY          float64
	attackVisualRanged  bool
	playerHealth        int
	message             string
}

func NewGame() *Game {
	return &Game{
		playerX: 496,
		playerY: 288,
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
		aiHero:            AIHero{x: 720, y: 288, health: 10, active: true},
		portals:           []Portal{{x: 470, y: 400, name: "SUNKEN ARCHIVE"}},
		attackTarget:      -1,
		playerHealth:      maxHealth,
		showBuild:         true,
		message:           "Right-click to move. Open the workbench and choose the path.",
	}
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		*g = *NewGame()
		return nil
	}
	if g.inDungeon {
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
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mouseX, mouseY := ebiten.CursorPosition()
		if portalIndex, ok := g.portalAt(float64(mouseX), float64(mouseY)); ok {
			g.inDungeon = true
			g.hasTarget = false
			g.message = fmt.Sprintf("Entering pop-up dungeon: %s", g.portals[portalIndex].name)
			return nil
		}
		if g.aiHeroAt(float64(mouseX), float64(mouseY)) {
			if !g.autoAttackPicked {
				g.message = "Choose the path before engaging enemies."
				return nil
			}
			g.attackHero = true
			g.attackTarget = -1
			g.targetX, g.targetY = g.aiHero.x, g.aiHero.y
			target := worldToCellOrDefault(g.targetX, g.targetY)
			g.path = findPath(worldToCellOrDefault(g.playerX, g.playerY), target)
			g.pathIndex = 1
			g.hasTarget = len(g.path) > 1
			g.message = "Targeting enemy hero — auto-attack engaged"
			return nil
		}
		if enemyIndex, ok := g.enemyAt(float64(mouseX), float64(mouseY)); ok {
			if !g.autoAttackPicked {
				g.message = "Choose the path before engaging enemies."
				return nil
			}
			g.attackHero = false
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
	g.updateAutoAttack()
	g.updateEnemyAttacks()
	g.updateMinionWave()
	g.updateTowerAttacks()
	g.updateAIHero()
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
	}
	for _, minion := range g.minions {
		if !minion.active {
			continue
		}
		ebitenutil.DrawRect(screen, minion.x-8, minion.y-8, 16, 16, color.RGBA{86, 178, 205, 255})
		ebitenutil.DrawRect(screen, minion.x-5, minion.y-12, 10, 3, color.RGBA{180, 235, 230, 255})
	}
	for _, minion := range g.enemyMinions {
		if !minion.active {
			continue
		}
		ebitenutil.DrawRect(screen, minion.x-8, minion.y-8, 16, 16, color.RGBA{190, 75, 85, 255})
		ebitenutil.DrawRect(screen, minion.x-5, minion.y-12, 10, 3, color.RGBA{250, 170, 135, 255})
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
	ebitenutil.DrawRect(screen, g.playerX-11, g.playerY-11, 22, 22, color.RGBA{92, 196, 207, 255})
	ebitenutil.DrawRect(screen, g.playerX-5, g.playerY-17, 10, 7, color.RGBA{191, 235, 220, 255})
	ebitenutil.DrawRect(screen, g.playerX+9, g.playerY-3, 13, 5, color.RGBA{231, 177, 86, 255})
	g.drawAttackAnimation(screen)

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
		ebitenutil.DrawRect(screen, 170, 92, 620, 330, color.RGBA{7, 10, 19, 248})
		text.Draw(screen, "POP-UP DUNGEON // INSTANCE ACTIVE", basicfont.Face7x13, 330, 130, color.RGBA{210, 160, 245, 255})
		text.Draw(screen, "The lane portal has opened a temporary combat space.", basicfont.Face7x13, 260, 170, color.White)
		text.Draw(screen, "DUNGEON RULES COMING NEXT", basicfont.Face7x13, 355, 235, color.RGBA{240, 216, 150, 255})
		text.Draw(screen, "ESC  RETURN TO LANE", basicfont.Face7x13, 390, 360, color.RGBA{210, 160, 245, 255})
	}
}

func (g *Game) respec() {
	g.autoAttackPicked = false
	g.autoAttackRanged = false
	g.attackHero = false
	g.attackTarget = -1
	g.hasTarget = false
	g.resetEnemies()
	g.showBuild = true
	g.message = "Build reset. Enemies respawned. Choose the path before entering the lane."
}

func (g *Game) chooseAIHeroPath() {
	g.aiHero.autoAttackPicked = true
	g.aiHero.autoAttackRanged = rand.Intn(2) == 1
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
	g.aiHero = AIHero{x: 720, y: 288, health: 10, active: true}
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
	if !g.aiHero.active {
		return
	}
	if g.aiHero.attackCD > 0 {
		g.aiHero.attackCD--
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

func (g *Game) updateMinionWave() {
	if g.minionSpawnTimer > 0 {
		g.minionSpawnTimer--
	} else if g.activeMinions() < 6 && g.activeEnemyMinions() < 6 {
		spawnY := 270 + float64((g.activeMinions()%3)*18)
		g.minions = append(g.minions, Minion{x: 150, y: spawnY, health: 2, active: true})
		g.enemyMinions = append(g.enemyMinions, Minion{x: 810, y: spawnY, health: 2, active: true})
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

func (g *Game) updateAutoAttack() {
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
		g.attackCooldown = 30
		g.attackAnimation = 10
		g.attackStartX, g.attackStartY = g.playerX, g.playerY
		g.attackEndX, g.attackEndY = g.aiHero.x, g.aiHero.y
		g.attackVisualRanged = g.autoAttackRanged
		if g.aiHero.health <= 0 {
			g.aiHero.active = false
			g.attackHero = false
			g.hasTarget = false
			g.message = "Enemy hero defeated."
			return
		}
		g.message = fmt.Sprintf("Auto-attack hit enemy hero (%d health)", g.aiHero.health)
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

func (g *Game) selectedAttackRange() float64 {
	if g.autoAttackRanged {
		return 190
	}
	return 58
}

func (g *Game) refreshAttackPath() {
	var targetX, targetY float64
	var active bool
	if g.attackHero && g.aiHero.active {
		targetX, targetY = g.aiHero.x, g.aiHero.y
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
			g.playerHealth = maxHealth
			g.playerX, g.playerY = 496, 288
			g.hasTarget = false
			g.attackTarget = -1
			g.message = "Chassis disabled. Emergency reboot complete."
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
