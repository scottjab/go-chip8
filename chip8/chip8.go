package chip8

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"
)

var FONT = [80]byte{
	// Fontz.
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

var (
	// DefaultKeypad is the default Keypad to use for input. The default is
	// to always return 0x01.
	DefaultKeypad Keypad = NullKeypad

	// DefaultDisplay is the default Display to render graphics data to.
	DefaultDisplay Display = NullDisplay

	// DefaultClockSpeed is the default clock speed of the CPU. The CHIP-8
	// operated at 60 Hz.
	DefaultClockSpeed = time.Duration(60) // Hz

	// DefaultOptions is the default set of options that's used when calling
	// NewCPU.
	DefaultOptions = &Options{
		ClockSpeed: DefaultClockSpeed,
	}
	ErrQuit = errors.New("chip8: shutting down")
)

type Options struct {
	ClockSpeed time.Duration
}

type CPU struct {
	// 4096 bytes of memory.
	Memory [4096]byte

	// Registers
	V [16]byte

	// Index Register
	I uint16

	// Program counter
	ProgramCounter uint16

	// Stack
	Stack        [16]uint16
	StackPointer byte

	// The graphics map
	Graphics

	// Timers
	DelayTimer byte
	SoundTimer byte

	// Key
	key [16]byte

	// Keypad
	Keypad Keypad

	Clock <-chan time.Time
	stop  chan struct{}
}

func NewCPU(options *Options) *CPU {
	cpu := &CPU{
		ProgramCounter: 0x200,
		Clock:          time.Tick(time.Second / options.ClockSpeed),
		stop:           make(chan struct{}),
	}
	cpu.ProgramCounter = 0x200
	for i := 0; i < 80; i++ {
		cpu.Memory[i] = FONT[i]
	}
	return cpu
}

func (c *CPU) Load(r io.Reader) (int, error) {
	return c.load(0x200, r)
}

func (c *CPU) LoadBytes(b []byte) (int, error) {
	return c.Load(bytes.NewReader(b))

}

func (c *CPU) load(offset int, r io.Reader) (int, error) {
	return r.Read(c.Memory[offset:])
}

func (c *CPU) decodeOp() uint16 {
	return uint16(c.Memory[c.ProgramCounter])<<8 | uint16(c.Memory[c.ProgramCounter+1])
}

func (c *CPU) dispatch(opcode uint16) error {
	switch opcode & 0xF000 {
	// 0nn - SYS addr
	case 0x0000:
		switch opcode {
		case 0x00E0:
			c.Graphics.Clear()
			c.ProgramCounter += 2
			break
		case 0x00EE:
			// Return from subroutine.
			// Set the program counter to
			// Address at the top of stack, then subtract
			// one from the stack pointer.

			c.ProgramCounter = c.Stack[c.StackPointer]
			c.StackPointer--

			c.ProgramCounter += 2
			break
		default:

			return &UnknownOpcode{Opcode: opcode}
		}
	case 0x1000:
		/// JUMP to location nnn
		c.ProgramCounter = opcode & 0x0FFF
		break
	case 0x2000:
		// CALL subroutine at nnn
		c.StackPointer++
		c.Stack[c.StackPointer] = c.ProgramCounter
		c.ProgramCounter = opcode & 0x0FFF
		break
	case 0x3000:
		// 3XNN Skips the next instruction if VX equals NN.
		reg := (opcode & 0x0F00) >> 8
		nn := byte(opcode)
		c.ProgramCounter += 2
		if c.V[reg] == nn {
			c.ProgramCounter += 2
		}
		break
	case 0x4000:
		// 4XNN Skips the next instruction if VX doesn't equal NN.
		reg := (opcode & 0x0F00) >> 8
		nn := byte(opcode)
		c.ProgramCounter += 2
		if c.V[reg] != nn {
			c.ProgramCounter += 2
		}
		break
	case 0x5000:
		// 5XY0 Skips the next instruction if VX equals VY.
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 8
		c.ProgramCounter += 2
		if c.V[x] == c.V[y] {
			c.ProgramCounter += 2
		}
		break
	case 0x6000:
		// 6XNN	Sets VX to NN.
		x := (opcode & 0x0F00) >> 8
		nn := byte(opcode)
		c.ProgramCounter += 2
		c.V[x] = nn
		break
	case 0x7000:
		//7XNN	Adds NN to VX.
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode)
		c.V[x] = c.V[x] + kk
		c.ProgramCounter += 2
		break
	case 0x8000:
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4

		switch opcode & 0x000F {
		case 0x0000:
			// 8XY0	Sets VX to the value of VY.
			c.V[x] = c.V[y]
			c.ProgramCounter += 2
			break
		case 0x0001:
			// 8XY1	Sets VX to VX or VY.
			c.V[x] = c.V[y] | c.V[x]
			c.ProgramCounter += 2
			break
		case 0x0002:
			// 8XY2	Sets VX to VX and VY.
			c.V[x] = c.V[y] & c.V[x]
			c.ProgramCounter += 2
			break
		case 0x0003:
			// 8XY3	Sets VX to VX xor VY.
			c.V[x] = c.V[y] ^ c.V[x]
			c.ProgramCounter += 2
			break
		case 0x0004:
			// 8XY4	Adds VY to VX.
			// VF is set to 1 when there's a carry,
			// and to 0 when there isn't.
			result := uint16(c.V[x]) + uint16(c.V[y])

			var cf byte
			if result > 0xFF {
				cf = 1
			}
			c.V[0xF] = cf
			c.V[x] = byte(result)
			c.ProgramCounter += 2
			break
		case 0x0005:
			// 0XY5 VY is subtracted from VX.
			// VF is set to 0 when there's a borrow,
			// and 1 when there isn't.

			var cf byte
			if c.V[x] > c.V[y] {
				cf = 1
			}
			c.V[0xF] = cf

			c.V[x] = c.V[x] - c.V[y]
			c.ProgramCounter += 2
			break
		case 0x0006:
			// 8XY6	Shifts VX right by one.
			// VF is set to the value of the least significant
			// bit of VX before the shift.
			var cf byte
			if (c.V[x] & 0x01) == 0x01 {
				cf = 1
			}
			c.V[0xF] = cf
			c.V[x] = c.V[x] / 2
			c.ProgramCounter += 2
			break
		case 0x0007:
			// 8XY7	Sets VX to VY minus VX.
			// VF is set to 0 when there's a borrow,
			// and 1 when there isn't.
			var cf byte
			if c.V[y] > c.V[x] {
				cf = 1
			}
			c.V[0xF] = cf
			c.V[x] = c.V[y] - c.V[x]

			c.ProgramCounter += 2
			break
		case 0x000E:
			// 8XYE	Shifts VX left by one.
			// VF is set to the value of the most significant
			// bit of VX before the shift.
			var cf byte
			if (c.V[x] & 0x80) == 0x80 {
				cf = 1
			}
			c.V[0xF] = cf
			c.V[x] = c.V[x] * 2
			c.ProgramCounter += 2
			break
		}
		break
	case 0x9000:
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4
		switch opcode & 0x000F {
		// 9XY0 - SNE Vx, Vy
		case 0x0000:
			// Skip next instruction if Vx != Vy.
			//
			// The values of Vx and Vy are compared, and if they are
			// not equal, the program counter is increased by 2.

			c.ProgramCounter += 2
			if c.V[x] != c.V[y] {
				c.ProgramCounter += 2
			}

			break
		default:
			return &UnknownOpcode{Opcode: opcode}
		}

		break

	case 0xA000:
		// ANNN: Sets I to the address NNN
		c.I = opcode & 0x0FFF
		c.ProgramCounter += 2
		break
	case 0xB000:
		// BNNN	Jumps to the address NNN plus V0.
		c.ProgramCounter = opcode&0x0FFF + uint16(c.V[0])
		break
	case 0xC000:
		// CXNN	Sets VX to the result of a bitwise and operation on a random number and NN.
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode)
		c.V[x] = kk + byte(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(255))

		c.ProgramCounter += 2
		break
	case 0xD000:
		// DXYN	Draws a sprite at coordinate (VX, VY) that has a width of 8 pixels
		// and a height of N pixels. Each row of 8 pixels is read as bit-coded starting
		// from memory location I; I value doesn’t change after the execution of this instruction.
		// As described above, VF is set to 1 if any screen pixels are flipped from set to unset when
		// the sprite is drawn, and to 0 if that doesn’t happen

		var cf byte
		x := c.V[(opcode&0x0F00)>>8]
		y := c.V[(opcode&0x00F0)>>4]
		n := opcode & 0x000F

		if c.Graphics.WriteSprite(c.Memory[c.I:c.I+n], x, y) {
			cf = 0x01
		}

		c.V[0xF] = cf
		c.ProgramCounter += 2
		c.Graphics.Draw()
		break
	case 0xE000:
		x := (opcode & 0x0F00) >> 8
		switch opcode & 0x00FF {
		case 0x9E:
			// EX9E	Skips the next instruction if the key stored in VX is pressed.
			c.ProgramCounter += 2

			b, err := c.getKey()
			if err != nil {
				return err
			}

			if c.V[x] == b {
				c.ProgramCounter += 2
			}
			break
		case 0xA1:
			// EXA1	Skips the next instruction if the key stored in VX isn't pressed.
			c.ProgramCounter += 2
			b, err := c.getKey()
			if err != nil {
				return err
			}
			if c.V[x] != b {
				c.ProgramCounter += 2
			}
			break
		default:
			return &UnknownOpcode{Opcode: opcode}
		}
	case 0xF000:
		x := (opcode & 0x0F00) >> 8
		switch opcode & 0x00FF {
		case 0x07:
			// FX07	Sets VX to the value of the delay timer.
			c.V[x] = c.DelayTimer
			c.ProgramCounter += 2
			break
		case 0x0A:
			// FX0A	A key press is awaited, and then stored in VX.
			b, err := c.getKey()
			if err != nil {
				return err
			}

			c.V[x] = b
			c.ProgramCounter += 2
			break
		case 0x15:
			// FX15	Sets the delay timer to VX.
			c.DelayTimer = c.V[x]
			c.ProgramCounter += 2
			break
		case 0x18:
			// FX16 Sets the sound timer to the value of Vx
			c.SoundTimer = c.V[x]
			c.ProgramCounter += 2
			break
		case 0x1E:
			// FX1E	Adds VX to I.
			c.I = c.I + uint16(c.V[x])
			c.ProgramCounter += 2
			break
		case 0x29:
			// FX29	 Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font.
			c.I = uint16(c.V[x]) * uint16(0x05)
			c.ProgramCounter += 2
			break
		case 0x33:
			// FX33	Stores the binary-coded decimal representation of VX,
			// with the most significant of three digits at the address in I,
			// the middle digit at I plus 1,
			// and the least significant digit at I plus 2. (In other words,
			// take the decimal representation of VX, place the hundreds digit in memory at location in I,
			// the tens digit at location I+1, and the ones digit at location I+2.)
			c.Memory[c.I] = c.V[x] / 100
			c.Memory[c.I+1] = (c.V[x] / 10) % 10
			c.Memory[c.I+2] = (c.V[x] % 100) % 10
			c.ProgramCounter += 2
			break
		case 0x55:
			//FX55	Stores V0 to VX (including VX) in memory starting at address I.[4]
			for i := 0; uint16(i) <= x; i++ {
				c.Memory[c.I+uint16(i)] = c.V[i]
			}
			c.ProgramCounter += 2
			break
		case 0x65:
			// Fills V0 to VX (including VX) with values from memory starting at address I.[4]
			for i := 0; byte(i) <= byte(x); i++ {
				c.V[uint16(i)] = c.Memory[c.I+uint16(i)]
			}
			c.ProgramCounter += 2
			break
		default:
			return &UnknownOpcode{Opcode: opcode}
		}

	default:
		return &UnknownOpcode{Opcode: opcode}
	}
	return nil
}

