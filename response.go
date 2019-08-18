package httpkit

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type HTTPResponse struct {
	setting *HTTPSettings
	resp    *http.Response
	body    []byte
}

func (b *HTTPRequest) Execute() (*HTTPResponse, error) {
	response, err := b.getResponse()
	httpResponse := &HTTPResponse{
		resp:    response,
		setting: &b.setting,
	}
	return httpResponse, err
}
func (r *HTTPResponse) Bytes() ([]byte, error) {
	resp := r.resp
	if resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()
	if r.setting.Gzip && resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		r.body, err = ioutil.ReadAll(reader)
		return r.body, err
	}
	var err error
	r.body, err = ioutil.ReadAll(resp.Body)
	return r.body, err
}
func (r *HTTPResponse) String() (string, error) {
	data, err := r.Bytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToFile saves the body data in response to one file.
// it calls Response inner.
func (r *HTTPResponse) ToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if r.resp.Body == nil {
		return nil
	}
	defer r.resp.Body.Close()
	_, err = io.Copy(f, r.resp.Body)
	return err
}

// ToJSON returns the map that marshals from the body bytes as json in response .
// it calls Response inner.
func (r *HTTPResponse) ToJSON(v interface{}) error {
	data, err := r.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ToXML returns the map that marshals from the body bytes as xml in response .
// it calls Response inner.
func (r *HTTPResponse) ToXML(v interface{}) error {
	data, err := r.Bytes()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}
func (r *HTTPResponse) Headers() http.Header {
	return r.resp.Header
}
func (r *HTTPResponse) Header(h string) string {
	return r.resp.Header.Get(h)
}
func (r *HTTPResponse) StatusCode() int {
	return r.resp.StatusCode
}
func (r *HTTPResponse) Cookies() []*http.Cookie {
	return r.resp.Cookies()
}
func (r *HTTPResponse) Response() *http.Response {
	return r.resp
}
