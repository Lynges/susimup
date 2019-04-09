# susimup
An intentionally simplified music player.

susimup was created to fill the need of a music player that you can let anyone operate under less than ideal conditions while giving said operator as few ways to make a mess of things as possible.
The original purpose is as a sound player during live performance where the operator either had very little training/experience using keyboard/screen or is situated where it is not possible to provide them with feedback on what the software is doing.
As such it has some limitations that are actually features. 


## (Non)Features
* Uses a terminal interface and as a result does not require a desktop environment nor even a display server.
* Only reads MP3 files
* Does *not* read any file metadata (IDv2, IDv3, APEv2 and such). Only diplays the filename
* Filenames are always listed alphabetically with subdirectories listed first
* Will not play next track when current one ends. Silence is played instead.
* If current filename ends in `_loop.mp3` the playback will loop indefinitely.
* Takes only one command line argument: the directory to start in. If none is provided, the current directory is used
* Can not access directories above the starting directory.
* When entering a new subdirectory, playback stops.

## Controls
* **\<Up\>/\<Down\>**: Move selection bar up or down.
* **\<Enter\>**: Select item under bar. Mp3 will be played, folder will be accessed.
* **\<Backspace\>**: Go up one directory level unless already at starting directory
* **j**: Play next number. If none have been playing, play first number in current folder.
* **h**: Play previous number. If none have been playing, play first number in current folder.
* **q**: Quit susimup.

## Installation
In the near future there will be precompiled executeables available for download, perhaps even an appimage for linux. I will also be looking into bundling the dependency.
For now you must build the project yourself.

### Prerequsites

This project depends on [beep](https://github.com/faiface/beep/) and in turn [oto](https://github.com/hajimehoshi/oto) that requires an alsa library to work on linux.

To install the lib on ubuntu do:
```
sudo apt install libasound2-dev
```
For other distros take a look at the list of package names here: http://rosindex.github.io/d/libasound2-dev/

### Building

 Build the music player:
```
go get github.com/Lynges/susimup
cd $GOPATH/src/github.com/susimup
go build
```
You can now run the resulting executeable.

To tell susimup where to look for mp3 files, provide the path as an argument: `/path/to/susimup /path/to/soundfiles`
Alternatively you can just move/copy susimup to the folder containing the soundfiles and then: `/path/to/soundfiles/susimup`
