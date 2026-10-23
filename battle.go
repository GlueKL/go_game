package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/gluekl/go_game/rpg"
)

const (
	leftX  = 390.0
	rightX = 890.0
	footY  = groundY + 18.0
)

type fighterView struct {
	p      *rpg.Player
	sprite *sprite
	homeX  float64
	dir    float64 // 1 — смотрит вправо, -1 — влево
	hp     float64 // здоровье, отображаемое на полоске (догоняет hpTarget)
	hpGoal float64
	debuff rpg.Debuff
	offX   float64
	pose   float64
	flash  float64
	dying  float64
	dead   bool
}

func newFighterView(p *rpg.Player, x, dir float64) *fighterView {
	return &fighterView{p: p, sprite: newSprite(), homeX: x, dir: dir, hp: float64(p.Health), hpGoal: float64(p.Health)}
}

func (v *fighterView) x() float64 { return v.homeX + v.offX*v.dir }

// hand — экранные координаты оружия (откуда летят снаряды).
func (v *fighterView) hand() (float64, float64) {
	hx, hy := handPos(v.p.Class, v.pose)
	return v.x() + hx*v.dir, footY + hy
}

func (v *fighterView) chest() (float64, float64) { return v.x(), footY - 110 }

type logLine struct {
	text string
	clr  color.RGBA
}

type projectile struct {
	kind           rpg.Class
	ability        bool
	x0, y0, x1, y1 float64
	t              float64 // 0..1
}

const (
	phaseIntro = iota
	phaseFight
	phaseOutro
)

type battleScene struct {
	g      *Game
	battle *rpg.Battle
	views  [2]*fighterView

	phase  int
	phaseT float64
	time   float64

	queue    []rpg.Event
	cur      *rpg.Event
	curT     float64
	impacted bool
	shot     *projectile

	fx    effects
	shake float64
	skip  bool // промотать бой до конца
}

func newBattleScene(g *Game) *battleScene {
	b := g.tournament.NextBattle()
	s := &battleScene{g: g, battle: b}
	s.views[0] = newFighterView(b.Fighters[0], leftX, 1)
	s.views[1] = newFighterView(b.Fighters[1], rightX, -1)
	g.addLog(fmt.Sprintf("Бой №%d: %s (%s) против %s (%s)", g.tournament.Number,
		b.Fighters[0].Name, b.Fighters[0].Class, b.Fighters[1].Name, b.Fighters[1].Class), colGold)
	return s
}

func (s *battleScene) view(p *rpg.Player) *fighterView {
	if s.views[0].p == p {
		return s.views[0]
	}
	return s.views[1]
}

func (s *battleScene) other(v *fighterView) *fighterView {
	if s.views[0] == v {
		return s.views[1]
	}
	return s.views[0]
}

// Длительность события и момент попадания (в секундах при скорости 1x).
func timing(ev *rpg.Event) (dur, impact float64) {
	switch ev.Kind {
	case rpg.EvAttack:
		return 1.0, 0.55
	case rpg.EvAbility:
		return 1.45, 0.85
	case rpg.EvBurn:
		return 0.75, 0.2
	case rpg.EvSkip:
		return 0.95, 0.15
	default:
		return 1.6, 0.05
	}
}

func (s *battleScene) Update(dt float64) {
	if s.skip && s.phase != phaseOutro {
		s.finishInstantly()
	}
	s.time += dt
	s.phaseT += dt
	s.fx.update(dt)
	s.shake = math.Max(0, s.shake-dt*3)

	for _, v := range s.views {
		v.hp += (v.hpGoal - v.hp) * math.Min(1, dt*6)
		v.flash = math.Max(0, v.flash-dt*4)
		if v.debuff == rpg.Burning && !v.dead && rand.Float64() < dt*14 {
			x, y := v.chest()
			s.fx.fire(x, y+20, 50, 1)
		}
	}

	switch s.phase {
	case phaseIntro:
		if s.phaseT > 1.6 {
			s.phase, s.phaseT = phaseFight, 0
		}
	case phaseFight:
		s.updateFight(dt)
	case phaseOutro:
		if s.phaseT > 2.4 {
			s.g.tournament.FinishBattle()
			if s.g.tournament.Champion != nil {
				s.g.setScene(newChampionScene(s.g))
			} else {
				s.g.setScene(newBattleScene(s.g))
			}
		}
	}
}

