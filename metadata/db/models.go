package db

import (
	"fmt"
	"github.com/satori/go.uuid"
	"strconv"
)

type MediaType int
type UUIDable struct {
	UUID string `json:"uuid"`
}

func (self *UUIDable) SetUUID() error {
	uuid, err := uuid.NewV4()

	if err != nil {
		fmt.Println("Could not generate unique UID", err)
		return err
	}
	self.UUID = uuid.String()
	return nil
}

func (self *UUIDable) GetUUID() string {
	return self.UUID
}

type MediaItem struct {
	UUIDable
	Title     string
	Year      uint64
	FileName  string
	FilePath  string
	Size      int64
	Library   Library
	LibraryID uint
}

func (self *MediaItem) YearAsString() string {
	return strconv.FormatUint(self.Year, 10)
}

type MediaResult struct {
	Movie     *MovieFile
	TvEpisode *EpisodeFile
}

func FindContentByUUID(uuid string) *MediaResult {
	count := 0
	var movie MovieFile
	var episode EpisodeFile

	env.Db.Where("uuid = ?", uuid).Find(&movie).Count(&count)
	if count > 0 {
		return &MediaResult{Movie: &movie}
	}

	count = 0
	env.Db.Where("uuid = ?", uuid).Find(&episode).Count(&count)
	if count > 0 {
		return &MediaResult{TvEpisode: &episode}
	}

	return &MediaResult{}
}
