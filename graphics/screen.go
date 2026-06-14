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

	bg_pixel   Pixel
	pixels     []Pixel
	textPixels []Pixel

	buf           strings.Builder
	cancelContext context.Context
}

func NewScreen(bg_pixel Pixel, cancelContext context.Context) (*Screen, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return nil, err
	}
	width -= 2
	height -= 1

	if height%2 != 0 {
		height--
	}

	termHeight := height
	logicalHeight := height * 2

	out := &Screen{
		bg_pixel:      bg_pixel,
		Width:         width,
		Height:        logicalHeight,
		cancelContext: cancelContext,
	}

	out.pixels = make([]Pixel, width*logicalHeight)
	for n := range out.pixels {
		out.pixels[n] = bg_pixel
	}

	out.textPixels = make([]Pixel, width*termHeight)
	for n := range out.textPixels {
		out.textPixels[n] = Pixel{Char: "", R: -1, FR: -1}
	}

	out.buf.Grow(width * termHeight * 40)

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

		if x < 0 || x >= s.Width || y < 0 || y >= s.Height/2 {
			continue
		}

		idx := y*s.Width + x
		s.textPixels[idx] = colorPix
		s.textPixels[idx].Char = string(c)
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

func (s *Screen) GetPixel(x, y int) *Pixel {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return nil
	}
	return &s.pixels[y*s.Width+x]
}

func (s *Screen) ClearPixels() error {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	width -= 2
	height -= 1

	if height%2 != 0 {
		height--
	}

	s.Width = width
	s.Height = height * 2
	termHeight := height

	s.buf.Reset()

	s.pixels = make([]Pixel, s.Width*s.Height)
	for n := range s.pixels {
		s.pixels[n] = s.bg_pixel
	}

	s.textPixels = make([]Pixel, s.Width*termHeight)
	for n := range s.textPixels {
		s.textPixels[n] = Pixel{Char: "", R: -1, FR: -1}
	}

	return nil
}

func (s *Screen) Blit() {
	s.buf.Reset()
	var curFR, curFG, curFB = -1, -1, -1
	var curR, curG, curB = -1, -1, -1

	for y := 0; y < s.Height; y += 2 {
		for x := 0; x < s.Width; x++ {
			pxTop := &s.pixels[y*s.Width+x]
			pxBot := &s.pixels[(y+1)*s.Width+x]

			topFR, topFG, topFB := pxTop.FR, pxTop.FG, pxTop.FB
			if topFR == -1 && pxTop.R != -1 {
				topFR, topFG, topFB = pxTop.R, pxTop.G, pxTop.B
			}

			botR, botG, botB := pxBot.R, pxBot.G, pxBot.B
			if botR == -1 && pxBot.FR != -1 {
				botR, botG, botB = pxBot.FR, pxBot.FG, pxBot.FB
			}

			newFR, newFG, newFB := topFR, topFG, topFB
			newR, newG, newB := botR, botG, botB

			if newFR != curFR || newFG != curFG || newFB != curFB ||
				newR != curR || newG != curG || newB != curB {

				s.buf.WriteString("\033[0m")

				hasFG := newFR != -1
				hasBG := newR != -1

				if hasFG && hasBG {
					fmt.Fprintf(&s.buf, "\033[38;2;%d;%d;%d;48;2;%d;%d;%dm",
						newFR, newFG, newFB, newR, newG, newB)
				} else if hasFG {
					fmt.Fprintf(&s.buf, "\033[38;2;%d;%d;%dm", newFR, newFG, newFB)
				} else if hasBG {
					fmt.Fprintf(&s.buf, "\033[48;2;%d;%d;%dm", newR, newG, newB)
				}

				curFR, curFG, curFB = newFR, newFG, newFB
				curR, curG, curB = newR, newG, newB
			}

			charToDraw := "▀"
			if pxTop.Char != " " && pxTop.Char != "" {
				charToDraw = pxTop.Char
			}
			s.buf.WriteString(charToDraw)
		}

		s.buf.WriteString("\033[0m\n")

		curFR, curFG, curFB = -1, -1, -1
		curR, curG, curB = -1, -1, -1
	}

	os.Stdout.WriteString(escClear + s.buf.String())
}