func (s *battleScene) updateFight(dt float64) {
	if s.cur == nil {
		if len(s.queue) == 0 {
			if s.battle.Over() {
				s.phase, s.phaseT = phaseOutro, 0
				s.g.addLog(fmt.Sprintf("Победитель боя — %s!", s.battle.Winner.Name), colGold)
				return
			}
			s.queue = s.battle.Step(s.g.tournament.Rng())
		}
		ev := s.queue[0]
		s.queue = s.queue[1:]
		s.cur, s.curT, s.impacted, s.shot = &ev, 0, false, nil
		s.g.addLog(ev.Text, logColor(ev.Kind))
		if ev.Kind == rpg.EvAbility {
			x, _ := s.view(ev.Actor).chest()
			s.fx.text(ev.Actor.Class.AbilityName()+"!", x, footY-250, 26, colGold)
		}
	}

	ev := s.cur
	s.curT += dt
	dur, impact := timing(ev)
	s.animate(ev, s.curT/impact, (s.curT-impact)/(dur-impact))
	if !s.impacted && s.curT >= impact {
		s.impacted = true
		s.applyImpact(ev)
	}
	if s.curT >= dur {
		a := s.view(ev.Actor)
		a.pose, a.offX = 0, 0
		s.cur, s.shot = nil, nil
	}
}

// finishInstantly доигрывает бой без анимаций и сразу показывает итог.
func (s *battleScene) finishInstantly() {
	for !s.battle.Over() || len(s.queue) > 0 {
		if len(s.queue) == 0 {
			s.queue = s.battle.Step(s.g.tournament.Rng())
		}
		ev := s.queue[0]
		s.queue = s.queue[1:]
		s.g.addLog(ev.Text, logColor(ev.Kind))
	}
	for _, v := range s.views {
		v.pose, v.offX = 0, 0
		v.hpGoal = float64(v.p.Health)
		v.debuff = v.p.Debuff
		if !v.p.Alive {
			v.dead, v.dying = true, 1
		}
	}
	s.cur, s.shot = nil, nil
	s.phase, s.phaseT = phaseOutro, 0
	s.g.addLog(fmt.Sprintf("Победитель боя — %s!", s.battle.Winner.Name), colGold)
}

// animate двигает атакующего. in — прогресс до попадания, out — после него.
func (s *battleScene) animate(ev *rpg.Event, in, out float64) {
	switch ev.Kind {
	case rpg.EvAttack, rpg.EvAbility:
	case rpg.EvDeath:
		v := s.view(ev.Subject)
		v.dying = ease(out)
		return
	default:
		return
	}
	a := s.view(ev.Actor)
	t := s.view(ev.Subject)
	ability := ev.Kind == rpg.EvAbility

	if ability && in < 1 && rand.Float64() < 0.5 {
		x, y := a.chest()
		s.fx.burst(x, y, 1, mix(classColor(a.p.Class), colWhite, 0.4), 120, 0, 4)
	}

	switch a.p.Class {
	case rpg.Warrior:
		dist := math.Abs(t.homeX-a.homeX) - 130
		switch {
		case in < 0.45:
			a.offX = -20 * ease(in/0.45)
			a.pose = -0.2 * ease(in/0.45)
		case in < 1:
			k := ease((in - 0.45) / 0.55)
			a.offX = lerp(-20, dist, k)
			a.pose = lerp(-0.2, 1, k)
		default:
			k := ease(out * 1.4)
			a.offX = lerp(dist, 0, k)
			a.pose = lerp(1, 0, k)
		}
	default:
		const release = 0.6
		if in < release {
			a.pose = ease(in / release)
		} else if in < 1 {
			a.pose = 1
			if s.shot == nil {
				hx, hy := a.hand()
				cx, cy := t.chest()
				s.shot = &projectile{kind: a.p.Class, ability: ability, x0: hx, y0: hy, x1: cx + 20*t.dir, y1: cy}
			}
			s.shot.t = (in - release) / (1 - release)
			if ability && s.shot != nil {
				x, y := s.shot.pos()
				if a.p.Class == rpg.Archer {
					s.fx.fire(x, y, 8, 1)
				} else {
					s.fx.burst(x, y, 1, colPurple, 40, 0, 5)
				}
			}
		} else {
			s.shot = nil
			a.pose = lerp(1, 0, ease(out*2))
		}
	}
}

