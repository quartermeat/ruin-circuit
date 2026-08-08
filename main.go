package main

import (
	"fmt"
	"image/color"
	"math"

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
	version      = "v0.7.0"
	maxHealth    = 10
)

var ruinWalls = [][4]float64{{90, 90, 180, 18}, {90, 90, 18, 115}, {680, 80, 190, 18}, {850, 80, 18, 130}, {330, 420, 260, 18}, {330, 350, 18, 88}}

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

type Game struct {
	playerX, playerY   float64
	targetX, targetY   float64
	hasTarget          bool
	path               []Cell
	pathIndex          int
	enemies            []Enemy
	showBuild          bool
	autoAttackPicked   bool
	autoAttackRanged   bool
	attackTarget       int
	attackCooldown     int
	attackAnimation    int
	attackStartX       float64
	attackStartY       float64
	attackEndX         float64
	attackEndY         float64
	attackVisualRanged bool
	playerHealth       int
	message            string
}

func NewGame() *Game {
	return &Game{
		playerX: 496,
		playerY: 288,
		enemies: []Enemy{
			{x: 220, y: 180, name: "scavenger drone", health: 3, active: true},
			{x: 740, y: 170, name: "scrap hound", health: 3, active: true},
			{x: 700, y: 390, name: "vault sentinel", health: 3, active: true},
		},
		attackTarget: -1,
		playerHealth: maxHealth,
		message:      "Right-click to move. Open the workbench and choose the path.",
	}
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		if pointInRect(mouseX, mouseY, 760, 465, 168, 34) {
			g.respec()
		} else if g.showBuild && pointInRect(mouseX, mouseY, 270, 245, 220, 34) {
			g.autoAttackPicked = true
			g.autoAttackRanged = false
			g.message = "Path chosen: close-range auto-attack online. Experiment freely."
		} else if g.showBuild && pointInRect(mouseX, mouseY, 270, 285, 220, 34) {
			g.autoAttackPicked = true
			g.autoAttackRanged = true
			g.message = "Path chosen: ranged auto-attack online. Keep your distance."
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mouseX, mouseY := ebiten.CursorPosition()
		if enemyIndex, ok := g.enemyAt(float64(mouseX), float64(mouseY)); ok {
			if !g.autoAttackPicked {
				g.message = "Choose the path before engaging enemies."
				return nil
			}
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
	if g.attackCooldown > 0 {
		g.attackCooldown--
	}
	if g.attackAnimation > 0 {
		g.attackAnimation--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.showBuild = !g.showBuild
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
	// Broken walls and machine ruins.
	for _, r := range ruinWalls {
		ebitenutil.DrawRect(screen, r[0], r[1], r[2], r[3], color.RGBA{52, 66, 82, 255})
		ebitenutil.DrawRect(screen, r[0]+3, r[1]+3, r[2]-6, 3, color.RGBA{82, 105, 124, 255})
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
}

func (g *Game) respec() {
	g.autoAttackPicked = false
	g.autoAttackRanged = false
	g.attackTarget = -1
	g.hasTarget = false
	g.showBuild = true
	g.message = "Build reset. Choose the path when you are ready."
}

func (g *Game) enemyAt(x, y float64) (int, bool) {
	for index, enemy := range g.enemies {
		if enemy.active && math.Hypot(enemy.x-x, enemy.y-y) <= 26 {
			return index, true
		}
	}
	return -1, false
}

func (g *Game) updateAutoAttack() {
	if g.attackTarget < 0 || g.attackTarget >= len(g.enemies) || !g.enemies[g.attackTarget].active {
		g.attackTarget = -1
		return
	}
	enemy := &g.enemies[g.attackTarget]
	enemy.threat = true
	attackRange := 58.0
	if g.autoAttackRanged {
		attackRange = 190
	}
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
