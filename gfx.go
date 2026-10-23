package main

import (
	"bytes"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/gluekl/go_game/rpg"
)

const (
	screenW = 1280
	screenH = 720
	groundY = 470
)

var (
	colWhite   = rgb(0xf4, 0xf1, 0xe8)
	colMuted   = rgb(0xa8, 0xad, 0xc0)
	colGold    = rgb(0xf2, 0xc1, 0x4e)
	colRed     = rgb(0xe0, 0x4f, 0x4f)
	colOrange  = rgb(0xff, 0x8c, 0x2a)
	colPurple  = rgb(0xb4, 0x8c, 0xff)
	colPanel   = color.RGBA{0x10, 0x13, 0x22, 0xd8}
	colPanelHi = rgb(0x3a, 0x42, 0x66)
)

var (
	regularSrc *text.GoTextFaceSource
	boldSrc    *text.GoTextFaceSource
	faceCache  = map[[2]int]*text.GoTextFace{}
)

func init() {
	var err error
	if regularSrc, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF)); err != nil {
		log.Fatal(err)
	}
	if boldSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF)); err != nil {
		log.Fatal(err)
	}
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 0xff} }

// fade умножает цвет на прозрачность a (color.RGBA хранит premultiplied-значения).
func fade(c color.RGBA, a float64) color.RGBA {
	a = clamp(a, 0, 1)
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(float64(c.A) * a)}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = clamp(t, 0, 1)
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func lerp(a, b, t float64) float64 { return a + (b-a)*clamp(t, 0, 1) }

// ease — плавное ускорение и замедление на отрезке [0, 1].
func ease(t float64) float64 {
	t = clamp(t, 0, 1)
	return t * t * (3 - 2*t)
}

func face(bold bool, size int) *text.GoTextFace {
	b := 0
	if bold {
		b = 1
	}
	key := [2]int{b, size}
	if f, ok := faceCache[key]; ok {
		return f
	}
	src := regularSrc
	if bold {
		src = boldSrc
	}
	f := &text.GoTextFace{Source: src, Size: float64(size)}
	faceCache[key] = f
	return f
}

type align int

const (
	alignLeft align = iota
	alignCenter
	alignRight
)

// drawText рисует строку; y — верхняя граница текста.
func drawText(dst *ebiten.Image, s string, f *text.GoTextFace, x, y float64, clr color.Color, a align) {
	op := &text.DrawOptions{}
	switch a {
	case alignCenter:
		op.PrimaryAlign = text.AlignCenter
	case alignRight:
		op.PrimaryAlign = text.AlignEnd
	}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, f, op)
}

// drawTextShadow рисует текст с тенью для читаемости поверх сцены.
func drawTextShadow(dst *ebiten.Image, s string, f *text.GoTextFace, x, y float64, clr color.Color, a align) {
	off := math.Max(2, f.Size/14)
	drawText(dst, s, f, x+off, y+off, color.RGBA{0, 0, 0, 0xb0}, a)
	drawText(dst, s, f, x, y, clr, a)
}

func textWidth(s string, f *text.GoTextFace) float64 {
	w, _ := text.Measure(s, f, 0)
	return w
}

func fillPath(dst *ebiten.Image, p *vector.Path, clr color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, p, nil, op)
}

func strokePath(dst *ebiten.Image, p *vector.Path, width float32, clr color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.StrokePath(dst, p, &vector.StrokeOptions{Width: width, LineCap: vector.LineCapRound, LineJoin: vector.LineJoinRound}, op)
}

// fillPoly заливает многоугольник, заданный парами координат x, y.
func fillPoly(dst *ebiten.Image, clr color.Color, pts ...float32) {
	var p vector.Path
	p.MoveTo(pts[0], pts[1])
	for i := 2; i+1 < len(pts); i += 2 {
		p.LineTo(pts[i], pts[i+1])
	}
	p.Close()
	fillPath(dst, &p, clr)
}

func fillEllipse(dst *ebiten.Image, cx, cy, rx, ry float32, clr color.Color) {
	const n = 32
	pts := make([]float32, 0, n*2)
	for i := 0; i < n; i++ {
		a := float64(i) / n * 2 * math.Pi
		pts = append(pts, cx+rx*float32(math.Cos(a)), cy+ry*float32(math.Sin(a)))
	}
	fillPoly(dst, clr, pts...)
}

