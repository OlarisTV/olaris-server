package helpers

import (
	"path"
)


func GetDefaultRcloneConfigPath() string {
	return path.Join(GetHome(), ".config", "rclone", "rclone.conf")
}