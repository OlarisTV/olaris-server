package db

import (
	"encoding/json"
)

type MediaPlaybackState struct {
	Filename string `json:"filename"`
	Playtime int    `json:"playtime"`
}

func (db *DB) InsertOrUpdateMediaPlaybackState(m MediaPlaybackState) error {
	val, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return db.db.Put([]byte("playback-state-"+m.Filename), val, nil)
}

func (db *DB) GetMediaPlaybackState(filename string) (*MediaPlaybackState, error) {
	val, err := db.db.Get([]byte("playback-state-"+filename), nil)
	if val == nil {
		return nil, nil
	}
	m := MediaPlaybackState{}
	err = json.Unmarshal(val, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
