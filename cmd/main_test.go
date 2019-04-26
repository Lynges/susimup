package main

import (
	"fmt"
	"os"
	"testing"
)

func fileinfos() map[string]os.FileInfo {
	rm := make(map[string]os.FileInfo)
	var err error
	rm["testfile"], err = os.Stat("./testmusic/poker-chips-daniel_simon.mp3")
	if err != nil {
		fmt.Println(err)
	}
	rm["testdir"], err = os.Stat("./testmusic")
	if err != nil {
		fmt.Println(err)
	}
	rm["testfile_loop"], err = os.Stat("./testmusic/van-sliding-door-daniel_simon_loop.mp3")
	if err != nil {
		fmt.Println(err)
	}
	return rm
}

func Test_menuEntry_represent(t *testing.T) {
	testFileMap := fileinfos()
	type fields struct {
		File       os.FileInfo
		path       string
		shouldLoop bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"normal", fields{testFileMap["testfile"], "/home/someone/sound", false}, "   poker-chips-daniel_simon.mp3"},
		{"is-directory", fields{testFileMap["testdir"], "/home/someone/sound", false}, "-> testmusic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf := &menuEntry{
				File:       tt.fields.File,
				path:       tt.fields.path,
				shouldLoop: tt.fields.shouldLoop,
			}
			if got := sf.represent(); got != tt.want {
				t.Errorf("menuEntry.represent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generatePosition(t *testing.T) {
	type args struct {
		currentpos int
		modifier   int
		listsize   int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"going nowhere", args{3, 0, 6}, 3},
		{"forward normal", args{3, 1, 6}, 4},
		{"forward limit hit", args{5, 1, 6}, 5},
		{"backward normal", args{3, -1, 6}, 2},
		{"backward limit hit", args{0, -1, 6}, 0},
		{"double forward normal", args{3, 2, 6}, 5},
		{"double forward limit hit", args{4, 2, 6}, 5},
		{"double backward normal", args{3, -2, 6}, 1},
		{"double backward limit hit", args{1, -2, 6}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generatePosition(tt.args.currentpos, tt.args.modifier, tt.args.listsize); got != tt.want {
				t.Errorf("generatePosition() = %v, want %v", got, tt.want)
			}
		})
	}
}
