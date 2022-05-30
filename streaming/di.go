package streaming

import "github.com/goava/di"

func Options() di.Option {
	return di.Options(
		di.Provide(NewStreamingController, di.Tags{"type": "streaming"}),
	)
}
