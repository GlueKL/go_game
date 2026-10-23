// Akvelon RPG SAGA — турнир героев с графикой на Ebitengine.
package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/gluekl/go_game/rpg"
)

type scene interface {
	Update(dt float64)
	Draw(screen *ebiten.Image)
}

type Game struct {
	scene        scene
	tournament   *rpg.Tournament
	playersCount int
	log          []logLine
	speed        int
	paused       bool
	worldBuf     *ebiten.Image
}

func newGame() *Game {
	g := &Game{speed: 1, playersCount: 8, worldBuf: ebiten.NewImage(screenW, screenH)}
	g.scene = newTitleScene(g)
	return g
}

func (g *Game) setScene(s scene) { g.scene = s }

func (g *Game) newTournament(count int) {
	g.tournament = rpg.NewTournament(rand.New(rand.NewSource(time.Now().UnixNano())), count)
	g.log = nil
	g.addLog("Добро пожаловать в Akvelon RPG SAGA!", colGold)
	g.setScene(newRosterScene(g))
}

func (g *Game) addLog(s string, clr color.RGBA) {
	g.log = append(g.log, logLine{text: s, clr: clr})
	if len(g.log) > 100 {
		g.log = g.log[len(g.log)-100:]
	}
}

func (g *Game) Update() error {
	if b, ok := g.scene.(*battleScene); ok {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.paused = !g.paused
		}
		for i, k := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3} {
			if inpututil.IsKeyJustPressed(k) {
				g.speed = 1 << i
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			b.skip = true
			g.paused = false
		}
		if g.paused {
			return nil
		}
		g.scene.Update(float64(g.speed) / 60)
		return nil
	}
	g.scene.Update(1.0 / 60)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.scene.Draw(screen)
}

func (g *Game) Layout(int, int) (int, int) { return screenW, screenH }

func (g *Game) drawLog(dst *ebiten.Image, x, y, w, h float32) {
	drawPanel(dst, x, y, w, h)
	drawText(dst, "Журнал боя", face(true, 16), float64(x+16), float64(y+10), colGold, alignLeft)
	const lines = 6
	start := max(0, len(g.log)-lines)
	for i, l := range g.log[start:] {
		age := float64(len(g.log)-start-1-i) / lines
		drawText(dst, l.text, face(false, 17), float64(x+16), float64(y+36)+float64(i)*20.5, fade(l.clr, 1-age*0.7), alignLeft)
	}
}

func (g *Game) drawStandings(dst *ebiten.Image, x, y, w, h float32, b *rpg.Battle) {
	drawPanel(dst, x, y, w, h)
	t := g.tournament
	alive := 0
	for _, p := range t.Players {
		if p.Alive {
			alive++
		}
	}
	drawText(dst, fmt.Sprintf("Участники · в строю %d из %d", alive, len(t.Players)),
		face(true, 16), float64(x+16), float64(y+10), colGold, alignLeft)
	const rows = 8
	colW := (w - 32) / 2
	f := face(false, 14)
	for i, p := range t.Players {
		cx := x + 16 + float32(i/rows)*colW
		cy := y + 36 + float32(i%rows)*15.5
		fighting := b != nil && (b.Fighters[0] == p || b.Fighters[1] == p)
		clr := colWhite
		switch {
		case fighting:
			clr = colGold
		case !p.Alive:
			clr = fade(colMuted, 0.6)
		}
		dot := classColor(p.Class)
		if !p.Alive {
			dot = fade(dot, 0.35)
		}
		fillCircle(dst, cx+5, cy+8, 4, dot)
		label := fmt.Sprintf("%s · %s", p.Name, p.Class)
		if p.Wins > 0 {
			label += fmt.Sprintf(" · %d поб.", p.Wins)
		}
		drawText(dst, label, f, float64(cx+16), float64(cy), clr, alignLeft)
		if !p.Alive {
			vector.StrokeLine(dst, cx+14, cy+9, cx+16+float32(textWidth(label, f)), cy+9, 1.5, fade(colRed, 0.8), true)
		}
	}
}

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Akvelon RPG SAGA")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
