package http

import (
	"net/http"
	"strings"
	"testing"
)

type BannerResponse struct {
	Banners []map[string]interface{}
	Error   string `json:"error"`
}

func TestGetJSON(t *testing.T) {
	resp := BannerResponse{}
	err := GetJSON(&resp, "https://postman-echo.com/get?data=test", nil)
	t.Logf("response: %v\n", resp)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestRequestJSON(t *testing.T) {
	resp := BannerResponse{}
	var err error

	err = GetJSON(&resp, "https://postman-echo.com/get?data=test", nil)
	t.Logf("response: %v\n", resp)
	if err != nil {
		t.Error(err.Error())
	}

	fastHTTPClient := GetClient(RawClientType(ClientFastHTTP))
	err = fastHTTPClient.RequestJSON(&resp, http.MethodGet, "https://postman-echo.com/get", QueryString, &DataMap{"data": "test-1"})
	t.Logf("response: %v\n", resp)
	if err != nil {
		t.Error(err.Error())
	}

	listResp := map[string]interface{}{}
	postData := &DataMap{
		"message": []map[string]string{
			{"dada1": "test-1"},
			{"data2": "test-2"},
		},
	}
	err = fastHTTPClient.RequestJSON(&listResp, http.MethodPost, "https://postman-echo.com/post", JSON, postData)
	t.Logf("response: %v\n", listResp)
	if err != nil {
		t.Error(err.Error())
	}

	err = fastHTTPClient.RequestJSON(&listResp, http.MethodGet, "https://postman-echo.com/status/406", QueryString, nil)
	if err == nil || !strings.Contains(err.Error(), "406") {
		t.Errorf("Expect to have http 406")
	}

}

func TestHttpJsonWithHeaders(t *testing.T) {
	resp := BannerResponse{}
	header := &DataMap{"Test-Header-Str": "Test_Value", "Test-Header-Obj": &struct{ AAA string }{AAA: "field_one"}}

	err := GetJSON(&resp, "https://postman-echo.com/get", nil, header)
	t.Logf("response: %v\n", resp)
	if err != nil {
		t.Error(err.Error())
	}

}

func TestSetHeader(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://www.google.com/search?q=foo&q=bar&both=x&prio=1&orphan=nope&empty=not",
		strings.NewReader("z=post&both=y&prio=2&=nokey&orphan;empty=&"))
	headers := &DataMap{
		"Test-Header-Str":        "Test_Value",
		"Test-Header-Int":        123,
		"Test-Header-Obj":        &struct{ AAA string }{AAA: "field_one"},
		"Test-Header-Arr":        []interface{}{"hello", "world", 123},
		"Test-Header-Map":        map[string]string{"aaa": "bbb", "ccc": "ddd"},
		"Test-Header-Map-In-Arr": []map[string]string{{"aaa": "bbb", "ccc": "ddd"}, {"aaa": "bbb", "ccc": "ddd"}},
	}

	expected := map[string]interface{}{
		"Test-Header-Str":        "Test_Value",
		"Test-Header-Int":        "123",
		"Test-Header-Obj":        `{"AAA":"field_one"}`,
		"Test-Header-Arr":        "hello world 123",
		"Test-Header-Map":        `{"aaa":"bbb","ccc":"ddd"}`,
		"Test-Header-Map-In-Arr": `{"aaa":"bbb","ccc":"ddd"} {"aaa":"bbb","ccc":"ddd"}`,
	}

	for k, v := range *headers {
		err := setHeader(req.Header, v, k)
		if err != nil {
			t.Error(err.Error())
		}
	}

	for k, expectedVal := range expected {
		if strings.Join(req.Header[k], " ") != expectedVal {
			t.Errorf("fail to set header [%s], actual value=%v", k, req.Header[k])

		}
	}
}

func TestMain(m *testing.M) {
	m.Run()
}
