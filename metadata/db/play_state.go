package db

import (
	"github.com/jinzhu/gorm"
)

type PlayState struct {
	gorm.Model
	UUID      string
	UserID    uint
	Finished  bool
	Playtime  float64
	MediaID   uint
	MediaType string
}

func CreatePlayState(userID uint, uuid string, finished bool, playtime float64) bool {
	var ps PlayState
	ctx.Db.FirstOrInit(&ps, PlayState{UUID: uuid, UserID: userID})
	ps.Finished = finished
	ps.Playtime = playtime
	ctx.Db.Save(&ps)
	return true

	/*
		//TODO(Maran): I think we don't need actual polymorphism here so we can probably omit this
		count := 0
		var movie Movie
		var episode TvEpisode

		ctx.Db.Where("uuid = ?", uuid).Find(&movie).Count(&count)
		if count > 0 {
			movie.PlayState = ps
			ctx.Db.Save(&movie)
			return true
		}

		count = 0
		ctx.Db.Where("uuid = ?", uuid).Find(&episode).Count(&count)
		if count > 0 {
			episode.PlayState = ps
			ctx.Db.Save(&episode)
			return true
		}
	*/
}