func (c *CPU) emulateCycle() (uint16, error) {
	opcode := c.decodeOp()

	if err := c.dispatch(opcode); err != nil {
		return opcode, err
	}
	if c.DelayTimer > 0 {
		c.DelayTimer--
	}
	if c.SoundTimer > 0 {
		if c.SoundTimer == 1 {
			fmt.Print('\a')
		}
		c.SoundTimer--
	}
	return opcode, nil
}

func (c *CPU) Run() error {
	for {
		select {
		case <-c.stop:
			return nil
		case <-c.Clock:
			_, err := c.emulateCycle()
			if err != nil {
				if err == ErrQuit {
					return nil
				}
				return err
			}
			//log.Printf("op=0x%04X %s\n", op, c)
		}
	}
	return nil
}
func (c *CPU) Stop() {
	close(c.stop)
}
func (c *CPU) getKey() (byte, error) {
	b, err := c.keypad().GetKey()
	if err != nil {
		if err == ErrQuit {
			return b, err
		}

		return b, fmt.Errorf("chip8: unable to get key from keypad: %s", err.Error())
	}

	return b, nil
}

func (c *CPU) keypad() Keypad {
	if c.Keypad == nil {
		return DefaultKeypad
	}
	return c.Keypad
}

// UnknownOpcode is return when the opcode is not recognized.
type UnknownOpcode struct {
	Opcode uint16
}

func (e *UnknownOpcode) Error() string {
	return fmt.Sprintf("chip8: unknown opcode: 0x%04X", e.Opcode)
}
