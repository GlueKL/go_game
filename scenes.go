package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/gluekl/go_game/rpg"
)

const (
	minPlayers = 2
	maxPlayers = 16
)

// titleScene — стартовый экран с выбором количества игроков.
type titleScene struct {
	g       *Game
	time    float64
	count   int
	sprites [3]*sprite
	minus   button
	plus    button
	start   button
}

func newTitleScene(g *Game) *titleScene {
	s := &titleScene{g: g, count: g.playersCount}
	for i := range s.sprites {
		s.sprites[i] = newSprite()
	}
	s.minus = button{x: screenW/2 - 190, y: 492, w: 60, h: 60, label: "−"}
	s.plus = button{x: screenW/2 + 130, y: 492, w: 60, h: 60, label: "+"}
	s.start = button{x: screenW/2 - 170, y: 602, w: 340, h: 66, label: "Начать турнир"}
	return s
}

func (s *titleScene) Update(dt float64) {
	s.time += dt
	if s.minus.clicked() || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		s.count = max(minPlayers, s.count-2)
	}
	if s.plus.clicked() || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		s.count = min(maxPlayers, s.count+2)
	}
	if s.start.clicked() || inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.g.playersCount = s.count
		s.g.newTournament(s.count)
	}
}

func (s *titleScene) Draw(screen *ebiten.Image) {
	screen.DrawImage(background(), nil)
	drawTorches(screen, s.time)

	classes := []rpg.Class{rpg.Warrior, rpg.Mage, rpg.Archer}
	xs := []float64{screenW/2 - 260, screenW / 2, screenW/2 + 260}
	for i, c := range classes {
		s.sprites[i].render(c, s.time+float64(i), 0.15+0.15*math.Sin(s.time*1.5+float64(i)))
		s.sprites[i].draw(screen, spriteDraw{x: xs[i], y: footY - 64, scale: 0.95, flip: i == 2, alpha: 1})
		drawTextShadow(screen, c.String(), face(true, 22), xs[i], footY-56, classColor(c), alignCenter)
	}

	pulse := 1 + 0.03*math.Sin(s.time*2)
	drawTextShadow(screen, "AKVELON RPG SAGA", face(true, int(76*pulse)), screenW/2, 40, colGold, alignCenter)
	drawText(screen, "Турнир героев: последний выживший забирает славу", face(false, 24), screenW/2, 130, colWhite, alignCenter)

	drawPanel(screen, screenW/2-220, 480, 440, 84)
	s.minus.draw(screen)
	s.plus.draw(screen)
	drawText(screen, "Бойцов", face(false, 18), screenW/2, 488, colMuted, alignCenter)
	drawTextShadow(screen, fmt.Sprint(s.count), face(true, 40), screenW/2, 508, colWhite, alignCenter)
	s.start.draw(screen)
	drawText(screen, "← / → — количество   ·   Enter — начать", face(false, 16), screenW/2, 688, colMuted, alignCenter)
}

// rosterScene показывает карточки всех участников перед турниром.
type rosterScene struct {
	g       *Game
	time    float64
	sprites []*sprite
	start   button
}

func newRosterScene(g *Game) *rosterScene {
	s := &rosterScene{g: g}
	for _, p := range g.tournament.Players {
		sp := newSprite()
		sp.render(p.Class, float64(p.ID), 0)
		s.sprites = append(s.sprites, sp)
	}
	s.start = button{x: screenW/2 - 150, y: 628, w: 300, h: 64, label: "В бой!"}
	return s
}

func (s *rosterScene) Update(dt float64) {
	s.time += dt
	if s.start.clicked() || inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.g.setScene(newBattleScene(s.g))
	}
}

