package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	stddraw "image/draw"
)

const (
	harnessTickRate  = 60
	harnessFPS       = 15
	harnessFrameSkip = harnessTickRate / harnessFPS
	harnessWidth     = 720
	harnessHeight    = 1280
)

type balanceEvent struct {
	Frame        int    `json:"frame"`
	Seconds      int    `json:"seconds"`
	PlayerHP     int    `json:"player_hp"`
	EnemyHeroHP  int    `json:"enemy_hero_hp"`
	BlueTowerHP  int    `json:"blue_tower_hp"`
	RedTowerHP   int    `json:"red_tower_hp"`
	PlayerDeaths int    `json:"player_deaths"`
	EnemyDeaths  int    `json:"enemy_deaths"`
	Outcome      string `json:"outcome"`
	Message      string `json:"message"`
}

type balanceSummary struct {
	Version      string `json:"version"`
	Outcome      string `json:"outcome"`
	DurationSec  int    `json:"duration_seconds"`
	PlayerDeaths int    `json:"player_deaths"`
	EnemyDeaths  int    `json:"enemy_deaths"`
	BlueTowerHP  int    `json:"blue_tower_hp"`
	RedTowerHP   int    `json:"red_tower_hp"`
}

func runBalanceHarness() error {
	outputRoot := os.Getenv("BALANCE_OUTPUT")
	if outputRoot == "" {
		outputRoot = filepath.Join("artifacts", "balance-run")
	}
	seconds := 180
	if value := os.Getenv("BALANCE_SECONDS"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("invalid BALANCE_SECONDS %q", value)
		}
		seconds = parsed
	}
	framesDir := filepath.Join(outputRoot, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		return err
	}
	logFile, err := os.Create(filepath.Join(outputRoot, "events.jsonl"))
	if err != nil {
		return err
	}
	defer logFile.Close()
	logWriter := bufio.NewWriter(logFile)
	defer logWriter.Flush()

	game := NewGame()
	game.showBuild = false
	game.autoAttackPicked = true
	game.autoAttackRanged = true
	game.aiHero.autoAttackPicked = true
	game.aiHero.autoAttackRanged = false
	game.message = "NEO balance harness: ranged chassis online"

	lastEvent := balanceEvent{}
	videoFrame := 0
	maxFrames := seconds * harnessTickRate
	for frame := 0; frame <= maxFrames; frame++ {
		if frame%harnessFrameSkip == 0 {
			event := makeBalanceEvent(game)
			if err := json.NewEncoder(logWriter).Encode(event); err != nil {
				return err
			}
			lastEvent = event
			if err := renderBalanceFrame(game, event, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.png", videoFrame))); err != nil {
				return err
			}
			videoFrame++
		}
		if game.won || game.lost || frame == maxFrames {
			break
		}
		balanceTick(game)
	}

	summary := balanceSummary{
		Version: version, Outcome: lastEvent.Outcome, DurationSec: lastEvent.Seconds,
		PlayerDeaths: game.playerDeaths, EnemyDeaths: game.aiHero.deaths,
		BlueTowerHP: game.playerTowerHealth, RedTowerHP: game.enemyTowerHealth,
	}
	summaryFile, err := os.Create(filepath.Join(outputRoot, "summary.json"))
	if err != nil {
		return err
	}
	defer summaryFile.Close()
	if err := json.NewEncoder(summaryFile).Encode(summary); err != nil {
		return err
	}
	fmt.Printf("balance harness complete: outcome=%s duration=%ds player_deaths=%d enemy_deaths=%d blue_tower=%d red_tower=%d frames=%d\n", summary.Outcome, summary.DurationSec, summary.PlayerDeaths, summary.EnemyDeaths, summary.BlueTowerHP, summary.RedTowerHP, videoFrame)
	return nil
}

