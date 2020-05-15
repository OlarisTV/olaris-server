package ffmpeg

import (
	"fmt"
	"net/url"

	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/metadata/auth"
)

// NOTE(Leon Handreke): This doesn't really belong here. It doesn't really belong anywhere since it
// really breaks the layering. I'm not really sure where to put this. Probably some URL building
// package.
func buildFfmpegUrlFromFileLocator(fileLocator filesystem.FileLocator) string {
	switch fileLocator.Backend {
	case filesystem.BackendRclone:
		jwt, _ := auth.CreateStreamingJWT(0, fileLocator.String())
		return fmt.Sprintf("http://127.0.0.1:%d/olaris/s/files/jwt/%s",
			viper.GetInt("server.port"), url.PathEscape(jwt))
	case filesystem.BackendLocal:
		return "file://" + fileLocator.Path
	}
	panic(fmt.Sprintf("Unknown fileLocator backend in %s", fileLocator))
}