func (p *projectile) pos() (float64, float64) {
	x := lerp(p.x0, p.x1, p.t)
	y := lerp(p.y0, p.y1, p.t)
	if p.kind == rpg.Archer {
		y -= math.Sin(p.t*math.Pi) * 30
	}
	return x, y
}

func (s *battleScene) applyImpact(ev *rpg.Event) {
	v := s.view(ev.Subject)
	v.hpGoal = float64(ev.HealthAfter)
	v.debuff = ev.DebuffAfter
	x, y := v.chest()

	switch ev.Kind {
	case rpg.EvAttack, rpg.EvAbility:
		v.flash = 1
		ability := ev.Kind == rpg.EvAbility
		clr := colWhite
		switch ev.Actor.Class {
		case rpg.Warrior:
			clr = colGold
		case rpg.Mage:
			clr = colPurple
		case rpg.Archer:
			clr = rgb(0xd8, 0xc0, 0x90)
			if ability {
				clr = colOrange
			}
		}
		n, size := 14, 28
		if ability {
			n, size = 34, 40
			s.shake = 1
		}
		s.fx.burst(x, y, n, clr, 260, 300, 5)
		dmg := fmt.Sprintf("-%d", ev.Damage)
		if ability {
			dmg += "!"
		}
		s.fx.text(dmg, x, y-110, size, colRed)
		if ability && ev.Actor.Class == rpg.Archer {
			s.fx.fire(x, y, 50, 20)
			s.fx.text("Горение", x, y-150, 18, colOrange)
		}
		if ability && ev.Actor.Class == rpg.Mage {
			s.fx.text("Заворожен", x, y-150, 18, colPurple)
		}
	case rpg.EvBurn:
		v.flash = 0.6
		s.fx.fire(x, y+20, 50, 18)
		s.fx.text(fmt.Sprintf("-%d", ev.Damage), x, y-110, 24, colOrange)
	case rpg.EvSkip:
		s.fx.text("Пропуск хода", x, y-150, 20, colPurple)
		for i := 0; i < 6; i++ {
			s.fx.burst(x, y-80, 1, colPurple, 80, -40, 4)
		}
	case rpg.EvDeath:
		v.dead = true
		s.fx.burst(x, y, 30, rgb(0x8a, 0x8f, 0xa0), 160, 200, 6)
		s.shake = 0.6
	}
}

func logColor(k rpg.EventKind) color.RGBA {
	switch k {
	case rpg.EvAbility:
		return colGold
	case rpg.EvBurn:
		return colOrange
	case rpg.EvSkip:
		return colPurple
	case rpg.EvDeath:
		return colRed
	default:
		return colWhite
	}
}