func balanceTick(game *Game) {
	if game.won || game.lost {
		return
	}
	if !game.playerActive {
		game.updateMinionWave()
		game.updateTowerAttacks()
		game.updateAIHero()
		game.updatePlayerRespawn()
		game.matchFrames++
		return
	}
	if game.playerHealth <= 3 {
		game.attackTower = false
		game.attackHero = false
		game.attackTargetIsCreep = false
		game.hasTarget = false
		moveHarnessPlayer(game, blueHeroSpawnX, laneY)
	} else if game.aiHero.active && game.aiHero.threat && distanceBetween(game.playerX, game.playerY, game.aiHero.x, game.aiHero.y) < 260 {
		game.attackTower = false
		game.attackHero = true
		game.targetX, game.targetY = game.aiHero.x, game.aiHero.y
		moveHarnessPlayer(game, game.aiHero.x-150, game.aiHero.y)
	} else {
		game.attackHero = false
		game.attackTower = true
		game.targetX, game.targetY = 864, laneY
		moveHarnessPlayer(game, 864-150, laneY)
	}
	game.updateAutoAttack()
	game.updateEnemyAttacks()
	game.updateMinionWave()
	game.updateTowerAttacks()
	game.updateAIHero()
	game.checkMatchEnd()
	game.matchFrames++
	if game.attackCooldown > 0 {
		game.attackCooldown--
	}
}

func moveHarnessPlayer(game *Game, targetX, targetY float64) {
	dx, dy := targetX-game.playerX, targetY-game.playerY
	distance := distanceBetween(game.playerX, game.playerY, targetX, targetY)
	if distance < 1 {
		return
	}
	step := 3.8
	if step > distance {
		step = distance
	}
	nextX := game.playerX + dx/distance*step
	nextY := game.playerY + dy/distance*step
	if canOccupy(nextX, game.playerY, 20, 20) {
		game.playerX = nextX
	}
	if canOccupy(game.playerX, nextY, 20, 20) {
		game.playerY = nextY
	}
}

func distanceBetween(x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	return sqrt(dx*dx + dy*dy)
}

func sqrt(value float64) float64 {
	guess := value
	if guess == 0 {
		return 0
	}
	for i := 0; i < 8; i++ {
		guess = (guess + value/guess) / 2
	}
	return guess
}

func makeBalanceEvent(game *Game) balanceEvent {
	outcome := "RUNNING"
	if game.won {
		outcome = "WON"
	} else if game.lost {
		outcome = "LOST"
	}
	return balanceEvent{
		Frame: game.matchFrames, Seconds: game.matchFrames / harnessTickRate,
		PlayerHP: game.playerHealth, EnemyHeroHP: game.aiHero.health,
		BlueTowerHP: game.playerTowerHealth, RedTowerHP: game.enemyTowerHealth,
		PlayerDeaths: game.playerDeaths, EnemyDeaths: game.aiHero.deaths,
		Outcome: outcome, Message: game.message,
	}
}

