package ffmpeg

type ClientCodecCapabilities struct {
	PlayableCodecs []string `json:"playableCodecs"`
}

func (c *ClientCodecCapabilities) Filter(
	representations []StreamRepresentation) []StreamRepresentation {
	filtered := []StreamRepresentation{}

	for _, r := range representations {
		for _, playableCodec := range c.PlayableCodecs {
			if playableCodec == r.Representation.Codecs {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

func (c *ClientCodecCapabilities) CanPlay(sr StreamRepresentation) bool {
	for _, playableCodec := range c.PlayableCodecs {
		if playableCodec == sr.Representation.Codecs {
			return true
		}
	}
	return false
}
