# Simple Terminal Graphics

Simple project for bootstraping TUI projects

## Usage

```go
import (
	"context"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"time"

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
				v := p.Noise2DNormalized( // perlin code is in example/main.go
					float64(x)/float64(s.Width)+tick/100,
					float64(y)/float64(s.Height)+tick/100,
				)

				s.SetPixel(x, y, NewBGPixel(uint(v*255), uint(v*255), uint(v*255), " "))
			}
		}

		s.SetText(0, 0, "Just some text to show", NewFGPixel(255, 0, 255, ""))

		s.Blit()
		time.Sleep(time.Millisecond * 100)
		tick++
	}
}
```
