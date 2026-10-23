package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gluekl/go_game/rpg"
)

// Фигурка бойца рисуется векторно на отдельном холсте лицом вправо,
// а затем выводится на экран с нужным масштабом, отражением и поворотом.
const (
	spriteW     = 200
	spriteH     = 260
	spriteFootX = 100
	spriteFootY = 244
)

var (
	colSkin    = rgb(0xf1, 0xc2, 0x8b)
	colSteel   = rgb(0xb8, 0xc2, 0xcc)
	colDark    = rgb(0x2a, 0x22, 0x1e)
	colLeather = rgb(0x7a, 0x4e, 0x2a)
)

type sprite struct {
	img *ebiten.Image
}

func newSprite() *sprite {
	return &sprite{img: ebiten.NewImage(spriteW, spriteH)}
}

// render перерисовывает фигурку. t — время для анимации дыхания,
// pose — степень замаха/натяжения от 0 до 1.
func (s *sprite) render(c rpg.Class, t, pose float64) {
	dst := s.img
	dst.Clear()
	bob := float32(math.Sin(t*2.6) * 2.5)
	p := float32(pose)

	switch c {
	case rpg.Warrior:
		drawWarrior(dst, bob, p)
	case rpg.Mage:
		drawMage(dst, bob, p, t)
	case rpg.Archer:
		drawArcher(dst, bob, p)
	}
}

type spriteDraw struct {
	x, y    float64 // точка между ступнями на экране
	scale   float64
	flip    bool
	alpha   float64
	flash   float64 // белая вспышка при попадании
	angle   float64 // поворот вокруг ступней (падение)
	tint    color.RGBA
	useTint bool
}

func (s *sprite) draw(dst *ebiten.Image, d spriteDraw) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-spriteFootX, -spriteFootY)
	sx := d.scale
	if d.flip {
		sx = -sx
	}
	op.GeoM.Scale(sx, d.scale)
	op.GeoM.Rotate(d.angle)
	op.GeoM.Translate(d.x, d.y)
	if d.useTint {
		op.ColorScale.ScaleWithColor(d.tint)
	}
	op.ColorScale.ScaleAlpha(float32(d.alpha))
	dst.DrawImage(s.img, op)
	if d.flash > 0 {
		op.Blend = ebiten.BlendLighter
		op.ColorScale.Reset()
		op.ColorScale.Scale(1, 0.9, 0.8, 1)
		op.ColorScale.ScaleAlpha(float32(d.flash * 0.8 * d.alpha))
		dst.DrawImage(s.img, op)
	}
}

// handPos возвращает положение руки с оружием относительно ступней (в координатах холста).
func handPos(c rpg.Class, pose float64) (float64, float64) {
	switch c {
	case rpg.Warrior:
		return 128 - spriteFootX + 8*pose, 168 - spriteFootY - 20*pose
	case rpg.Mage:
		return 134 - spriteFootX + 4*pose, 98 - spriteFootY - 10*pose
	default:
		return 140 - spriteFootX, 150 - spriteFootY
	}
}

func shadowAndLegs(dst *ebiten.Image, bob float32, pants, boots color.RGBA) {
	fillEllipse(dst, 100, spriteFootY, 44, 9, color.RGBA{0, 0, 0, 0x60})
	line(dst, 92, 186+bob, 86, spriteFootY-6, 11, pants)
	line(dst, 108, 186+bob, 116, spriteFootY-6, 11, pants)
	fillRoundRect(dst, 76, spriteFootY-12, 20, 11, 4, boots)
	fillRoundRect(dst, 108, spriteFootY-12, 22, 11, 4, boots)
}

func head(dst *ebiten.Image, bob float32) {
	fillCircle(dst, 100, 108+bob, 17, colSkin)
	fillCircle(dst, 108, 106+bob, 2.5, colDark)
}

func drawWarrior(dst *ebiten.Image, bob, pose float32) {
	red := rgb(0xb0, 0x3a, 0x2e)
	shadowAndLegs(dst, bob, rgb(0x4a, 0x3a, 0x30), colDark)
	// Щит за спиной
	fillCircle(dst, 76, 160+bob, 22, rgb(0x8a, 0x24, 0x1c))
	strokeArc(dst, 76, 160+bob, 22, 0, 2*math.Pi, 4, colGold)
	fillCircle(dst, 76, 160+bob, 5, colGold)
	// Тело в доспехах
	fillRoundRect(dst, 80, 128+bob, 40, 64, 10, colSteel)
	fillRoundRect(dst, 84, 134+bob, 32, 50, 8, red)
	fillRoundRect(dst, 80, 184+bob, 40, 8, 3, colLeather)
	fillCircle(dst, 86, 134+bob, 11, colSteel)
	fillCircle(dst, 114, 134+bob, 11, colSteel)
	head(dst, bob)
	// Шлем с плюмажем
	strokeArc(dst, 100, 106+bob, 18, math.Pi*1.02, math.Pi*1.98, 9, colSteel)
	fillRoundRect(dst, 81, 102+bob, 38, 6, 2, colSteel)
	fillRoundRect(dst, 98, 104+bob, 4, 14, 2, colSteel)
	for i := 0; i < 5; i++ {
		fillCircle(dst, 96-float32(i)*6, 84+bob-float32(i)*2+float32(i*i)*1.2, 6, rgb(0xd8, 0x3b, 0x2c))
	}
	// Рука и меч
	hx, hy := 128+8*pose, 168+bob-20*pose
	line(dst, 112, 140+bob, hx, hy, 10, colSteel)
	angle := float64(-1.25 + 1.45*pose)
	dx, dy := float32(math.Cos(angle)), float32(math.Sin(angle))
	line(dst, hx-dx*12, hy-dy*12, hx, hy, 6, colLeather)
	line(dst, hx-dy*12, hy+dx*12, hx+dy*12, hy-dx*12, 6, colGold)
	line(dst, hx, hy, hx+dx*78, hy+dy*78, 7, colSteel)
	line(dst, hx+dx*4, hy+dy*4, hx+dx*72, hy+dy*72, 2, colWhite)
	fillCircle(dst, hx, hy, 6, colSkin)
}

