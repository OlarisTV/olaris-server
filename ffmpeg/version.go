package ffmpeg

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Version represents the FFmpeg or FFprobe version number.
type Version struct {
	Major int
	Minor int
	Patch int
}

// VersionFromString parses a Version from a string.
func VersionFromString(versionString string) (*Version, error) {
	version := Version{}

	nums := strings.Split(versionString, ".")
	for i, v := range nums {
		num, err := strconv.Atoi(v)
		if err != nil {
			return nil, NewVersionParseError(err)
		}

		switch i {
		case 0:
			version.Major = num
		case 1:
			version.Minor = num
		case 2:
			version.Patch = num
		default:
			// ignore extras
		}
	}

	// If the version is 0.0.0, it's likely that the version string was
	// malformed.
	if version.Major == 0 && version.Minor == 0 && version.Patch == 0 {
		return nil, NewVersionParseError(
			fmt.Errorf("unable to parse version number from string '%s'", versionString))
	}

	return &version, nil
}

// ToString returns a string representation of the Version.
func (v *Version) ToString() string {
	if v.Patch == 0 {
		// FFmpeg seems to leave the patch version off if it's 0
		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// GetFfmpegVersion returns the version output from ffmpeg and any error that
// occurs.
func GetFfmpegVersion() (version *Version, err error) {
	var out []byte
	cmd := exec.Command("ffmpeg", "-version")
	out, err = cmd.Output()

	r := regexp.MustCompile(`(([0-9])+\.*)+`)
	matches := r.FindStringSubmatch(string(out))

	if len(matches) > 0 {
		version, err = VersionFromString(matches[0])
	}

	return
}

// GetFfprobeVersion returns the version output from ffprobe and any error that
// occurs.
func GetFfprobeVersion() (version *Version, err error) {
	var out []byte
	cmd := exec.Command("ffprobe", "-version")
	out, err = cmd.Output()

	r := regexp.MustCompile(`(([0-9])+\.*)+`)
	matches := r.FindStringSubmatch(string(out))

	if len(matches) > 0 {
		version, err = VersionFromString(matches[0])
	}

	return
}

// VersionParseError is a type of error returned when there is a problem parsing
// the FFmpeg or FFprobe version.
type VersionParseError struct {
	inner error
}

// NewVersionParseError returns a new VersionParseError that wraps the given
// error.
func NewVersionParseError(err error) *VersionParseError {
	return &VersionParseError{
		inner: err,
	}
}

func (err *VersionParseError) Error() string {
	return err.inner.Error()
}

func (err *VersionParseError) Unwrap() error {
	return err.inner

}
