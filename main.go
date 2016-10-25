package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsf/termbox-go"
	"github.com/scottjab/go-chip8/chip8"
	"io/ioutil"
)

func main() {
	d, err := chip8.NewTermboxDisplay(
		termbox.ColorDefault,
		termbox.ColorDefault,
	)
	defer d.Close()
	if err != nil {
		panic(err)
	}
	k := chip8.NewTermboxKeypad()
	cpu := chip8.NewCPU(&chip8.Options{
		ClockSpeed: 60,
	})
	cpu.Graphics.Display = d
	cpu.Keypad = k

	log.Println("Loading rom")
	program, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	_, err = cpu.LoadBytes(program)
	if err != nil {
		panic(err)
	}
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		cpu.Stop()
	}()

	err = cpu.Run()
	if err != nil {
		panic(err)
	}
}
