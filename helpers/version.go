package helpers

import "fmt"

const (
	VersionMajor = 0
	VersionMinor = 0
	VersionPatch = 1
)

func Version() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}
