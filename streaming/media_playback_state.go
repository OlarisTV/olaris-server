package streaming

import (
	"encoding/json"
	"gitlab.com/bytesized/bytesized-streaming/db"
	"net/http"
)

type SetMediaPlaybackStateRequest struct {
	Filename string `json:"filename"`
	Playtime int    `json:"playtime"`
}

func handleSetMediaPlaybackState(w http.ResponseWriter, r *http.Request) {
	req := SetMediaPlaybackStateRequest{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	err = db.GetSharedDB().InsertOrUpdateMediaPlaybackState(
		db.MediaPlaybackState{Filename: req.Filename, Playtime: req.Playtime})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
