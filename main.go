package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/gizak/termui"
	ui "github.com/gizak/termui"
)

const styleMarked = "(fg-black,bg-green)"
const stylePlaying = "(fg-white,bg-green)"
const styleNormal = "(fg-green,bg-black)"
const labelEnd = "  -  Press backspace to go up a folder level"

var currentlyPlaying string

type menuEntry struct {
	File       os.FileInfo
	path       string
	shouldLoop bool
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
	currentlyPlaying = ""
}

func (sf *menuEntry) play(playControl <-chan string, playReturn chan<- string) {
	currentlyPlaying = sf.File.Name()

	f, err := os.Open(filepath.Join(sf.path, sf.File.Name()))
	if err != nil {
		log.Fatal(err)
	}

	loopcount := 1
	if sf.shouldLoop {
		loopcount = -1
	}

	stream, format, _ := mp3.Decode(f)
	ctrl := &beep.Ctrl{Streamer: beep.Loop(loopcount, stream)}

	speaker.Clear()
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)) // speaker samplerate of second/10 is from example code.

	// Internal channel, which will signal the end of the playback.
	internalPlay := make(chan struct{})
	var secondstream beep.Streamer
	if sf.shouldLoop {
		secondstream = beep.Callback(func() { close(internalPlay) })
	} else {
		secondstream = beep.Silence(-1)
	}
	speaker.Play(beep.Seq(ctrl, secondstream))

loop:
	for {
		select {
		case stm := <-playControl:
			switch stm {
			case "pause":
				speaker.Lock()
				if ctrl.Paused {
					ctrl.Paused = false
				} else {
					ctrl.Paused = true
				}
				speaker.Unlock()
			case "stop":
				speaker.Clear()
				break loop
			}
		case <-internalPlay:
			break loop
		}
	}
	playReturn <- "player_stopped"
}

func getFolderContent(path string) []menuEntry {
	rawfiles, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	files := []menuEntry{}
	directories := []menuEntry{}
	for _, rf := range rawfiles {
		if !strings.HasPrefix(rf.Name(), ".") {
			switch {
			case rf.IsDir():
				directories = append(directories, menuEntry{rf, path, false})
			case strings.HasSuffix(rf.Name(), ".mp3"):
				shouldLoop := false
				if strings.HasSuffix(rf.Name(), "_loop.mp3") {
					shouldLoop = true
				}
				files = append(files, menuEntry{rf, path, shouldLoop})
			}
		}
	}

	sort.Slice(directories, func(i, j int) bool {
		return strings.ToLower(directories[i].File.Name()) < strings.ToLower(directories[j].File.Name())
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].File.Name()) < strings.ToLower(files[j].File.Name())
	})

	return append(directories, files...)
}

func updateList(ls *ui.List, entries []menuEntry, marker int) {
	items := []string{}
	for index := 0; index < len(entries); index++ {
		representation := entries[index].represent()
		if marker == index {
			representation = "[" + representation + "]" + styleMarked
		} else if entries[index].File.Name() == currentlyPlaying {
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
		if entries[sfi].File.Name() == currentlyPlaying {
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

func channelCombiner(uichan <-chan termui.Event, playchan <-chan string, returnchan chan<- string) {
	for {
		select {
		case event := <-uichan:
			returnchan <- event.ID
		case msg := <-playchan:
			returnchan <- msg
		}
	}
}

func main() {
	var basepath string
	var err error
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
	pathelements := []string{}
	pathelements = append(pathelements, basepath)

	sfiles := getFolderContent(basepath)

	err = ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	ls := ui.NewList()
	ls.BorderLabel = strings.Join(pathelements, "/") + labelEnd
	ls.ItemFgColor = ui.ColorGreen
	barpos := 0

	updateList(ls, sfiles, barpos)

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(10, 0, ls)),
	)
	ui.Body.Align()
	ui.Render(ls)

	// start channel for player control
	playControl := make(chan string)
	// and the channel for player talkback
	playReturn := make(chan string)
	// and the channel for uievents
	uiEvents := ui.PollEvents()
	// and finally the mainchannel
	coms := make(chan string)
	go channelCombiner(uiEvents, playReturn, coms)

	playThis := -1 // -1 is "do nothing, but update" -2 is "stop playing" and 0+ is "play this track in the current folder"
	for {
		com := <-coms
		switch com {
		case "q", "<C-c>":
			return
		case "<Up>":
			barpos = generatePosition(barpos, -1, len(ls.Items))
		case "<Down>":
			barpos = generatePosition(barpos, 1, len(ls.Items))
		case "<Enter>":
			if sfiles[barpos].File.IsDir() {
				pathelements = append(pathelements, sfiles[barpos].File.Name())
				sfiles = getFolderContent(filepath.Join(pathelements...))
				barpos = 0
				playThis = -2
			} else {
				playThis = barpos
			}
		case "h":
			playThis = getNextPlayable(-1, sfiles)
		case "j":
			playThis = getNextPlayable(1, sfiles)
		case "<Backspace>", "C-8>":
			if len(pathelements) > 1 {
				pathelements = pathelements[:len(pathelements)-1]
				sfiles = getFolderContent(strings.Join(pathelements, "/"))
				barpos = 0
			}
		case "<Space>":
			select {
			case playControl <- "pause":
			default:
				// we do this to prevent blocking if space is pressed without a track is playing
			}
		case "player_stopped":
			// empty here to we fall out the bottom and update the list to remove the playerbar
		}

		if playThis >= 0 || playThis == -2 {
			select {
			case playControl <- "stop":
			default:
				// we send a non-blocking stop to prevent any lingering goroutines from causing trouble.
			}
		}
		if playThis >= 0 {
			go sfiles[playThis].play(playControl, playReturn)
			playThis = -1
		}
		updateList(ls, sfiles, barpos)
		ls.BorderLabel = strings.Join(pathelements, "/") + labelEnd
		ui.Clear()
		ui.Render(ls)
	}
}
