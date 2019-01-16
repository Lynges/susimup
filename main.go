package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	ui "github.com/gizak/termui"
)

const styleMarked = "(fg-black,bg-green)"
const stylePlaying = "(fg-white,bg-green)"
const styleNormal = "(fg-green,bg-black)"

type soundfile struct {
	File      os.FileInfo
	path      string
	IsPlaying bool
}

func (sf *soundfile) represent() string {
	if sf.IsPlaying {
		return sf.File.Name() + stylePlaying
	}
	return sf.File.Name()
}

func (sf *soundfile) play() {
	// Open first sample File
	f, err := os.Open(sf.path)

	// Check for errors when opening the file
	if err != nil {
		log.Fatal(err)
	}

	// Decode the .mp3 File, if you have a .wav file, use wav.Decode(f)
	s, format, _ := mp3.Decode(f)

	// Init the Speaker with the SampleRate of the format and a buffer size of 1/10s
	speaker.Clear()
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	speaker.Play(s)
	// // Channel, which will signal the end of the playback.
	// playing := make(chan struct{})

	// // Now we Play our Streamer on the Speaker
	// speaker.Play(beep.Seq(s, beep.Callback(func() {
	// 	// Callback after the stream Ends
	// 	close(playing)
	// })))
	// <-playing
}

func getFolderContent(path string) []soundfile {
	rawfiles, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	files := []soundfile{}
	for _, rf := range rawfiles {
		if !strings.HasPrefix(rf.Name(), ".") && (strings.HasSuffix(rf.Name(), ".mp3") || rf.IsDir()) {
			files = append(files, soundfile{rf, path, false})
		}
	}
	return files
}

func updateList(ls *ui.List, soundfiles []soundfile, marker int) {
	items := []string{}
	for index := 0; index < len(soundfiles); index++ {
		representation := soundfiles[index].represent()
		if marker == index {
			representation = "[" + representation + "]" + styleMarked
		}
		items = append(items, representation)
	}
	ls.Items = items

	ls.Height = len(ls.Items) + 2 // to make up list including its own size in this number
}

func generatePosition(currentpos int, modifier int, listsize int) int {
	if modifier == 0 {
		return currentpos
	}
	if currentpos+modifier < 0 {
		return 0
	}
	if currentpos+modifier > listsize-1 {
		return listsize - 1
	}
	return currentpos + modifier
}

func main() {
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	// strs := []string{
	// 	"[0] github.com/gizak/termui",
	// 	"[1] [你好，世界](fg-blue)",
	// 	"[2] [こんにちは世界](fg-red)",
	// 	"[3] [color output](fg-white,bg-green)",
	// 	"[4] output.go",
	// 	"[5] random_out.go",
	// 	"[6] dashboard.go",
	// 	"[7] nsf/termbox-go"}

	basepath := "/home/lynge/ildhesten-music-controller/showsound"
	path := []string{}
	path = append(path, basepath)

	sfiles := getFolderContent(basepath)

	ls := ui.NewList()
	ls.BorderLabel = strings.Join(path, "/")
	ls.ItemFgColor = ui.ColorGreen
	barpos := 0

	updateList(ls, sfiles, barpos)

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(10, 0, ls)),
		//ui.NewCol(2, 0, widget1)),
	)

	// calculate layout
	ui.Body.Align()

	ui.Render(ls)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>", "f":
			return
		case "<Up>":
			barpos = generatePosition(barpos, -1, len(ls.Items))

		case "<Down>":
			barpos = generatePosition(barpos, 1, len(ls.Items))

		case "<Enter>":
			if sfiles[barpos].File.IsDir() {
				path = append(path, sfiles[barpos].File.Name())
				sfiles = getFolderContent(strings.Join(path, "/"))
				barpos = 0
			} else {
				sfiles[barpos].play()
			}
		case "<Backspace>", "C-8>":
			if len(path) > 1 {
				path = path[:len(path)-1]
				sfiles = getFolderContent(strings.Join(path, "/"))
				barpos = 0
			}
		}

		updateList(ls, sfiles, barpos)
		// ls.BorderLabel = e.ID
		ls.BorderLabel = strings.Join(path, "/")
		ui.Clear()
		ui.Render(ls)
	}
}

/*
List of events:
	mouse events:
		<MouseLeft> <MouseRight> <MouseMiddle>
		<MouseWheelUp> <MouseWheelDown>
	keyboard events:
		any uppercase or lowercase letter or a set of two letters like j or jj or J or JJ
		<C-d> etc
		<M-d> etc
		<Up> <Down> <Left> <Right>
		<Insert> <Delete> <Home> <End> <Previous> <Next>
		<Backspace> <Tab> <Enter> <Escape> <Space>
		<C-<Space>> etc
	terminal events:
		<Resize>
*/
