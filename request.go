package httpkit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// HTTPRequest provides more useful methods for requesting one url than http.Request.
type HTTPRequest struct {
	ctx     context.Context
	url     string
	req     *http.Request
	params  map[string][]string
	files   map[string]string
	setting HTTPSettings
	resp    *http.Response
	body    []byte
	dump    []byte
}

var defaultcookieJar http.CookieJar

// createDefaultCookie creates a global cookieJar to store cookies.
func createDefaultCookie() {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultcookieJar, _ = cookiejar.New(nil)
}

// NewRequest return *HTTPRequest with specific method
func NewRequest(rawURL, method string) *HTTPRequest {
	var resp http.Response
	u, err := url.Parse(rawURL)
	if err != nil {
		log.Println("Httplib:", err)
	}
	req := http.Request{
		URL:        u,
		Method:     method,
		Header:     make(http.Header),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	return &HTTPRequest{
		url:     rawURL,
		req:     &req,
		params:  map[string][]string{},
		files:   map[string]string{},
		setting: defaultSetting,
		resp:    &resp,
	}
}

// Get returns *HTTPRequest with GET method.
func Get(url string) *HTTPRequest {
	return NewRequest(url, "GET")
}

// Post returns *HTTPRequest with POST method.
func Post(url string) *HTTPRequest {
	return NewRequest(url, "POST")
}

// Put returns *HTTPRequest with PUT method.
func Put(url string) *HTTPRequest {
	return NewRequest(url, "PUT")
}

// Delete returns *HTTPRequest DELETE method.
func Delete(url string) *HTTPRequest {
	return NewRequest(url, "DELETE")
}

// Head returns *HTTPRequest with HEAD method.
func Head(url string) *HTTPRequest {
	return NewRequest(url, "HEAD")
}

// GetRequest return the request object
func (b *HTTPRequest) GetRequest() *http.Request {
	return b.req
}

// SetBasicAuth sets the request's Authorization header to use HTTP Basic Authentication with the provided username and password.
func (b *HTTPRequest) SetBasicAuth(username, password string) *HTTPRequest {
	b.req.SetBasicAuth(username, password)
	return b
}

// Retry sets Retries config.
// default is 0 means no retried.
// duration means after time.Sleep(duration) retry start again
// statusCode means when response.StatusCode in statusCode ,retry will work
func (b *HTTPRequest) SetRetry(count int, duration time.Duration, statusCode ...int) *HTTPRequest {
	for _, code := range statusCode {
		statusText := http.StatusText(code)
		if len(statusText) == 0 {
			log.Println("StatusCode '" + strconv.Itoa(code) + "' doesn't exist in http package")
		}
	}

	b.setting.Retry = struct {
		Status   []int
		Duration time.Duration
		Count    int
		Attempt  int
		Enable   bool
	}{
		statusCode,
		duration,
		count,
		0,
		true,
	}
	return b
}

// DumpRequest return the DumpRequest
func (b *HTTPRequest) DumpRequest() []byte {
	return b.dump
}

// Header add header item string in request.
func (b *HTTPRequest) Header(key, value string) *HTTPRequest {
	b.req.Header.Set(key, value)
	return b
}

// SetHost set the request host
func (b *HTTPRequest) SetHost(host string) *HTTPRequest {
	b.req.Host = host
	return b
}

// SetProtocolVersion Set the protocol version for incoming requests.
// Client requests always use HTTP/1.1.
func (b *HTTPRequest) SetProtocolVersion(vers string) *HTTPRequest {
	if len(vers) == 0 {
		vers = "HTTP/1.1"
	}

	major, minor, ok := http.ParseHTTPVersion(vers)
	if ok {
		b.req.Proto = vers
		b.req.ProtoMajor = major
		b.req.ProtoMinor = minor
	}

	return b
}

// SetCookie add cookie into request.
func (b *HTTPRequest) SetCookie(cookie *http.Cookie) *HTTPRequest {
	b.req.Header.Add("Cookie", cookie.String())
	return b
}

// Param adds query param in to request.
// params build query string as ?key1=value1&key2=value2...
func (b *HTTPRequest) Param(key, value string) *HTTPRequest {
	if param, ok := b.params[key]; ok {
		b.params[key] = append(param, value)
	} else {
		b.params[key] = []string{value}
	}
	return b
}

// PostFile add a post file to the request
func (b *HTTPRequest) PostFile(formName, filename string) *HTTPRequest {
	b.files[formName] = filename
	return b
}

// Body adds request raw body.
// it supports string and []byte.
func (b *HTTPRequest) Body(data interface{}) *HTTPRequest {
	switch t := data.(type) {
	case string:
		bf := bytes.NewBufferString(t)
		b.req.Body = ioutil.NopCloser(bf)
		b.req.ContentLength = int64(len(t))
	case []byte:
		bf := bytes.NewBuffer(t)
		b.req.Body = ioutil.NopCloser(bf)
		b.req.ContentLength = int64(len(t))
	}
	return b
}

// JSONBody adds request raw body encoding by JSON.
func (b *HTTPRequest) JSONBody(obj interface{}) (*HTTPRequest, error) {
	if b.req.Body == nil && obj != nil {
		byts, err := json.Marshal(obj)
		if err != nil {
			return b, err
		}
		b.req.Body = ioutil.NopCloser(bytes.NewReader(byts))
		b.req.ContentLength = int64(len(byts))
		b.req.Header.Set("Content-Type", "application/json")
	}
	return b, nil
}

func (b *HTTPRequest) buildURL(paramBody string) {
	// build GET url with query string
	if b.req.Method == "GET" && len(paramBody) > 0 {
		if strings.Contains(b.url, "?") {
			b.url += "&" + paramBody
		} else {
			b.url = b.url + "?" + paramBody
		}
		return
	}

	// build POST/PUT/PATCH url and body
	if (b.req.Method == "POST" || b.req.Method == "PUT" || b.req.Method == "PATCH" || b.req.Method == "DELETE") && b.req.Body == nil {
		// with files
		if len(b.files) > 0 {
			pr, pw := io.Pipe()
			bodyWriter := multipart.NewWriter(pw)
			go func() {
				for formname, filename := range b.files {
					fileWriter, err := bodyWriter.CreateFormFile(formname, filename)
					if err != nil {
						log.Println("Httplib:", err)
					}
					fh, err := os.Open(filename)
					if err != nil {
						log.Println("Httplib:", err)
					}
					// iocopy
					_, err = io.Copy(fileWriter, fh)
					fh.Close()
					if err != nil {
						log.Println("Httplib:", err)
					}
				}
				for k, v := range b.params {
					for _, vv := range v {
						bodyWriter.WriteField(k, vv)
					}
				}
				bodyWriter.Close()
				pw.Close()
			}()
			b.Header("Content-Type", bodyWriter.FormDataContentType())
			b.req.Body = ioutil.NopCloser(pr)
			return
		}

		// with params
		if len(paramBody) > 0 {
			b.Header("Content-Type", "application/x-www-form-urlencoded")
			b.Body(paramBody)
		}
	}
}

func (b *HTTPRequest) getResponse() (*http.Response, error) {
	if b.resp.StatusCode != 0 {
		return b.resp, nil
	}
	resp, err := b.DoRequest()
	if err != nil {
		return nil, err
	}
	b.resp = resp
	return resp, nil
}

// DoRequest will do the client.Do
func (b *HTTPRequest) DoRequest() (resp *http.Response, err error) {
	var paramBody string
	if len(b.params) > 0 {
		var buf bytes.Buffer
		for k, v := range b.params {
			for _, vv := range v {
				buf.WriteString(url.QueryEscape(k))
				buf.WriteByte('=')
				buf.WriteString(url.QueryEscape(vv))
				buf.WriteByte('&')
			}
		}
		paramBody = buf.String()
		paramBody = paramBody[0 : len(paramBody)-1]
	}

	b.buildURL(paramBody)
	url, err := url.Parse(b.url)
	if err != nil {
		return nil, err
	}

	b.req.URL = url

	trans := b.setting.Transport

	if trans == nil {
		// create default transport
		trans = &http.Transport{
			TLSClientConfig:     b.setting.TLSClientConfig,
			Proxy:               b.setting.Proxy,
			DialContext:         TimeoutDialerContext(b.setting.ConnectTimeout, b.setting.ReadWriteTimeout),
			MaxIdleConnsPerHost: -1,
		}
	} else {
		// if b.transport is *http.Transport then set the settings.
		if t, ok := trans.(*http.Transport); ok {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = b.setting.TLSClientConfig
			}
			if t.Proxy == nil {
				t.Proxy = b.setting.Proxy
			}

			if t.DialContext == nil {
				t.DialContext = TimeoutDialerContext(b.setting.ConnectTimeout, b.setting.ReadWriteTimeout)
			}
		}
	}

	var jar http.CookieJar
	if b.setting.EnableCookie {
		if defaultcookieJar == nil {
			createDefaultCookie()
		}
		jar = defaultcookieJar
	}

	client := &http.Client{
		Transport: trans,
		Jar:       jar,
	}

	if b.setting.UserAgent != "" && b.req.Header.Get("User-Agent") == "" {
		b.req.Header.Set("User-Agent", b.setting.UserAgent)
	}

	if b.setting.CheckRedirect != nil {
		client.CheckRedirect = b.setting.CheckRedirect
	}

	if b.setting.EnableDumpBody {
		dump, err := httputil.DumpRequest(b.req, b.setting.EnableDumpBody)
		if err != nil {
			log.Println(err.Error())
		}
		b.dump = dump
	}

	if b.setting.EnableDebug {
		curl, _ := b.getCurlCommand()
		log.Println(curl)
	}

	// retry default is disabled, it will run once.
	// if retry is setted, it will check response status code , if the code  in retry.Status
	// retry will run again ,until retry.Attempt  equal to retry.Count
	for {
		resp, err = client.Do(b.req)

		if err != nil {
			return resp, err
		}

		retry := &b.setting.Retry

		if retry.Enable && retry.Attempt < retry.Count && contains(resp.StatusCode, retry.Status) {
			time.Sleep(retry.Duration)
			retry.Attempt++
			resp.Header.Set("Retry-Count", strconv.Itoa(retry.Attempt))
			log.Println("retry ", b.req.Method, b.url, resp.StatusCode)
			continue
		}

		break
	}
	return resp, err
}

func contains(respStatus int, statuses []int) bool {
	for _, status := range statuses {
		if status == respStatus {
			return true
		}
	}
	return false
}

// Response executes request client gets response mannually.
func (b *HTTPRequest) Response() (*http.Response, error) {
	return b.getResponse()
}

// TimeoutDialer returns functions of connection dialer with timeout settings for http.Transport Dial field.
func TimeoutDialerContext(cTimeout time.Duration, rwTimeout time.Duration) func(ctx context.Context, net, addr string) (net.Conn, error) {
	return func(ctx context.Context, netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		err = conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, err
	}
}

// nopCloser is used to create a new io.ReadCloser for req.Body
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
