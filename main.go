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
	version      = "v0.0.1"
)

type Enemy struct {
	x, y   float64
	name   string
	active bool
}

type Game struct {
	playerX, playerY float64
	targetX, targetY float64
	hasTarget        bool
	enemies          []Enemy
	showBuild        bool
	message          string
}

func NewGame() *Game {
	return &Game{
		playerX: 480,
		playerY: 300,
		enemies: []Enemy{
			{x: 220, y: 180, name: "scavenger drone", active: true},
			{x: 740, y: 170, name: "scrap hound", active: true},
			{x: 700, y: 390, name: "vault sentinel", active: true},
		},
		message: "Right-click to move. Q/W/E/R are waiting for their first components.",
	}
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mouseX, mouseY := ebiten.CursorPosition()
		g.targetX, g.targetY = float64(mouseX), float64(mouseY)
		g.hasTarget = true
		g.message = fmt.Sprintf("Moving to %.0f, %.0f", g.targetX, g.targetY)
	}
	if g.hasTarget {
		dx, dy := g.targetX-g.playerX, g.targetY-g.playerY
		distance := math.Hypot(dx, dy)
		if distance < 3 {
			g.hasTarget = false
		} else {
			step := 3.8
			g.playerX += dx / distance * math.Min(step, distance)
			g.playerY += dy / distance * math.Min(step, distance)
		}
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
	ruins := [][4]float64{{90, 90, 180, 18}, {90, 90, 18, 115}, {680, 80, 190, 18}, {850, 80, 18, 130}, {330, 420, 260, 18}, {330, 350, 18, 88}}
	for _, r := range ruins {
		ebitenutil.DrawRect(screen, r[0], r[1], r[2], r[3], color.RGBA{52, 66, 82, 255})
		ebitenutil.DrawRect(screen, r[0]+3, r[1]+3, r[2]-6, 3, color.RGBA{82, 105, 124, 255})
	}
	for _, enemy := range g.enemies {
		if !enemy.active {
			continue
		}
		ebitenutil.DrawRect(screen, enemy.x-10, enemy.y-8, 20, 16, color.RGBA{145, 58, 65, 255})
		ebitenutil.DrawRect(screen, enemy.x-4, enemy.y-13, 8, 5, color.RGBA{227, 153, 74, 255})
	}
	if g.hasTarget {
		ebitenutil.DrawRect(screen, g.targetX-5, g.targetY-1, 10, 2, color.RGBA{110, 219, 222, 220})
		ebitenutil.DrawRect(screen, g.targetX-1, g.targetY-5, 2, 10, color.RGBA{110, 219, 222, 220})
	}
	// Player combat chassis.
	ebitenutil.DrawRect(screen, g.playerX-11, g.playerY-11, 22, 22, color.RGBA{92, 196, 207, 255})
	ebitenutil.DrawRect(screen, g.playerX-5, g.playerY-17, 10, 7, color.RGBA{191, 235, 220, 255})
	ebitenutil.DrawRect(screen, g.playerX+9, g.playerY-3, 13, 5, color.RGBA{231, 177, 86, 255})

	text.Draw(screen, "RUIN CIRCUIT // COMBAT CHASSIS ONLINE", basicfont.Face7x13, 32, 26, color.RGBA{190, 224, 224, 255})
	text.Draw(screen, version, basicfont.Face7x13, 880, 26, color.RGBA{150, 190, 195, 255})
	text.Draw(screen, "RIGHT-CLICK MOVE", basicfont.Face7x13, 760, 26, color.RGBA{150, 180, 190, 255})
	text.Draw(screen, g.message, basicfont.Face7x13, 32, 520, color.RGBA{220, 200, 150, 255})
	for i, skill := range []string{"Q  COMPONENT", "W  COMPONENT", "E  COMPONENT", "R  COMPONENT"} {
		x := 32 + i*142
		ebitenutil.DrawRect(screen, float64(x), 465, 132, 34, color.RGBA{18, 28, 39, 245})
		text.Draw(screen, skill, basicfont.Face7x13, x+9, 486, color.RGBA{125, 221, 225, 255})
	}
	if g.showBuild {
		ebitenutil.DrawRect(screen, 220, 72, 520, 300, color.RGBA{8, 14, 24, 242})
		text.Draw(screen, "BUILD WORKBENCH", basicfont.Face7x13, 400, 105, color.RGBA{240, 216, 150, 255})
		text.Draw(screen, "Five component sockets will define every ability:", basicfont.Face7x13, 270, 145, color.White)
		text.Draw(screen, "TRIGGER   TARGETING   EFFECT   MODIFIER   SCALING", basicfont.Face7x13, 270, 180, color.RGBA{125, 221, 225, 255})
		text.Draw(screen, "Tech currency and dungeon materials will fill this panel.", basicfont.Face7x13, 270, 220, color.RGBA{190, 200, 210, 255})
		text.Draw(screen, "TAB to close", basicfont.Face7x13, 430, 330, color.RGBA{240, 216, 150, 255})
	}
}

func (g *Game) Layout(_, _ int) (int, int) { return screenWidth, screenHeight }

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Ruin Circuit")
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}
