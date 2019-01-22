package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	ui "github.com/gizak/termui"
)

const styleMarked = "(fg-black,bg-green)"
const stylePlaying = "(fg-white,bg-green)"
const styleNormal = "(fg-green,bg-black)"

type menuEntry struct {
	File      os.FileInfo
	path      string
	IsPlaying bool
}

func (sf *menuEntry) represent() string {
	var prefix string
	if sf.File.IsDir() {
		prefix = "-> "
	} else {
		prefix = "   "
	}
	return prefix + sf.File.Name()
}

func (sf *menuEntry) stopPlaying() {
	sf.IsPlaying = false
}

func (sf *menuEntry) play(playControl <-chan string) {

	f, err := os.Open(sf.path + "/" + sf.File.Name())

	if err != nil {
		log.Fatal(err)
	}

	stream, format, _ := mp3.Decode(f)

	ctrl := &beep.Ctrl{Streamer: beep.Loop(-1, stream)}

	// Init the Speaker with the SampleRate of the format and a buffer size of 1/10s
	speaker.Clear()
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// Internal channel, which will signal the end of the playback.
	internalPlay := make(chan struct{})

	speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
		close(internalPlay)
	})))

	sf.IsPlaying = true
	defer sf.stopPlaying()
loop:
	for {
		select {
		case <-internalPlay:
			break loop
			// internalPlay := make(chan struct{})

			// speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
			// 	close(internalPlay)
			// })))

		case stm := <-playControl:
			switch stm {
			case "pause":
				speaker.Lock()
				if ctrl.Paused {
					ctrl.Paused = false
				} else {
					log.Print(ctrl.Streamer.Err())
					ctrl.Paused = true
				}
				speaker.Unlock()
			case "stop":
				break loop
			}
		case <-playControl:
			break loop
		}
	}
}

func getFolderContent(path string) []menuEntry {
	rawfiles, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	files := []menuEntry{}
	for _, rf := range rawfiles {
		if !strings.HasPrefix(rf.Name(), ".") && (strings.HasSuffix(rf.Name(), ".mp3") || rf.IsDir()) {
			files = append(files, menuEntry{rf, path, false})
		}
	}
	return files
}

func updateList(ls *ui.List, entries []menuEntry, marker int) {
	items := []string{}
	for index := 0; index < len(entries); index++ {
		representation := entries[index].represent()
		if marker == index {
			representation = "[" + representation + "]" + styleMarked
		} else if entries[index].IsPlaying {
			representation = "[" + representation + "]" + stylePlaying
		}
		items = append(items, representation)
	}
	ls.Items = items

	ls.Height = len(ls.Items) + 2 // to make up for list including its own size in this number
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

func getNextPlayable(direction int, entries []menuEntry) int {
	position := -1
	for sfi := 0; sfi < len(entries); sfi++ {
		if entries[sfi].IsPlaying {
			position = sfi
		}
	}
	if position == -1 {
		direction = 1
	}
	for index := position + direction; index > -1 && index < len(entries); index = index + direction {
		if !entries[index].File.IsDir() {
			return index
		}
	}
	return position
}

func main() {
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	log.SetOutput(file)

	// strs := []string{
	// 	"[0] github.com/gizak/termui",
	// 	"[1] [你好，世界](fg-blue)",
	// 	"[2] [こんにちは世界](fg-red)",
	// 	"[3] [color output](fg-white,bg-green)",
	// 	"[4] output.go",
	// 	"[5] random_out.go",
	// 	"[6] dashboard.go",
	// 	"[7] nsf/termbox-go"}

	basepath := "testmusic"
	path := []string{}
	path = append(path, basepath)

	sfiles := getFolderContent(basepath)

	// start channel for player control
	playControl := make(chan string)
	playThis := -1
	// configure ui
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
		case "q", "<C-c>":
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
				playThis = barpos
			}
		case "<Backspace>", "C-8>":
			if len(path) > 1 {
				path = path[:len(path)-1]
				sfiles = getFolderContent(strings.Join(path, "/"))
				barpos = 0
			}
		case "<Space>":
			playControl <- "pause"
		}
		if playThis >= 0 {
			playControl = make(chan string)
			go sfiles[playThis].play(playControl)
			playThis = -1
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
