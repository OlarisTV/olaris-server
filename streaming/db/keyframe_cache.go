package db

import (
	"encoding/json"
)

type KeyframeCache struct {
	Filename           string  `json:"filename"`
	KeyframeTimestamps []int64 `json:"keyframeTimestamps"`
}

func (db *DB) InsertOrUpdateKeyframeCache(c KeyframeCache) error {
	val, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return db.db.Put([]byte("keyframe-cache-"+c.Filename), val, nil)
}

func (db *DB) GetKeyframeCache(filename string) (*KeyframeCache, error) {
	val, err := db.db.Get([]byte("keyframe-cache-"+filename), nil)
	if val == nil {
		return nil, nil
	}
	m := KeyframeCache{}
	err = json.Unmarshal(val, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
