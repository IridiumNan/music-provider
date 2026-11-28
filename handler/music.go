package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"mp3-stream/app"
)

func HandlerMusic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("only GET method support"))
		return
	}
	var err error

	query := r.URL.Query()
	percentInt := 0
	percent := query.Get("percent")
	if percent != "" {
		percentInt, err = strconv.Atoi(percent)
		if err != nil || percentInt < 0 || percentInt > 100 {
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write([]byte("invalid percent, percent should be in range 0 - 100"))
			if err != nil {
				slog.Error("/music while writing music body", "err", err)
			}
			return
		}
	}

	name := query.Get("name")
	mc, err := app.NewMusicPercent(name, int64(percentInt))
	if err != nil {
		switch {
		case errors.Is(err, app.ErrMusicNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, app.ErrInvalidName):
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}

		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			slog.Error("/music while writing music body", "err", err)
		}
		return
	}
	defer func() {
		if err := mc.Data.Close(); err != nil {
			slog.Error("/music while closing music file", "err", err)
		}
	}()

	// Set the audio type explicitly: name comes from the query string, so
	// letting ServeContent guess from the extension would let a client pick
	// a non-audio Content-Type. "inline" keeps the browser playing it
	// instead of offering a download.
	w.Header().Set("Content-Type", mc.ContentType)
	w.Header().Set("Content-Disposition", "inline")

	// Serve the tail of the file as one self-contained resource, so that
	// Content-Length, Accept-Ranges and 206 Content-Range all refer to the
	// same bytes the client asked for via percent.
	content := io.NewSectionReader(mc.Data, mc.Offset, mc.Size-mc.Offset)
	http.ServeContent(w, r, name, mc.ModTime, content)
}
