package chip8

import "github.com/nsf/termbox-go"

const (
	GraphicsWidth  = 64 // Pixels
	GraphicsHeight = 32 // Pixels
)

type Display interface {
	Render(*Graphics) error
}

type DisplayFunc func(*Graphics) error

func (f DisplayFunc) Render(g *Graphics) error {
	return f(g)
}
var NullDisplay = DisplayFunc(func(*Graphics) error {
	return nil
})
type Graphics struct {
	Pixels [GraphicsWidth * GraphicsHeight]byte
	Display
}

func (g *Graphics) WriteSprite(sprite []byte, x, y byte) (collision bool) {
	n := len(sprite)

	for yl := 0; yl < n; yl++ {
		// A row of sprite data.
		r := sprite[yl]

		for xl := 0; xl < 8; xl++ {
			// This represents a mask for the bit that we
			// care about for this coordinate.
			i := 0x80 >> byte(xl)

			// Whether the bit is set or not.
			on := (r & byte(i)) == byte(i)

			// The X position for this pixel
			xp := uint16(x) + uint16(xl)
			if xp >= GraphicsWidth {
				xp = xp - GraphicsWidth
			}

			// The Y position for this pixel
			yp := uint16(y) + uint16(yl)
			if yp >= GraphicsHeight {
				yp = yp - GraphicsHeight
			}

			if g.Set(xp, yp, on) {
				collision = true
			}
		}
	}

	return
}

// Clear clears the display.
func (g *Graphics) Clear() {
	g.EachPixel(func(_, _ uint16, addr int) {
		g.Pixels[addr] = 0
	})
}

// Draw draws the graphics array to the Display.
func (g *Graphics) Draw() error {
	return g.display().Render(g)
}

// EachPixel yields each pixel in the graphics array to fn.
func (g *Graphics) EachPixel(fn func(x, y uint16, addr int)) {
	for y := 0; y < GraphicsHeight-1; y++ {
		for x := 0; x < GraphicsWidth-1; x++ {
			a := y*GraphicsWidth + x
			fn(uint16(x), uint16(y), a)
		}
	}
}

// Set turns the pixel at the given coordinates on or off. If there's a
// collision, it returns true.
func (g *Graphics) Set(x, y uint16, on bool) (collision bool) {
	a := x + y*GraphicsWidth

	if g.Pixels[a] == 0x01 {
		collision = true
	}

	var v byte
	if on {
		v = 0x01
	}

	g.Pixels[a] = g.Pixels[a] ^ v

	return
}

func (g *Graphics) display() Display {
	if g.Display == nil {
		return DefaultDisplay
	}

	return g.Display
}

// termboxInit initializes termbox with appropriate settings. This should be
// called before using the TermboxDisplay and TermboxKeypad.
func termboxInit(bg termbox.Attribute) error {
	if err := termbox.Init(); err != nil {
		return err
	}

	termbox.HideCursor()

	if err := termbox.Clear(bg, bg); err != nil {
		return err
	}

	return termbox.Flush()
}

// TermboxDisplay is an implementation of the Display interface that renders
// the graphics array to the terminal.
type TermboxDisplay struct {
	fg, bg termbox.Attribute
}

// NewTermboxDisplay returns a new TermboxDisplay instance.
func NewTermboxDisplay(fg, bg termbox.Attribute) (*TermboxDisplay, error) {
	return &TermboxDisplay{
		fg: fg,
		bg: bg,
	}, termboxInit(bg)
}

// Render renders the graphics array to the terminal using Termbox.
func (d *TermboxDisplay) Render(g *Graphics) error {
	g.EachPixel(func(x, y uint16, addr int) {
		v := ' '

		if g.Pixels[addr] == 0x01 {
			v = '█'
		}

		termbox.SetCell(
			int(x),
			int(y),
			v,
			d.fg,
			d.bg,
		)
	})

	return termbox.Flush()
}

func (d *TermboxDisplay) Close() {
	termbox.Close()
}
