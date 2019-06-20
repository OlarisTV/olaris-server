package hubic

import (
	"net/http"
	"time"

	"github.com/ncw/rclone/fs"
	"github.com/ncw/swift"
)

// auth is an authenticator for swift
type auth struct {
	f *Fs
}

// newAuth creates a swift authenticator
func newAuth(f *Fs) *auth {
	return &auth{
		f: f,
	}
}

// Request constructs a http.Request for authentication
//
// returns nil for not needed
func (a *auth) Request(*swift.Connection) (r *http.Request, err error) {
	const retries = 10
	for try := 1; try <= retries; try++ {
		err = a.f.getCredentials()
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
		fs.Debugf(a.f, "retrying auth request %d/%d: %v", try, retries, err)
	}
	return nil, err
}

// Response parses the result of an http request
func (a *auth) Response(resp *http.Response) error {
	return nil
}

// The public storage URL - set Internal to true to read
// internal/service net URL
func (a *auth) StorageUrl(Internal bool) string { // nolint
	return a.f.credentials.Endpoint
}

// The access token
func (a *auth) Token() string {
	return a.f.credentials.Token
}

// The CDN url if available
func (a *auth) CdnUrl() string { // nolint
	return ""
}

// Check the interfaces are satisfied
var _ swift.Authenticator = (*auth)(nil)
