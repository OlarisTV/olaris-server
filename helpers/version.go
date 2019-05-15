package helpers

import "fmt"

var GitCommit string

const (
	VersionMajor = 0
	VersionMinor = 1
	VersionPatch = 1
)

func Version() string {
	version := fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
	if GitCommit != "" {
		version = fmt.Sprintf("%s (%s)", version, GitCommit)
	}
	return version
}