func (s *battleScene) Draw(screen *ebiten.Image) {
	world := screen
	if s.shake > 0 {
		world = s.g.worldBuf
		world.Clear()
	}

	world.DrawImage(background(), nil)
	drawTorches(world, s.time)

	for _, v := range s.views {
		v.sprite.render(v.p.Class, s.time+v.homeX, v.pose)
		alpha := 1 - 0.55*v.dying
		x := v.x()
		v.sprite.draw(world, spriteDraw{
			x: x, y: footY, scale: 1, flip: v.dir < 0, alpha: alpha, flash: v.flash,
			angle: -v.dir * v.dying * math.Pi / 2.2,
		})
		if v.debuff == rpg.Skip && !v.dead {
			for i := 0; i < 3; i++ {
				a := s.time*3 + float64(i)*2*math.Pi/3
				star(world, float32(x+math.Cos(a)*26), float32(footY-170+math.Sin(a)*8), 7, s.time*4, colGold)
			}
		}
	}
	if s.shot != nil {
		drawProjectile(world, s.shot, s.time)
	}
	s.fx.draw(world)

	if world != screen {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate((rand.Float64()-0.5)*14*s.shake, (rand.Float64()-0.5)*14*s.shake)
		screen.DrawImage(world, op)
	}

	s.drawHUD(screen)

	switch s.phase {
	case phaseIntro:
		k := math.Min(1, s.phaseT*3) * math.Min(1, (1.6-s.phaseT)*3)
		drawBanner(screen, fmt.Sprintf("Бой №%d", s.g.tournament.Number),
			fmt.Sprintf("%s  против  %s", s.views[0].p.Name, s.views[1].p.Name), k)
	case phaseOutro:
		drawBanner(screen, "Победа!", s.battle.Winner.Name+" ("+s.battle.Winner.Class.String()+")", math.Min(1, s.phaseT*3))
	}
}

func drawProjectile(dst *ebiten.Image, p *projectile, t float64) {
	x, y := p.pos()
	fx, fy := float32(x), float32(y)
	switch p.kind {
	case rpg.Archer:
		dx := p.x1 - p.x0
		dy := (p.y1 - p.y0) - math.Cos(p.t*math.Pi)*30*math.Pi
		l := math.Hypot(dx, dy)
		ux, uy := float32(dx/l), float32(dy/l)
		shaft := rgb(0xd8, 0xc0, 0x90)
		if p.ability {
			fillCircle(dst, fx, fy, 14, fade(colOrange, 0.35))
			shaft = colOrange
		}
		line(dst, fx-ux*36, fy-uy*36, fx, fy, 3, shaft)
		fillPoly(dst, colSteel, fx+ux*8, fy+uy*8, fx-uy*5, fy+ux*5, fx+uy*5, fy-ux*5)
		line(dst, fx-ux*36, fy-uy*36, fx-ux*28-uy*6, fy-uy*28+ux*6, 3, colWhite)
		line(dst, fx-ux*36, fy-uy*36, fx-ux*28+uy*6, fy-uy*28-ux*6, 3, colWhite)
	case rpg.Mage:
		r := float32(10)
		if p.ability {
			r = 16
		}
		pulse := float32(1 + 0.15*math.Sin(t*20))
		fillCircle(dst, fx, fy, r*2.2*pulse, fade(colPurple, 0.2))
		fillCircle(dst, fx, fy, r*1.4, fade(colPurple, 0.6))
		fillCircle(dst, fx, fy, r*0.8, rgb(0xf2, 0xea, 0xff))
	}
}

func drawBanner(dst *ebiten.Image, title, sub string, k float64) {
	if k <= 0 {
		return
	}
	y := float32(200)
	vector.FillRect(dst, 0, y, screenW, 120, fade(color.RGBA{0x08, 0x0a, 0x16, 0xff}, 0.75*k), false)
	vector.FillRect(dst, 0, y, screenW, 3, fade(colGold, k), false)
	vector.FillRect(dst, 0, y+117, screenW, 3, fade(colGold, k), false)
	drawTextShadow(dst, title, face(true, 48), screenW/2, float64(y)+10, fade(colGold, k), alignCenter)
	drawText(dst, sub, face(false, 26), screenW/2, float64(y)+72, fade(colWhite, k), alignCenter)
}

