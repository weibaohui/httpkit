package httpkit

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

func bashEscape(str string) string {
	return `'` + strings.Replace(str, `'`, `'\''`, -1) + `'`
}

// CurlCommand contains exec.Command compatible slice + helpers
type curlCommand struct {
	slice []string
}

// append appends a string to the CurlCommand
func (c *curlCommand) append(newSlice ...string) {
	c.slice = append(c.slice, newSlice...)
}

// String returns a ready to copy/paste command
func (c *curlCommand) String() string {
	return strings.Join(c.slice, " ")
}

// GetCurlCommand returns a CurlCommand corresponding to an http.Request
func (b *HTTPRequest) getCurlCommand() (*curlCommand, error) {
	command := curlCommand{}

	command.append("curl")

	command.append("-X", bashEscape(b.req.Method))

	if b.req.Body != nil {
		body, err := ioutil.ReadAll(b.req.Body)
		if err != nil {
			return nil, err
		}
		b.req.Body = nopCloser{bytes.NewBuffer(body)}
		bodyEscaped := bashEscape(string(body))
		command.append("-d", bodyEscaped)
	}

	var keys []string

	for k := range b.req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		command.append("-H", bashEscape(fmt.Sprintf("%s: %s", k, strings.Join(b.req.Header[k], " "))))
	}

	command.append(bashEscape(b.req.URL.String()))

	return &command, nil
}