func renderBalanceFrame(game *Game, event balanceEvent, path string) error {
	canvas := image.NewRGBA(image.Rect(0, 0, harnessWidth, harnessHeight))
	fill(canvas, color.RGBA{7, 11, 18, 255})
	fillRect(canvas, 48, 70, 624, 4, color.RGBA{70, 110, 120, 255})
	fillRect(canvas, 48, 490, 624, 4, color.RGBA{70, 110, 120, 255})
	fillRect(canvas, 48, 70, 4, 424, color.RGBA{70, 110, 120, 255})
	fillRect(canvas, 668, 70, 4, 424, color.RGBA{70, 110, 120, 255})
	fillRect(canvas, 48, 220, 624, 170, color.RGBA{28, 38, 48, 255})
	fillRect(canvas, 48, 302, 624, 5, color.RGBA{63, 87, 96, 255})
	drawTowerCard(canvas, 92, 280, "BLUE TOWER", game.playerTowerHealth, color.RGBA{92, 196, 207, 255})
	drawTowerCard(canvas, 628, 280, "RED TOWER", game.enemyTowerHealth, color.RGBA{220, 90, 100, 255})
	for _, minion := range game.minions {
		if minion.active {
			drawCreep(canvas, harnessX(minion.x), harnessY(minion.y), minion.health, color.RGBA{86, 178, 205, 255})
		}
	}
	for _, minion := range game.enemyMinions {
		if minion.active {
			drawCreep(canvas, harnessX(minion.x), harnessY(minion.y), minion.health, color.RGBA{190, 75, 85, 255})
		}
	}
	if game.aiHero.active {
		drawUnit(canvas, harnessX(game.aiHero.x), harnessY(game.aiHero.y), game.aiHero.health, color.RGBA{220, 90, 100, 255})
	}
	drawUnit(canvas, harnessX(game.playerX), harnessY(game.playerY), game.playerHealth, color.RGBA{90, 205, 225, 255})

	drawLabel(canvas, "RUIN CIRCUIT // AI BALANCE RUN", 48, 42, color.RGBA{190, 224, 224, 255})
	drawLabel(canvas, "TIKTOK REPLAY // "+version, 48, 590, color.RGBA{150, 190, 195, 255})
	drawLabel(canvas, fmt.Sprintf("MATCH TIME  %02d:%02d", event.Seconds/60, event.Seconds%60), 48, 650, color.RGBA{240, 216, 150, 255})
	drawLabel(canvas, fmt.Sprintf("BLUE TOWER  %02d/%02d", event.BlueTowerHP, maxTowerHP), 48, 715, color.RGBA{125, 221, 225, 255})
	drawLabel(canvas, fmt.Sprintf("RED TOWER   %02d/%02d", event.RedTowerHP, maxTowerHP), 48, 750, color.RGBA{245, 140, 145, 255})
	drawLabel(canvas, fmt.Sprintf("PLAYER HP   %02d/%02d   DEATHS %d", event.PlayerHP, maxHealth, event.PlayerDeaths), 48, 800, color.White)
	drawLabel(canvas, fmt.Sprintf("ENEMY HP    %02d/%02d   DEATHS %d", event.EnemyHeroHP, maxHealth, event.EnemyDeaths), 48, 835, color.White)
	drawLabel(canvas, "AI POLICY: RANGED PUSH / RETREAT AT 3 HP", 48, 905, color.RGBA{180, 205, 205, 255})
	drawLabel(canvas, event.Outcome, 48, 1010, color.RGBA{255, 230, 150, 255})
	if event.Message != "" {
		drawLabel(canvas, event.Message, 48, 1060, color.RGBA{220, 200, 150, 255})
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, canvas)
}

func fill(canvas *image.RGBA, fillColor color.Color) {
	stddraw.Draw(canvas, canvas.Bounds(), image.NewUniform(fillColor), image.Point{}, stddraw.Src)
}

func fillRect(canvas *image.RGBA, x, y, width, height int, fillColor color.Color) {
	stddraw.Draw(canvas, image.Rect(x, y, x+width, y+height), image.NewUniform(fillColor), image.Point{}, stddraw.Src)
}

func drawTowerCard(canvas *image.RGBA, x, y int, label string, health int, fillColor color.Color) {
	drawLabel(canvas, label, x-42, y-48, fillColor)
	fillRect(canvas, x-42, y-32, 84, 10, color.RGBA{25, 30, 35, 255})
	fillRect(canvas, x-42, y-32, 84*health/maxTowerHP, 10, fillColor)
	fillRect(canvas, x-18, y-18, 36, 36, fillColor)
}

func drawUnit(canvas *image.RGBA, x, y, health int, fillColor color.Color) {
	fillRect(canvas, x-12, y-12, 24, 24, fillColor)
	fillRect(canvas, x-20, y-25, 40, 5, color.RGBA{25, 30, 35, 255})
	fillRect(canvas, x-20, y-25, 40*health/maxHealth, 5, fillColor)
}

func drawCreep(canvas *image.RGBA, x, y, health int, fillColor color.Color) {
	fillRect(canvas, x-6, y-6, 12, 12, fillColor)
	fillRect(canvas, x-8, y-16, 16, 3, color.RGBA{25, 30, 35, 255})
	fillRect(canvas, x-8, y-16, 8*health, 3, fillColor)
}

func harnessX(worldX float64) int {
	return 92 + int((worldX-blueHeroSpawnX)*536/(redHeroSpawnX-blueHeroSpawnX))
}

func harnessY(worldY float64) int {
	return 306 + int((worldY-laneY)*1.3)
}

func drawLabel(canvas *image.RGBA, value string, x, y int, textColor color.Color) {
	drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(textColor), Face: basicfont.Face7x13, Dot: fixed.P(x, y)}
	drawer.DrawString(value)
}