func (s *battleScene) drawHUD(dst *ebiten.Image) {
	g := s.g
	// Верхняя полоса
	vector.FillRect(dst, 0, 0, screenW, 44, color.RGBA{0x08, 0x0a, 0x16, 0xc0}, false)
	drawText(dst, "AKVELON RPG SAGA", face(true, 22), 20, 9, colGold, alignLeft)
	drawText(dst, fmt.Sprintf("Бой №%d", g.tournament.Number), face(true, 22), screenW/2, 9, colWhite, alignCenter)
	status := fmt.Sprintf("Скорость x%d", g.speed)
	if g.paused {
		status = "ПАУЗА"
	}
	drawText(dst, status, face(true, 20), screenW-20, 10, colMuted, alignRight)

	drawFighterPanel(dst, s.views[0], 30, 60, s.time)
	drawFighterPanel(dst, s.views[1], screenW-30-410, 60, s.time)

	drawText(dst, "Пробел — пауза   ·   1 / 2 / 3 — скорость   ·   Enter — промотать бой",
		face(false, 16), screenW/2, 513, colMuted, alignCenter)

	g.drawLog(dst, 20, 540, 820, 165)
	g.drawStandings(dst, 856, 540, 404, 165, s.battle)
}

func drawFighterPanel(dst *ebiten.Image, v *fighterView, x, y float32, t float64) {
	const w, h = 410, 104
	drawPanel(dst, x, y, w, h)
	p := v.p
	cc := classColor(p.Class)
	fillCircle(dst, x+30, y+30, 14, cc)
	drawText(dst, string([]rune(p.Class.String())[0]), face(true, 13), float64(x+30), float64(y+21), colDark, alignCenter)
	drawText(dst, p.Name, face(true, 24), float64(x+54), float64(y+14), colWhite, alignLeft)
	drawText(dst, fmt.Sprintf("%s · Сила %d · Побед %d", p.Class, p.Strength, p.Wins), face(false, 16),
		float64(x+w-16), float64(y+20), colMuted, alignRight)

	// Полоска здоровья
	bx, by, bw, bh := x+16, y+54, float32(w-32), float32(24)
	fillRoundRect(dst, bx, by, bw, bh, 8, rgb(0x24, 0x1a, 0x22))
	frac := clamp(v.hp/float64(p.MaxHealth), 0, 1)
	hpClr := mix(colRed, colGold, frac*2)
	if frac > 0.5 {
		hpClr = mix(colGold, rgb(0x4c, 0xc2, 0x7c), frac*2-1)
	}
	if frac > 0 {
		fillRoundRect(dst, bx, by, float32(math.Max(16, float64(bw)*frac)), bh, 8, hpClr)
		fillRoundRect(dst, bx+4, by+3, float32(math.Max(8, float64(bw)*frac-8)), 6, 3, fade(colWhite, 0.25))
	}
	strokeRoundRect(dst, bx, by, bw, bh, 8, 2, fade(colWhite, 0.3))
	drawTextShadow(dst, fmt.Sprintf("%d / %d", int(math.Max(0, math.Round(v.hp))), p.MaxHealth),
		face(true, 16), float64(bx+bw/2), float64(by+2), colWhite, alignCenter)

	// Статусы
	sy := float64(y + 84)
	switch {
	case v.dead:
		drawText(dst, "Повержен", face(true, 14), float64(x+16), sy, colRed, alignLeft)
	case v.debuff == rpg.Burning:
		flame(dst, x+24, float32(sy)+16, 18, t)
		drawText(dst, "Горит: -2 за ход", face(true, 14), float64(x+38), sy, colOrange, alignLeft)
	case v.debuff == rpg.Skip:
		star(dst, x+24, float32(sy)+8, 8, t*3, colPurple)
		drawText(dst, "Заворожен: пропустит ход", face(true, 14), float64(x+38), sy, colPurple, alignLeft)
	default:
		drawText(dst, "Способность: «"+p.Class.AbilityName()+"»", face(false, 14), float64(x+16), sy, colMuted, alignLeft)
	}
}