func (s *rosterScene) Draw(screen *ebiten.Image) {
	screen.DrawImage(background(), nil)
	vector.FillRect(screen, 0, 0, screenW, screenH, color.RGBA{0x05, 0x06, 0x10, 0x90}, false)
	drawTextShadow(screen, "Участники турнира", face(true, 44), screenW/2, 24, colGold, alignCenter)

	players := s.g.tournament.Players
	cols := min(4, len(players))
	rows := (len(players) + cols - 1) / cols
	const cw, ch, gap = 290.0, 120.0, 14.0
	x0 := (screenW - (float64(cols)*cw + float64(cols-1)*gap)) / 2
	y0 := 96 + (520-(float64(rows)*ch+float64(rows-1)*gap))/2
	for i, p := range players {
		x := x0 + float64(i%cols)*(cw+gap)
		y := y0 + float64(i/cols)*(ch+gap)
		appear := ease(s.time*3 - float64(i)*0.12)
		if appear <= 0 {
			continue
		}
		y += (1 - appear) * 30
		drawPanel(screen, float32(x), float32(y), cw, ch)
		vector.FillRect(screen, float32(x)+2, float32(y)+10, 4, ch-20, classColor(p.Class), true)
		s.sprites[i].draw(screen, spriteDraw{x: x + 58, y: y + ch - 8, scale: 0.46, alpha: appear})
		tx := x + 110
		drawText(screen, p.Name, face(true, 22), tx, y+12, fade(colWhite, appear), alignLeft)
		drawText(screen, p.Class.String(), face(true, 16), tx, y+42, fade(classColor(p.Class), appear), alignLeft)
		drawText(screen, fmt.Sprintf("Здоровье: %d", p.MaxHealth), face(false, 16), tx, y+66, fade(colMuted, appear), alignLeft)
		drawText(screen, fmt.Sprintf("Сила: %d", p.Strength), face(false, 16), tx, y+88, fade(colMuted, appear), alignLeft)
	}
	s.start.draw(screen)
}

// championScene — финальный экран с победителем турнира.
type championScene struct {
	g      *Game
	time   float64
	sprite *sprite
	fx     effects
	again  button
}

func newChampionScene(g *Game) *championScene {
	s := &championScene{g: g, sprite: newSprite()}
	s.fx.confetti(160)
	s.again = button{x: screenW/2 - 170, y: 620, w: 340, h: 66, label: "Новый турнир"}
	g.addLog(fmt.Sprintf("%s выиграл турнир!", g.tournament.Champion.Name), colGold)
	return s
}

func (s *championScene) Update(dt float64) {
	s.time += dt
	s.fx.update(dt)
	if len(s.fx.particles) < 80 {
		s.fx.confetti(40)
	}
	// Короткая задержка, чтобы случайное нажатие Enter при перемотке не закрыло экран.
	if s.time > 1.5 && (s.again.clicked() || inpututil.IsKeyJustPressed(ebiten.KeyR)) {
		s.g.setScene(newTitleScene(s.g))
	}
}

func (s *championScene) Draw(screen *ebiten.Image) {
	screen.DrawImage(background(), nil)
	drawTorches(screen, s.time)
	c := s.g.tournament.Champion

	// Лучи света за чемпионом
	cx, cy := float32(screenW/2), float32(330)
	for i := 0; i < 12; i++ {
		a := s.time*0.3 + float64(i)*math.Pi/6
		x1 := cx + 700*float32(math.Cos(a-0.08))
		y1 := cy + 700*float32(math.Sin(a-0.08))
		x2 := cx + 700*float32(math.Cos(a+0.08))
		y2 := cy + 700*float32(math.Sin(a+0.08))
		fillPoly(screen, fade(colGold, 0.08), cx, cy, x1, y1, x2, y2)
	}
	fillEllipse(screen, screenW/2, footY+4, 150, 22, rgb(0x6b, 0x5a, 0x48))
	fillEllipse(screen, screenW/2, footY-2, 140, 18, rgb(0x8a, 0x76, 0x60))

	s.sprite.render(c.Class, s.time, 0.3+0.3*math.Sin(s.time*2))
	s.sprite.draw(screen, spriteDraw{x: screenW / 2, y: footY, scale: 1.25, alpha: 1})

	// Корона
	kx, ky := float32(screenW/2), float32(footY-235+math.Sin(s.time*2)*4)
	fillPoly(screen, colGold, kx-30, ky+20, kx-34, ky-10, kx-16, ky+4, kx, ky-20, kx+16, ky+4, kx+34, ky-10, kx+30, ky+20)
	fillCircle(screen, kx, ky-20, 4, colRed)
	fillCircle(screen, kx-34, ky-10, 4, colPurple)
	fillCircle(screen, kx+34, ky-10, 4, colPurple)

	s.fx.draw(screen)

	drawTextShadow(screen, "Чемпион турнира", face(true, 34), screenW/2, 40, colWhite, alignCenter)
	drawTextShadow(screen, c.Name, face(true, 72), screenW/2, 84, colGold, alignCenter)
	drawText(screen, fmt.Sprintf("%s · Здоровье %d · Сила %d · Побед %d", c.Class, c.MaxHealth, c.Strength, c.Wins),
		face(false, 22), screenW/2, 176, colWhite, alignCenter)
	s.again.draw(screen)
}
