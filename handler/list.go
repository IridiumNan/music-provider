package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"mp3-stream/app"
)

func HandlerListJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("only GET method support"))
		return
	}
	type JsonBody struct {
		Code int      `json:"code"`
		List []string `json:"list"`
	}

	var body JsonBody
	if len(app.MusicList) == 0 {
		w.WriteHeader(http.StatusNotFound)
		body = JsonBody{
			Code: http.StatusNotFound,
			List: nil,
		}

	} else {
		body = JsonBody{
			Code: http.StatusOK,
			List: app.MusicList,
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		log.Fatal("error while marshal JsonBody, fix your struct", err)
	}

	_, err = w.Write(data)
	if err != nil {
		slog.Error("/list/json error while writing body", "err", err)
	}
}

func HandlerListText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("only GET method support"))
		return
	}
	if len(app.MusicList) == 0 {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte("No available music, contact music provider please"))
		if err != nil {
			slog.Error("/list/text while writting body", "err", err)
		}
		return
	}
	var body bytes.Buffer

	for i := range app.MusicList {
		fmt.Fprintln(&body, app.MusicList[i])
	}

	_, err := w.Write(body.Bytes())
	if err != nil {
		slog.Error("/list/text while writting body", "err", err)
	}
}
