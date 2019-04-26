package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"susimup"

	ui "github.com/gizak/termui"
)

func main() {
	var basepath string
	var err error

	err = ui.Init()
	if err != nil {
		panic(err)
	}

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
	susimup.Start(basepath)
	defer ui.Close()
}
