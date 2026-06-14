package main

import (
	"context"
	"math"
	"math/rand"
	"os"
	"os/signal"

	. "github.com/striter-no/stg/graphics"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	s, err := NewScreen(NewBGPixel(10, 10, 10, " "), ctx)
	if err != nil {
		panic(err)
	}

	s.EnterAlt()
	s.HideCursor()
	defer func() {
		s.ShowCursor()
		s.ExitAlt()
	}()

	p := NewPerlin(123321)

	var tick float64
	for s.IsRunning() {
		if err := s.ClearPixels(); err != nil {
			panic(err)
		}

		for y := range s.Height {
			for x := range s.Width {
				v := p.Noise2DNormalized(
					float64(x)/float64(s.Width)+tick/100,
					float64(y)/float64(s.Height)+tick/100,
				)

				s.SetPixel(x, y, NewBGPixel(uint(v*255), uint(v*255), uint(v*255), " "))
			}
		}

		s.SetText(0, 0, "Just some text to show", NewFGPixel(255, 0, 255, ""))

		s.Blit()
		// time.Sleep(time.Millisecond * 100)
		tick++
	}
}

// -- just for an example

type Perlin struct {
	permutation [512]int
}

func NewPerlin(seed int64) *Perlin {
	p := &Perlin{}

	r := rand.New(rand.NewSource(seed))
	perm := make([]int, 256)
	for i := range perm {
		perm[i] = i
	}

	for i := 255; i > 0; i-- {
		j := r.Intn(i + 1)
		perm[i], perm[j] = perm[j], perm[i]
	}

	for i := range p.permutation {
		p.permutation[i] = perm[i&255]
	}

	return p
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(t, a, b float64) float64 {
	return a + t*(b-a)
}

func grad(hash int, x, y float64) float64 {
	h := hash & 7
	var u, v float64

	if h < 4 {
		u = x
	} else {
		u = y
	}

	if h < 4 {
		v = y
	} else {
		v = x
	}

	if h&1 == 0 {
		return u + v
	} else {
		return -u + v
	}
}

// 2D Perlin noise, output value: [-1, 1]
func (p *Perlin) Noise2D(x, y float64) float64 {
	xi := int(math.Floor(x)) & 255
	yi := int(math.Floor(y)) & 255

	xf := x - math.Floor(x)
	yf := y - math.Floor(y)

	u := fade(xf)
	v := fade(yf)

	aa := p.permutation[p.permutation[xi]+yi]
	ab := p.permutation[p.permutation[xi]+yi+1]
	ba := p.permutation[p.permutation[xi+1]+yi]
	bb := p.permutation[p.permutation[xi+1]+yi+1]

	x1 := lerp(u, grad(aa, xf, yf), grad(ba, xf-1, yf))
	x2 := lerp(u, grad(ab, xf, yf-1), grad(bb, xf-1, yf-1))

	return lerp(v, x1, x2)
}

// 2D Perlin noise, output value: [0, 1]
func (p *Perlin) Noise2DNormalized(x, y float64) float64 {
	return (p.Noise2D(x, y) + 1) / 2
}
