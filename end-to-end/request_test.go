package end_to_end

import (
	"fmt"
	"testing"

	httpkit "github.com/weibaohui/http-request"
)

func TestPost(t *testing.T) {
	resp, err := httpkit.Get("http://baidu.com").Execute()
	if err != nil {
		fmt.Println(err.Error())
		t.Fail()
	}
	str, err := resp.String()
	fmt.Println(str)
	headers := resp.Headers()
	fmt.Println(headers)
	println(resp.Header("Content-Length"))
}
