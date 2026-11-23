package app

import (
	"log/slog"
	"os"
)

var MusicList []string

var (
	DataDir string = "./data"
	Addr           = ":8080"
)

func LoadMusicList() {
	entires, err := os.ReadDir(DataDir)
	if err != nil {
		slog.Error("error while reading data dir", "err", err, "dir", DataDir)
		os.Exit(1)
	}

	for _, f := range entires {
		if f.Type().IsDir() {
			continue
		}

		MusicList = append(MusicList, f.Name())
	}
}