func fillCircle(dst *ebiten.Image, cx, cy, r float32, clr color.Color) {
	vector.FillCircle(dst, cx, cy, r, clr, true)
}

func line(dst *ebiten.Image, x0, y0, x1, y1, w float32, clr color.Color) {
	var p vector.Path
	p.MoveTo(x0, y0)
	p.LineTo(x1, y1)
	strokePath(dst, &p, w, clr)
}

func roundRectPath(x, y, w, h, r float32) *vector.Path {
	var p vector.Path
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(x+w, y, x+w, y+r, r)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(x+w, y+h, x+w-r, y+h, r)
	p.LineTo(x+r, y+h)
	p.ArcTo(x, y+h, x, y+h-r, r)
	p.LineTo(x, y+r)
	p.ArcTo(x, y, x+r, y, r)
	p.Close()
	return &p
}

func fillRoundRect(dst *ebiten.Image, x, y, w, h, r float32, clr color.Color) {
	fillPath(dst, roundRectPath(x, y, w, h, r), clr)
}

func strokeRoundRect(dst *ebiten.Image, x, y, w, h, r, width float32, clr color.Color) {
	strokePath(dst, roundRectPath(x, y, w, h, r), width, clr)
}

func strokeArc(dst *ebiten.Image, cx, cy, r, a0, a1, w float32, clr color.Color) {
	var p vector.Path
	p.Arc(cx, cy, r, a0, a1, vector.Clockwise)
	strokePath(dst, &p, w, clr)
}

// star рисует пятиконечную звезду.
func star(dst *ebiten.Image, cx, cy, r float32, rot float64, clr color.Color) {
	pts := make([]float32, 0, 20)
	for i := 0; i < 10; i++ {
		rr := r
		if i%2 == 1 {
			rr = r * 0.45
		}
		a := rot + float64(i)*math.Pi/5 - math.Pi/2
		pts = append(pts, cx+rr*float32(math.Cos(a)), cy+rr*float32(math.Sin(a)))
	}
	fillPoly(dst, clr, pts...)
}

// flame рисует язычок пламени с основанием в (cx, cy).
func flame(dst *ebiten.Image, cx, cy, h float32, t float64) {
	sway := float32(math.Sin(t*9)) * h * 0.08
	fillPoly(dst, colOrange,
		cx-h*0.32, cy, cx-h*0.38, cy-h*0.35, cx-h*0.12, cy-h*0.62, cx+sway, cy-h,
		cx+h*0.2, cy-h*0.6, cx+h*0.38, cy-h*0.35, cx+h*0.32, cy)
	fillPoly(dst, colGold,
		cx-h*0.16, cy, cx-h*0.2, cy-h*0.28, cx+sway*0.6, cy-h*0.62, cx+h*0.2, cy-h*0.28, cx+h*0.16, cy)
}

func classColor(c rpg.Class) color.RGBA {
	switch c {
	case rpg.Warrior:
		return rgb(0xe0, 0x5a, 0x47)
	case rpg.Mage:
		return rgb(0x9b, 0x6b, 0xf2)
	default:
		return rgb(0x4c, 0xc2, 0x7c)
	}
}

// background рисует ночную арену один раз и кеширует её.
var bgImage *ebiten.Image

