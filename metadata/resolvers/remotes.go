package resolvers

import (
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/spf13/viper"
	"os"
	"sort"
)

// Remotes returns all Rclone remotes.
func (r *Resolver) Remotes() (remotes []*string) {
	// Set a custom config file for rclone, if specified. `rclone.configFile` defaults to '$HOME/.config/rclone/rclone.conf'
	config.SetConfigPath(os.ExpandEnv(viper.GetString("rclone.configFile")))
	configfile.Install()

	rems := config.FileSections()

	sort.Strings(rems)

	for i := range rems {
		remotes = append(remotes, &rems[i])
	}

	return remotes
}
