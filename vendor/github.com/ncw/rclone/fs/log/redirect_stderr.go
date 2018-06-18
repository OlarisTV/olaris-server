// Log the panic to the log file - for oses which can't do this

// +build !windows,!darwin,!dragonfly,!freebsd,!linux,!nacl,!netbsd,!openbsd

package log

import (
	"os"

	"github.com/ncw/rclone/fs"
)

// redirectStderr to the file passed in
func redirectStderr(f *os.File) {
	fs.Errorf(nil, "Can't redirect stderr to file")
}
