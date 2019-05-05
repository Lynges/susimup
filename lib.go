package susimup

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
	"github.com/gizak/termui/widgets"
)

const styleMarked = "(fg:black,bg:green)"
const stylePlaying = "(fg:white,bg:green)"
const styleNormal = "(fg:green,bg:black)"

var currentlyPlaying string

type menuEntry struct {
	File       os.FileInfo
	path       string
	shouldLoop bool
}

type message struct {
	key     string
	action  string
	payload string
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

func (sf *menuEntry) play(playControl <-chan message, playReturn chan<- message) {
	defer sendMessage(playReturn, message{"player_feedback", "player_stopped", sf.File.Name()})

	f, err := os.Open(filepath.Join(sf.path, sf.File.Name()))
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

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
			switch stm.key {
			case "player_control":
				if stm.payload == "all" || stm.payload == sf.File.Name() {
					switch stm.action {
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
				}

			}

		case <-internalPlay:
			break loop
		}
	}
}

func sendMessage(send chan<- message, msg message) {
	send <- msg
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

func updateList(ls *widgets.List, entries []menuEntry, marker int) {
	items := []string{}
	for index := 0; index < len(entries); index++ {
		representation := entries[index].represent()
		if entries[index].File.Name() == currentlyPlaying {
			representation = "[" + representation + "]" + stylePlaying
		}
		items = append(items, representation)
	}
	ls.Rows = items
	if len(items) > 0 {
		ls.SelectedRow = marker
	}
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

	for index := position + direction; index >= -1 && index <= len(entries); index = index + direction {
		if index == len(entries) {
			// we know that we reached the end of the list without finding anything
			return -2
		}
		if index == -1 {
			// we know that we reach the beginning without finding anything
			return -2
		}
		if !entries[index].File.IsDir() {
			return index
		}
	}

	return position
}

func createTitle(pathelements []string) string {
	var title = []string{strings.Join(pathelements, "/")}
	if len(pathelements) > 1 {
		title = append(title, "Backspace goes up a folder level")
	}
	title = append(title, "Q quits the player")
	return strings.Join(title, " - ")
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

func eventChanneller(uichan <-chan termui.Event, receivechan chan<- message) {
	for {
		select {
		case event := <-uichan:
			receivechan <- message{"uievent", "key_pressed", event.ID}
		}
	}
}

// Start handles showing the music player and playing music.
// basepath is expected to be a string of absolute path to the desired starting directory.
// manageui tells susimup to init and close its own termui instance.
func Start(basepath string, grid *ui.Grid) {
	pathelements := []string{}
	pathelements = append(pathelements, basepath)

	sfiles := getFolderContent(basepath)

	ls := widgets.NewList()
	ls.Title = createTitle(pathelements)
	ls.TextStyle = ui.NewStyle(ui.ColorGreen)
	ls.WrapText = false
	ls.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorGreen)
	barpos := 0

	updateList(ls, sfiles, barpos)

	// setup grid
	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(1.0, ls),
		),
	)
	// ui.Body.Align()
	ui.Render(grid)

	// and the channel for uievents
	uiEvents := ui.PollEvents()

	receive := make(chan message)
	send := make(chan message)
	go eventChanneller(uiEvents, receive)

	// status vars
	quitPlayer := false
	playThis := -1 // -1 is "do nothing, but update" -2 is "stop playing" and 0+ is "play this track in the current folder"
	for {
		com := <-receive
		switch com.key {
		case "uievent":
			switch com.payload {
			case "q", "<C-c>":
				playThis = -2
				quitPlayer = true
			case "<Up>":
				barpos = generatePosition(barpos, -1, len(ls.Rows))
			case "<Down>":
				barpos = generatePosition(barpos, 1, len(ls.Rows))
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
				case send <- message{"player_control", "pause", "all"}:
				default:
					// we do this to prevent blocking if space is pressed without a track is playing
				}
			}
		case "player_feedback":
			switch com.action {
			case "player_stopped":
				if currentlyPlaying == com.payload {
					currentlyPlaying = ""
				}
				// empty here to we fall out the bottom and update the list to remove the playerbar
			}
		}

		if playThis >= 0 || playThis == -2 {
			select {
			case send <- message{"player_control", "stop", "all"}:
			default:
				// we send a non-blocking stop to prevent any lingering goroutines from causing trouble.
			}
		}
		if playThis >= 0 {
			currentlyPlaying = sfiles[playThis].File.Name()
			go sfiles[playThis].play(send, receive)
			playThis = -1
		}

		updateList(ls, sfiles, barpos)
		ls.Title = createTitle(pathelements)
		ui.Clear()
		if quitPlayer {
			return
		}
		ui.Render(grid)
	}
}
