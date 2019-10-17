package resolvers

// MediaItemResolver is a resolver around media types.
type MediaItemResolver struct {
	r interface{}
}

// ToMovie tries to convert media to Movie
func (r *MediaItemResolver) ToMovie() (*MovieResolver, bool) {
	res, ok := r.r.(*MovieResolver)
	return res, ok
}

// ToEpisode tries to convert media to Episode
func (r *MediaItemResolver) ToEpisode() (*EpisodeResolver, bool) {
	res, ok := r.r.(*EpisodeResolver)
	return res, ok
}
