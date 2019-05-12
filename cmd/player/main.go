package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Lynges/susimup"

	ui "github.com/gizak/termui"
)

func main() {
	var basepath string
	var err error

	err = ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	if len(os.Args) > 1 {
		basepath, err = filepath.Abs(os.Args[1])
		if err != nil {
			log.Println(os.Args[1] + " caused: " + err.Error() + "\n Using cwd as basepath.")
		}
	} else {
		basepath, err = os.Getwd()
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
	}
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	susimup.Start(basepath, grid)

}