func background() *ebiten.Image {
	if bgImage != nil {
		return bgImage
	}
	img := ebiten.NewImage(screenW, screenH)
	top, horizon := rgb(0x0e, 0x12, 0x2b), rgb(0x45, 0x3b, 0x6e)
	for y := 0; y < groundY; y += 4 {
		vector.FillRect(img, 0, float32(y), screenW, 4, mix(top, horizon, float64(y)/groundY), false)
	}
	// Звёзды
	for i := 0; i < 90; i++ {
		x := float32((i*197 + 31) % screenW)
		y := float32((i*83 + 17) % 300)
		r := float32(0.6 + float64(i%3)*0.5)
		fillCircle(img, x, y, r, fade(colWhite, 0.35+float64(i%4)*0.15))
	}
	// Луна
	fillCircle(img, 1040, 110, 70, fade(rgb(0xff, 0xf0, 0xc8), 0.08))
	fillCircle(img, 1040, 110, 46, rgb(0xf3, 0xe9, 0xc6))
	fillCircle(img, 1024, 98, 9, rgb(0xdd, 0xd0, 0xaa))
	fillCircle(img, 1056, 126, 6, rgb(0xdd, 0xd0, 0xaa))
	// Горы
	mountains := func(clr color.RGBA, base float32, peaks []float32) {
		pts := []float32{0, groundY}
		step := float32(screenW) / float32(len(peaks)-1)
		for i, h := range peaks {
			pts = append(pts, float32(i)*step, base-h)
		}
		pts = append(pts, screenW, groundY)
		fillPoly(img, clr, pts...)
	}
	mountains(rgb(0x2b, 0x2a, 0x4f), groundY, []float32{120, 210, 150, 260, 170, 230, 130, 250, 190, 140})
	mountains(rgb(0x1f, 0x22, 0x3d), groundY, []float32{60, 110, 80, 150, 90, 70, 130, 100, 60, 90, 120, 70})
	// Стены арены
	for x := float32(0); x < screenW; x += 64 {
		fillRoundRect(img, x+2, groundY-46, 60, 46, 4, rgb(0x3a, 0x33, 0x40))
		fillRoundRect(img, x+6, groundY-42, 52, 8, 3, rgb(0x4a, 0x42, 0x50))
	}
	// Земля
	vector.FillRect(img, 0, groundY, screenW, screenH-groundY, rgb(0x5a, 0x45, 0x30), false)
	for y := float32(groundY); y < screenH; y += 6 {
		vector.FillRect(img, 0, y, screenW, 3, mix(rgb(0x6e, 0x55, 0x3a), rgb(0x3a, 0x2c, 0x20), float64(y-groundY)/(screenH-groundY)), false)
	}
	for i := 0; i < 60; i++ {
		x := float32((i*251 + 40) % screenW)
		y := float32(groundY + 10 + (i*37)%60)
		fillEllipse(img, x, y, float32(3+i%4), 2, rgb(0x7a, 0x62, 0x48))
	}
	bgImage = img
	return img
}

// drawTorches рисует анимированные факелы по краям арены.
func drawTorches(dst *ebiten.Image, t float64) {
	for _, x := range []float32{90, screenW - 90} {
		fillCircle(dst, x, groundY-150, 70+float32(math.Sin(t*7+float64(x)))*4, fade(colOrange, 0.10))
		vector.FillRect(dst, x-5, groundY-140, 10, 140, rgb(0x4a, 0x35, 0x22), true)
		fillRoundRect(dst, x-14, groundY-148, 28, 14, 4, rgb(0x6b, 0x6b, 0x75))
		flame(dst, x, groundY-146, 44, t+float64(x))
	}
}

func drawPanel(dst *ebiten.Image, x, y, w, h float32) {
	fillRoundRect(dst, x, y, w, h, 12, colPanel)
	strokeRoundRect(dst, x, y, w, h, 12, 2, colPanelHi)
}

type button struct {
	x, y, w, h float64
	label      string
}

func (b *button) hovered() bool {
	mx, my := ebiten.CursorPosition()
	return float64(mx) >= b.x && float64(mx) <= b.x+b.w && float64(my) >= b.y && float64(my) <= b.y+b.h
}

func (b *button) clicked() bool {
	return b.hovered() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}

func (b *button) draw(dst *ebiten.Image) {
	base := rgb(0xc8, 0x8a, 0x2a)
	if b.hovered() {
		base = rgb(0xe6, 0xa5, 0x3a)
	}
	x, y, w, h := float32(b.x), float32(b.y), float32(b.w), float32(b.h)
	fillRoundRect(dst, x, y+4, w, h, 10, rgb(0x5e, 0x3d, 0x10))
	fillRoundRect(dst, x, y, w, h, 10, base)
	fillRoundRect(dst, x+4, y+4, w-8, h/2-4, 6, fade(colWhite, 0.15))
	f := face(true, int(b.h*0.45))
	drawTextShadow(dst, b.label, f, b.x+b.w/2, b.y+(b.h-f.Size*1.2)/2, colWhite, alignCenter)
}
