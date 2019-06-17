package resolvers

import (
	"github.com/ncw/rclone/fs/config"
	"sort"
)

// Remotes returns all Rclone remotes.
func (r *Resolver) Remotes() (remotes []*string) {
	rems := config.FileSections()

	sort.Strings(rems)

	for i := range rems {
		remotes = append(remotes, &rems[i])
	}

	return remotes
}
