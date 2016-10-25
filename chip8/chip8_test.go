package chip8

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCPU(t *testing.T) {
	cpu := NewCPU(nil)
	assert.Equal(t, uint16(0x200), cpu.ProgramCounter)
	assert.Equal(t, byte(0x90), cpu.Memory[3])
}

func TestCPU_load(t *testing.T) {
	cpu := NewCPU(nil)
	program := []byte{0x01, 0x02}
	programReader := bytes.NewReader(program)

	instructions, err := cpu.load(0x300, programReader)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, instructions)
	assert.Equal(t, uint16(0x01), uint16(cpu.Memory[0x300]))
	assert.Equal(t, uint16(0x02), uint16(cpu.Memory[0x301]))
}

func TestCPU_Load(t *testing.T) {
	cpu := NewCPU(nil)
	program := []byte{0x01, 0x02}
	programReader := bytes.NewReader(program)
	instructions, err := cpu.Load(programReader)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, instructions)
	assert.Equal(t, uint16(0x01), uint16(cpu.Memory[0x200]))
	assert.Equal(t, uint16(0x02), uint16(cpu.Memory[0x201]))
}

func TestCPU_LoadBytes(t *testing.T) {
	cpu := NewCPU(nil)
	program := []byte{0x01, 0x02}

	instructions, err := cpu.LoadBytes(program)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, instructions)
	assert.Equal(t, uint16(0x01), uint16(cpu.Memory[0x200]))
	assert.Equal(t, uint16(0x02), uint16(cpu.Memory[0x201]))
}

func TestCPU_decodeop(t *testing.T) {
	cpu := NewCPU(nil)
	cpu.Memory[0x200] = 0xC0
	cpu.Memory[0x201] = 0xFE
	op := cpu.decodeOp()
	assert.Equal(t, uint16(0xC0FE), op)
}
