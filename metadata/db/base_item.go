package db

type BaseItem struct {
	UUIDable
	TmdbID       int
	Overview     string `gorm:"type:text"`
	BackdropPath string
	PosterPath   string
}
