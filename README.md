# susimup
An intentionally simplified music player.

## Installation
In the near future there will be precompiles executeables available for download, perhaps even appimage and similar. I will also be looking into bundling the dependency.
For now you must build the project yourself.

### Prerequsites

This project depends on [beep](https://github.com/faiface/beep/) and in turn [oto](https://github.com/hajimehoshi/oto) that requires an alsa library.

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
go build main.go
```
You can now run the resulting executeable.