func drawMage(dst *ebiten.Image, bob, pose float32, t float64) {
	robe := rgb(0x5b, 0x34, 0xb0)
	robeDark := rgb(0x3f, 0x22, 0x82)
	fillEllipse(dst, 100, spriteFootY, 46, 9, color.RGBA{0, 0, 0, 0x60})
	// Мантия
	fillPoly(dst, robeDark, 86, 128+bob, 114, 128+bob, 134, spriteFootY-2, 66, spriteFootY-2)
	fillPoly(dst, robe, 88, 128+bob, 112, 128+bob, 124, spriteFootY-4, 76, spriteFootY-4)
	line(dst, 100, 136+bob, 100, spriteFootY-6, 3, colGold)
	fillRoundRect(dst, 84, 176+bob, 32, 7, 3, colGold)
	head(dst, bob)
	// Борода
	fillPoly(dst, rgb(0xe8, 0xe4, 0xdc), 92, 114+bob, 112, 114+bob, 106, 144+bob, 100, 150+bob)
	// Шляпа
	fillEllipse(dst, 100, 96+bob, 30, 7, robeDark)
	fillPoly(dst, robe, 82, 96+bob, 118, 96+bob, 110, 64+bob, 84, 38+bob)
	star(dst, 102, 78+bob, 6, 0, colGold)
	// Посох
	sx := 134 + 18*pose
	sy := 104 + bob - 10*pose
	tilt := 14 * pose
	line(dst, sx-tilt, sy, sx-tilt*2+2, spriteFootY-2, 6, colLeather)
	line(dst, 110, 140+bob, sx-tilt*0.7, sy+38, 9, robe)
	fillCircle(dst, sx-tilt*0.7, sy+38, 5, colSkin)
	glow := float32(1+0.15*math.Sin(t*5)) * (1 + pose*0.8)
	fillCircle(dst, sx-tilt, sy-6, 16*glow, fade(colPurple, 0.25))
	fillCircle(dst, sx-tilt, sy-6, 10*glow, fade(colPurple, 0.6))
	fillCircle(dst, sx-tilt, sy-6, 6, rgb(0xee, 0xe2, 0xff))
}

func drawArcher(dst *ebiten.Image, bob, pose float32) {
	green := rgb(0x2e, 0x8b, 0x57)
	hood := rgb(0x1f, 0x6b, 0x40)
	// Плащ
	fillPoly(dst, rgb(0x17, 0x4f, 0x30), 84, 124+bob, 100, 124+bob, 84, 214+bob, 62, 206+bob)
	// Колчан
	fillPoly(dst, colLeather, 70, 118+bob, 80, 114+bob, 92, 168+bob, 82, 172+bob)
	for i := float32(0); i < 3; i++ {
		line(dst, 72+i*4, 116+bob-i, 68+i*4, 102+bob-i, 3, rgb(0xe8, 0xe4, 0xdc))
	}
	shadowAndLegs(dst, bob, rgb(0x5a, 0x48, 0x34), colLeather)
	fillRoundRect(dst, 84, 128+bob, 32, 62, 9, green)
	fillRoundRect(dst, 84, 176+bob, 32, 7, 3, colLeather)
	head(dst, bob)
	// Капюшон
	strokeArc(dst, 100, 108+bob, 19, math.Pi*0.75, math.Pi*2.05, 9, hood)
	fillPoly(dst, hood, 84, 98+bob, 70, 120+bob, 86, 120+bob)
	// Лук
	bx, by := float32(140), 150+bob
	strokeArc(dst, bx-36, by, 44, -0.95, 0.95, 5, colLeather)
	topX, topY := bx-36+44*float32(math.Cos(-0.95)), by+44*float32(math.Sin(-0.95))
	botX, botY := topX, by-44*float32(math.Sin(-0.95))
	pull := bx - 10 - 26*pose
	line(dst, topX, topY, pull, by, 1.5, colWhite)
	line(dst, pull, by, botX, botY, 1.5, colWhite)
	// Руки
	line(dst, 112, 140+bob, bx, by, 9, green)
	fillCircle(dst, bx, by, 5, colSkin)
	line(dst, 104, 142+bob, pull, by, 9, green)
	fillCircle(dst, pull, by, 5, colSkin)
	if pose > 0.05 {
		line(dst, pull, by, bx+16, by, 3, rgb(0xd8, 0xc0, 0x90))
		fillPoly(dst, colSteel, bx+22, by, bx+14, by-4, bx+14, by+4)
	}
}
