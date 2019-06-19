// Package fshttp contains the common http parts of the config, Transport and Client
package fshttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"reflect"
	"sync"
	"time"

	"github.com/ncw/rclone/fs"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/time/rate"
)

const (
	separatorReq  = ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
	separatorResp = "<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
)

var (
	transport    http.RoundTripper
	noTransport  = new(sync.Once)
	tpsBucket    *rate.Limiter // for limiting number of http transactions per second
	cookieJar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
)

// StartHTTPTokenBucket starts the token bucket if necessary
func StartHTTPTokenBucket() {
	if fs.Config.TPSLimit > 0 {
		tpsBurst := fs.Config.TPSLimitBurst
		if tpsBurst < 1 {
			tpsBurst = 1
		}
		tpsBucket = rate.NewLimiter(rate.Limit(fs.Config.TPSLimit), tpsBurst)
		fs.Infof(nil, "Starting HTTP transaction limiter: max %g transactions/s with burst %d", fs.Config.TPSLimit, tpsBurst)
	}
}

// A net.Conn that sets a deadline for every Read or Write operation
type timeoutConn struct {
	net.Conn
	timeout time.Duration
}

// create a timeoutConn using the timeout
func newTimeoutConn(conn net.Conn, timeout time.Duration) (c *timeoutConn, err error) {
	c = &timeoutConn{
		Conn:    conn,
		timeout: timeout,
	}
	err = c.nudgeDeadline()
	return
}

// Nudge the deadline for an idle timeout on by c.timeout if non-zero
func (c *timeoutConn) nudgeDeadline() (err error) {
	if c.timeout == 0 {
		return nil
	}
	when := time.Now().Add(c.timeout)
	return c.Conn.SetDeadline(when)
}

// readOrWrite bytes doing idle timeouts
func (c *timeoutConn) readOrWrite(f func([]byte) (int, error), b []byte) (n int, err error) {
	n, err = f(b)
	// Don't nudge if no bytes or an error
	if n == 0 || err != nil {
		return
	}
	// Nudge the deadline on successful Read or Write
	err = c.nudgeDeadline()
	return
}

// Read bytes doing idle timeouts
func (c *timeoutConn) Read(b []byte) (n int, err error) {
	return c.readOrWrite(c.Conn.Read, b)
}

// Write bytes doing idle timeouts
func (c *timeoutConn) Write(b []byte) (n int, err error) {
	return c.readOrWrite(c.Conn.Write, b)
}

// setDefaults for a from b
//
// Copy the public members from b to a.  We can't just use a struct
// copy as Transport contains a private mutex.
func setDefaults(a, b interface{}) {
	pt := reflect.TypeOf(a)
	t := pt.Elem()
	va := reflect.ValueOf(a).Elem()
	vb := reflect.ValueOf(b).Elem()
	for i := 0; i < t.NumField(); i++ {
		aField := va.Field(i)
		// Set a from b if it is public
		if aField.CanSet() {
			bField := vb.Field(i)
			aField.Set(bField)
		}
	}
}

// dial with context and timeouts
func dialContextTimeout(ctx context.Context, network, address string, ci *fs.ConfigInfo) (net.Conn, error) {
	dialer := NewDialer(ci)
	c, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return c, err
	}
	return newTimeoutConn(c, ci.Timeout)
}

// ResetTransport resets the existing transport, allowing it to take new settings.
// Should only be used for testing.
func ResetTransport() {
	noTransport = new(sync.Once)
}

// NewTransport returns an http.RoundTripper with the correct timeouts
func NewTransport(ci *fs.ConfigInfo) http.RoundTripper {
	(*noTransport).Do(func() {
		// Start with a sensible set of defaults then override.
		// This also means we get new stuff when it gets added to go
		t := new(http.Transport)
		setDefaults(t, http.DefaultTransport.(*http.Transport))
		t.Proxy = http.ProxyFromEnvironment
		t.MaxIdleConnsPerHost = 2 * (ci.Checkers + ci.Transfers + 1)
		t.MaxIdleConns = 2 * t.MaxIdleConnsPerHost
		t.TLSHandshakeTimeout = ci.ConnectTimeout
		t.ResponseHeaderTimeout = ci.Timeout

		// TLS Config
		t.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: ci.InsecureSkipVerify,
		}

		// Load client certs
		if ci.ClientCert != "" || ci.ClientKey != "" {
			if ci.ClientCert == "" || ci.ClientKey == "" {
				log.Fatalf("Both --client-cert and --client-key must be set")
			}
			cert, err := tls.LoadX509KeyPair(ci.ClientCert, ci.ClientKey)
			if err != nil {
				log.Fatalf("Failed to load --client-cert/--client-key pair: %v", err)
			}
			t.TLSClientConfig.Certificates = []tls.Certificate{cert}
			t.TLSClientConfig.BuildNameToCertificate()
		}

		// Load CA cert
		if ci.CaCert != "" {
			caCert, err := ioutil.ReadFile(ci.CaCert)
			if err != nil {
				log.Fatalf("Failed to read --ca-cert: %v", err)
			}
			caCertPool := x509.NewCertPool()
			ok := caCertPool.AppendCertsFromPEM(caCert)
			if !ok {
				log.Fatalf("Failed to add certificates from --ca-cert")
			}
			t.TLSClientConfig.RootCAs = caCertPool
		}

		t.DisableCompression = ci.NoGzip
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialContextTimeout(ctx, network, addr, ci)
		}
		t.IdleConnTimeout = 60 * time.Second
		t.ExpectContinueTimeout = ci.ConnectTimeout
		// Wrap that http.Transport in our own transport
		transport = newTransport(ci, t)
	})
	return transport
}

