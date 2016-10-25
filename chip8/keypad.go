package chip8

import (
	"errors"
	"fmt"

	"github.com/nsf/termbox-go"
)

type Keypad interface {
	GetKey() (byte, error)
}

type KeypadFunc func() (byte, error)

func (f KeypadFunc) GetKey() (byte, error) {
	return f()
}

var NullKeypad = KeypadFunc(func() (byte, error) {
	return 0x00, errors.New("null keypad not usable")
})

type TermboxKeypad struct{}

func NewTermboxKeypad() *TermboxKeypad {
	return &TermboxKeypad{}
}

var keyMap = map[rune]byte{
	'1': 0x01, '2': 0x02, '3': 0x03, '4': 0x0C,
	'q': 0x04, 'w': 0x05, 'e': 0x06, 'r': 0x0D,
	'a': 0x07, 's': 0x08, 'd': 0x09, 'f': 0x0E,
	'z': 0x0A, 'x': 0x00, 'c': 0x0B, 'v': 0x0F,
}

var escapeKey = '0'

func (k *TermboxKeypad) GetKey() (byte, error) {
	event := termbox.PollEvent()

	if event.Ch == escapeKey {
		return 0x00, ErrQuit
	}
	key, ok := keyMap[event.Ch]
	if !ok {
		return 0x00, fmt.Errorf("unknown key: %v", event.Ch)

	}
	return key, nil
}
