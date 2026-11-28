package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"slices"

	_ "embed"
	"mp3-stream/app"
	"mp3-stream/handler"
	"mp3-stream/middleware"
)

//go:embed USAGE.txt
var usage string

//go:embed public.pem
var publicKey []byte

var helpOptions = []string{"-h", "--help", "help"}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	if len(os.Args) > 1 {
		arg1 := os.Args[1]

		if slices.Contains(helpOptions, arg1) {
			fmt.Print(usage)
			return
		}
		app.Addr = os.Args[1]
	}

	if len(os.Args) > 2 {
		app.DataDir = os.Args[2]
	}

	if _, err := os.ReadDir(app.DataDir); os.IsNotExist(err) {
		slog.Error("data dir not found, please create yourself", "path", app.DataDir)
		os.Exit(1)
	}

	app.LoadMusicList()

	serve()
}

func serve() {
	auth, err := handler.NewAuth(publicKey)
	if err != nil {
		slog.Error("error when init auth obj", "err", err)
		os.Exit(1)
	}

	router := http.NewServeMux()

	router.HandleFunc("/", handler.HanlderHelp)

	router.HandleFunc("/auth/challenge", auth.HandleChallenge)

	// add auth middleware for source visit
	router.HandleFunc("/list/json", auth.MiddleWare(handler.HandlerListJSON))
	router.HandleFunc("/list/text", auth.MiddleWare(handler.HandlerListText))

	router.HandleFunc("/music", auth.MiddleWare(handler.HandlerMusic))

	slog.Info("starting serving musics", "data_dir", app.DataDir, "app.Addr", app.Addr)

	log.Fatal(http.ListenAndServe(app.Addr, middleware.MWTrackAddr(router)))
}