// NewClient returns an http.Client with the correct timeouts
func NewClient(ci *fs.ConfigInfo) *http.Client {
	transport := &http.Client{
		Transport: NewTransport(ci),
	}
	if ci.Cookie {
		transport.Jar = cookieJar
	}
	return transport
}

// Transport is a our http Transport which wraps an http.Transport
// * Sets the User Agent
// * Does logging
type Transport struct {
	*http.Transport
	dump          fs.DumpFlags
	filterRequest func(req *http.Request)
	userAgent     string
}

// newTransport wraps the http.Transport passed in and logs all
// roundtrips including the body if logBody is set.
func newTransport(ci *fs.ConfigInfo, transport *http.Transport) *Transport {
	return &Transport{
		Transport: transport,
		dump:      ci.Dump,
		userAgent: ci.UserAgent,
	}
}

// SetRequestFilter sets a filter to be used on each request
func (t *Transport) SetRequestFilter(f func(req *http.Request)) {
	t.filterRequest = f
}

// A mutex to protect this map
var checkedHostMu sync.RWMutex

// A map of servers we have checked for time
var checkedHost = make(map[string]struct{}, 1)

// Check the server time is the same as ours, once for each server
func checkServerTime(req *http.Request, resp *http.Response) {
	host := req.URL.Host
	if req.Host != "" {
		host = req.Host
	}
	checkedHostMu.RLock()
	_, ok := checkedHost[host]
	checkedHostMu.RUnlock()
	if ok {
		return
	}
	dateString := resp.Header.Get("Date")
	if dateString == "" {
		return
	}
	date, err := http.ParseTime(dateString)
	if err != nil {
		fs.Debugf(nil, "Couldn't parse Date: from server %s: %q: %v", host, dateString, err)
		return
	}
	dt := time.Since(date)
	const window = 5 * 60 * time.Second
	if dt > window || dt < -window {
		fs.Logf(nil, "Time may be set wrong - time from %q is %v different from this computer", host, dt)
	}
	checkedHostMu.Lock()
	checkedHost[host] = struct{}{}
	checkedHostMu.Unlock()
}

// cleanAuth gets rid of one authBuf header within the first 4k
func cleanAuth(buf, authBuf []byte) []byte {
	// Find how much buffer to check
	n := 4096
	if len(buf) < n {
		n = len(buf)
	}
	// See if there is an Authorization: header
	i := bytes.Index(buf[:n], authBuf)
	if i < 0 {
		return buf
	}
	i += len(authBuf)
	// Overwrite the next 4 chars with 'X'
	for j := 0; i < len(buf) && j < 4; j++ {
		if buf[i] == '\n' {
			break
		}
		buf[i] = 'X'
		i++
	}
	// Snip out to the next '\n'
	j := bytes.IndexByte(buf[i:], '\n')
	if j < 0 {
		return buf[:i]
	}
	n = copy(buf[i:], buf[i+j:])
	return buf[:i+n]
}

var authBufs = [][]byte{
	[]byte("Authorization: "),
	[]byte("X-Auth-Token: "),
}

// cleanAuths gets rid of all the possible Auth headers
func cleanAuths(buf []byte) []byte {
	for _, authBuf := range authBufs {
		buf = cleanAuth(buf, authBuf)
	}
	return buf
}

// RoundTrip implements the RoundTripper interface.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	// Get transactions per second token first if limiting
	if tpsBucket != nil {
		tbErr := tpsBucket.Wait(req.Context())
		if tbErr != nil {
			fs.Errorf(nil, "HTTP token bucket error: %v", err)
		}
	}
	// Force user agent
	req.Header.Set("User-Agent", t.userAgent)
	// Filter the request if required
	if t.filterRequest != nil {
		t.filterRequest(req)
	}
	// Logf request
	if t.dump&(fs.DumpHeaders|fs.DumpBodies|fs.DumpAuth|fs.DumpRequests|fs.DumpResponses) != 0 {
		buf, _ := httputil.DumpRequestOut(req, t.dump&(fs.DumpBodies|fs.DumpRequests) != 0)
		if t.dump&fs.DumpAuth == 0 {
			buf = cleanAuths(buf)
		}
		fs.Debugf(nil, "%s", separatorReq)
		fs.Debugf(nil, "%s (req %p)", "HTTP REQUEST", req)
		fs.Debugf(nil, "%s", string(buf))
		fs.Debugf(nil, "%s", separatorReq)
	}
	// Do round trip
	resp, err = t.Transport.RoundTrip(req)
	// Logf response
	if t.dump&(fs.DumpHeaders|fs.DumpBodies|fs.DumpAuth|fs.DumpRequests|fs.DumpResponses) != 0 {
		fs.Debugf(nil, "%s", separatorResp)
		fs.Debugf(nil, "%s (req %p)", "HTTP RESPONSE", req)
		if err != nil {
			fs.Debugf(nil, "Error: %v", err)
		} else {
			buf, _ := httputil.DumpResponse(resp, t.dump&(fs.DumpBodies|fs.DumpResponses) != 0)
			fs.Debugf(nil, "%s", string(buf))
		}
		fs.Debugf(nil, "%s", separatorResp)
	}
	if err == nil {
		checkServerTime(req, resp)
	}
	return resp, err
}

// NewDialer creates a net.Dialer structure with Timeout, Keepalive
// and LocalAddr set from rclone flags.
func NewDialer(ci *fs.ConfigInfo) *net.Dialer {
	dialer := &net.Dialer{
		Timeout:   ci.ConnectTimeout,
		KeepAlive: 30 * time.Second,
	}
	if ci.BindAddr != nil {
		dialer.LocalAddr = &net.TCPAddr{IP: ci.BindAddr}
	}
	return dialer
}
