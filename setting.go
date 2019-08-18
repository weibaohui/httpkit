package httpkit

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// HTTPSettings is the http.Client setting
type HTTPSettings struct {
	EnableDebug      bool
	UserAgent        string
	ConnectTimeout   time.Duration
	ReadWriteTimeout time.Duration
	TLSClientConfig  *tls.Config
	Proxy            func(*http.Request) (*url.URL, error)
	Transport        http.RoundTripper
	CheckRedirect    func(req *http.Request, via []*http.Request) error
	EnableCookie     bool
	Gzip             bool
	EnableDumpBody   bool
	Retry            struct {
		Status   []int
		Duration time.Duration
		Count    int
		Attempt  int
		Enable   bool
	}
}

var defaultSetting = HTTPSettings{
	ConnectTimeout:   60 * time.Second,
	ReadWriteTimeout: 60 * time.Second,
	Gzip:             true,
	EnableDumpBody:   false,
}
var settingMutex sync.Mutex

// SetDefaultSetting Overwrite default settings
func SetDefaultSetting(setting HTTPSettings) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultSetting = setting
}

// Setting Change request settings
func (b *HTTPRequest) Setting(setting HTTPSettings) *HTTPRequest {
	b.setting = setting
	return b
}

// SetUserAgent sets User-Agent header field
func (b *HTTPRequest) SetUserAgent(userAgent string) *HTTPRequest {
	b.setting.UserAgent = userAgent
	return b
}

// WithTimeout sets connect time out and read-write time out for Request.
func (b *HTTPRequest) SetTimeout(connectTimeout, readWriteTimeout time.Duration) *HTTPRequest {
	b.setting.ConnectTimeout = connectTimeout
	b.setting.ReadWriteTimeout = readWriteTimeout
	return b
}

// SetTLSClientConfig sets tls connection configurations if visiting https url.
func (b *HTTPRequest) SetTLSClientConfig(config *tls.Config) *HTTPRequest {
	b.setting.TLSClientConfig = config
	return b
}

// SetCheckRedirect specifies the policy for handling redirects.
//
// If CheckRedirect is nil, the Client uses its default policy,
// which is to stop after 10 consecutive requests.
func (b *HTTPRequest) SetCheckRedirect(redirect func(req *http.Request, via []*http.Request) error) *HTTPRequest {
	b.setting.CheckRedirect = redirect
	return b
}

// SetProxy set the http proxy
// example:
//
//	func(req *http.Request) (*url.URL, error) {
// 		u, _ := url.ParseRequestURI("http://127.0.0.1:8118")
// 		return u, nil
// 	}
func (b *HTTPRequest) SetProxy(proxy func(*http.Request) (*url.URL, error)) *HTTPRequest {
	b.setting.Proxy = proxy
	return b
}

// SetTransport set the setting transport
func (b *HTTPRequest) SetTransport(transport http.RoundTripper) *HTTPRequest {
	b.setting.Transport = transport
	return b
}

// EnableCookie sets enable/disable cookieJar
func (b *HTTPRequest) EnableCookie() *HTTPRequest {
	b.setting.EnableCookie = true
	return b
}

// EnableDebug sets show debug or not when executing request.
func (b *HTTPRequest) EnableDebug() *HTTPRequest {
	b.setting.EnableDebug = true
	return b
}

// EnableDump setting whether need to Dump the Body.
func (b *HTTPRequest) EnableDump() *HTTPRequest {
	b.setting.EnableDumpBody = true
	return b
}
