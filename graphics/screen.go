package graphics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	escClear = "\033[H"
	escBgRGB = "\033[48;2;%d;%d;%dm"
)

const (
	escAltScreenOn  = "\033[?1049h"
	escAltScreenOff = "\033[?1049l"
	escHideCursor   = "\033[?25l"
	escShowCursor   = "\033[?25h"
)

type Pixel struct {
	R, G, B    int
	FR, FG, FB int
	Char       string
}

func NewBGPixel(R, G, B uint, Char string) Pixel {
	return Pixel{
		R:    int(R),
		G:    int(G),
		B:    int(B),
		FR:   -1,
		FG:   -1,
		FB:   -1,
		Char: Char,
	}
}

func NewFGPixel(R, G, B uint, Char string) Pixel {
	return Pixel{
		FR:   int(R),
		FG:   int(G),
		FB:   int(B),
		R:    -1,
		G:    -1,
		B:    -1,
		Char: Char,
	}
}

func NewPixel(R, G, B, FR, FG, FB uint, Char string) Pixel {
	return Pixel{
		FR:   int(FR),
		FG:   int(FG),
		FB:   int(FB),
		R:    int(R),
		G:    int(G),
		B:    int(B),
		Char: Char,
	}
}

// ----------------------

type Screen struct {
	Width  int
	Height int

	bg_pixel Pixel
	pixels   []Pixel

	buf           strings.Builder
	cancelContext context.Context
}

func NewScreen(bg_pixel Pixel, cancelContext context.Context) (Screen, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return Screen{}, err
	}

	width -= 2
	height -= 1

	out := Screen{
		bg_pixel:      bg_pixel,
		Width:         width,
		Height:        height,
		cancelContext: cancelContext,
	}

	out.buf.Grow(width*height*(len(escBgRGB)+3)*2 + height*2)
	out.pixels = make([]Pixel, width*height)

	for n := range out.pixels {
		out.pixels[n] = bg_pixel
	}

	return out, nil
}

func (s *Screen) HideCursor() {
	os.Stdout.WriteString(escHideCursor)
}

func (s *Screen) ShowCursor() {
	os.Stdout.WriteString(escShowCursor)
}

func (s *Screen) EnterAlt() {
	os.Stdout.WriteString(escAltScreenOn)
}

func (s *Screen) ExitAlt() {
	os.Stdout.WriteString(escAltScreenOff)
}

func (s *Screen) SetText(tx, ty int, text string, colorPix Pixel) {
	y_offset := 0
	x_offset := 0

	for _, c := range text {
		if c == '\n' {
			y_offset++
			x_offset = 0
			continue
		}

		x := tx + x_offset
		y := ty + y_offset
		x_offset++

		if x < 0 || x >= s.Width || y < 0 || y >= s.Height {
			continue
		}

		if colorPix.R != -1 {
			s.pixels[y*s.Width+x].R = colorPix.R
			s.pixels[y*s.Width+x].G = colorPix.G
			s.pixels[y*s.Width+x].B = colorPix.B
		} else if colorPix.FR != -1 {
			s.pixels[y*s.Width+x].FR = colorPix.FR
			s.pixels[y*s.Width+x].FG = colorPix.FG
			s.pixels[y*s.Width+x].FB = colorPix.FB
		}
		s.pixels[y*s.Width+x].Char = string(c)
	}
}

func (s *Screen) IsRunning() bool {
	select {
	case <-s.cancelContext.Done():
		return false
	default:
		return true
	}
}

func (s *Screen) SetPixel(x, y int, pix Pixel) error {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return errors.New("Pixel is out of bounds")
	}

	s.pixels[y*s.Width+x] = pix
	return nil
}

func (s *Screen) ClearPixels() error {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}

	width -= 2
	height -= 1

	s.buf.Reset()
	s.pixels = make([]Pixel, width*height)

	for n := range s.pixels {
		s.pixels[n] = s.bg_pixel
	}

	s.Width = width
	s.Height = height
	return nil
}

func (s *Screen) Blit() {
	s.buf.Reset()

	var curFR, curFG, curFB = -1, -1, -1
	var curR, curG, curB = -1, -1, -1

	for y := range s.Height {
		for x := range s.Width {
			px := &s.pixels[y*s.Width+x]

			if px.FR != curFR || px.FG != curFG || px.FB != curFB ||
				px.R != curR || px.G != curG || px.B != curB {

				s.buf.WriteString("\033[0m")

				hasFG := px.FR != -1
				hasBG := px.R != -1

				if hasFG && hasBG {
					fmt.Fprintf(&s.buf, "\033[38;2;%d;%d;%d;48;2;%d;%d;%dm",
						px.FR, px.FG, px.FB, px.R, px.G, px.B)
				} else if hasFG {
					fmt.Fprintf(&s.buf, "\033[38;2;%d;%d;%dm", px.FR, px.FG, px.FB)
				} else if hasBG {
					fmt.Fprintf(&s.buf, "\033[48;2;%d;%d;%dm", px.R, px.G, px.B)
				}

				curFR, curFG, curFB = px.FR, px.FG, px.FB
				curR, curG, curB = px.R, px.G, px.B
			}

			s.buf.WriteString(px.Char)
		}

		s.buf.WriteString("\033[0m\n")

		curFR, curFG, curFB = -1, -1, -1
		curR, curG, curB = -1, -1, -1
	}

	os.Stdout.WriteString(escClear + s.buf.String())
}
