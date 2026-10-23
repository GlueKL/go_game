package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

type particle struct {
	x, y, vx, vy float64
	life, max    float64
	size         float64
	gravity      float64
	clr          color.RGBA
	isStar       bool
}

type floatText struct {
	text string
	x, y float64
	life float64
	size int
	clr  color.RGBA
}

// effects хранит частицы и всплывающие надписи.
type effects struct {
	particles []particle
	texts     []floatText
}

func (e *effects) burst(x, y float64, n int, clr color.RGBA, speed, gravity, size float64) {
	for i := 0; i < n; i++ {
		a := rand.Float64() * 2 * math.Pi
		v := speed * (0.3 + rand.Float64()*0.7)
		life := 0.4 + rand.Float64()*0.5
		e.particles = append(e.particles, particle{
			x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a) * v,
			life: life, max: life, size: size * (0.5 + rand.Float64()*0.7),
			gravity: gravity, clr: clr,
		})
	}
}

// fire добавляет поднимающиеся вверх искры пламени.
func (e *effects) fire(x, y, spread float64, n int) {
	for i := 0; i < n; i++ {
		life := 0.4 + rand.Float64()*0.5
		clr := colOrange
		if rand.Intn(3) == 0 {
			clr = colGold
		} else if rand.Intn(3) == 0 {
			clr = colRed
		}
		e.particles = append(e.particles, particle{
			x: x + (rand.Float64()-0.5)*spread, y: y + (rand.Float64()-0.5)*spread*0.5,
			vx: (rand.Float64() - 0.5) * 30, vy: -60 - rand.Float64()*80,
			life: life, max: life, size: 3 + rand.Float64()*5, clr: clr,
		})
	}
}

func (e *effects) confetti(n int) {
	palette := []color.RGBA{colGold, colRed, colPurple, rgb(0x4c, 0xc2, 0x7c), rgb(0x4f, 0xa8, 0xe0)}
	for i := 0; i < n; i++ {
		life := 3 + rand.Float64()*2
		e.particles = append(e.particles, particle{
			x: rand.Float64() * screenW, y: -20 - rand.Float64()*200,
			vx: (rand.Float64() - 0.5) * 60, vy: 60 + rand.Float64()*90,
			life: life, max: life, size: 4 + rand.Float64()*4,
			clr: palette[rand.Intn(len(palette))], isStar: rand.Intn(2) == 0,
		})
	}
}

func (e *effects) text(s string, x, y float64, size int, clr color.RGBA) {
	e.texts = append(e.texts, floatText{text: s, x: x, y: y, life: 1.2, size: size, clr: clr})
}

func (e *effects) update(dt float64) {
	ps := e.particles[:0]
	for _, p := range e.particles {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.vy += p.gravity * dt
		p.x += p.vx * dt
		p.y += p.vy * dt
		ps = append(ps, p)
	}
	e.particles = ps

	ts := e.texts[:0]
	for _, t := range e.texts {
		t.life -= dt
		if t.life <= 0 {
			continue
		}
		t.y -= 50 * dt
		ts = append(ts, t)
	}
	e.texts = ts
}

func (e *effects) draw(dst *ebiten.Image) {
	for _, p := range e.particles {
		a := p.life / p.max
		if p.isStar {
			star(dst, float32(p.x), float32(p.y), float32(p.size), p.life*4, fade(p.clr, math.Min(1, a*2)))
			continue
		}
		fillCircle(dst, float32(p.x), float32(p.y), float32(p.size*(0.4+0.6*a)), fade(p.clr, a))
	}
	for _, t := range e.texts {
		a := math.Min(1, t.life*2)
		pop := 1 + math.Max(0, t.life-1.0)*2.5
		drawTextShadow(dst, t.text, face(true, int(float64(t.size)*pop)), t.x, t.y, fade(t.clr, a), alignCenter)
	}
}
