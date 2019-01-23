package main

import (
	"io/ioutil"
	"log"
	"os"
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

func (sf *menuEntry) play(playControl <-chan string, playReturn chan<- string) {

	f, err := os.Open(sf.path + "/" + sf.File.Name())

	if err != nil {
		log.Fatal(err)
	}

	stream, format, _ := mp3.Decode(f)

	ctrl := &beep.Ctrl{Streamer: beep.Loop(-1, stream)}
	ctrl.Paused = false

	speaker.Clear()
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)) // speaker samplerate of second/10 is from example code.

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
		case stm := <-playControl:
			switch stm {
			case "pause":
				log.Print("pause for " + sf.File.Name())
				speaker.Lock()
				if ctrl.Paused {
					ctrl.Paused = false
					log.Print("paused = false")
				} else {
					ctrl.Paused = true
					log.Print("paused = true")
				}
				speaker.Unlock()
			case "stop":
				log.Print("stop for " + sf.File.Name())
				break loop
			}
		case <-internalPlay:
			log.Println("internalPlay has just been closedfor " + sf.File.Name())
			break loop
		}
	}
	playReturn <- "player_stopped"
}

func getFolderContent(path string) []menuEntry {
	rawfiles, err := ioutil.ReadDir(path)
	if err != nil {
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
				files = append(files, menuEntry{rf, path, false})
			}
		}
	}
	log.Println(directories)
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
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()
	// TODO: some better selection for musicdir
	basepath := "testmusic"
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

	playThis := -1
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
			select {
			case playControl <- "pause":
			default:
				// we do this to prevent blocking if space is pressed without a track is playing
			}
		case "player_stopped":
			// empty here to we fall through and update the list to remove the playerbar

		}
		if playThis >= 0 {
			select {
			case playControl <- "stop":
			default:
				// we send a non-blocking stop to prevent any lingering goroutines from causing trouble.
			}
			go sfiles[playThis].play(playControl, playReturn)
			playThis = -1
		}
		updateList(ls, sfiles, barpos)
		ls.BorderLabel = strings.Join(path, "/")
		ui.Clear()
		ui.Render(ls)
	}
}
