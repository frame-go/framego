package http

import (
	"github.com/valyala/fasthttp"
	nethttp "net/http"
	"testing"
	"time"
)

func TestGetClient(t *testing.T) {
	var c Client
	c = GetClient(RawClientType(ClientNetHTTP), Timeout(2*time.Second))

	var netHTTPClient *nethttp.Client
	var ok bool
	netHTTPClient, ok = c.GetRawClient().(*nethttp.Client)
	if !ok {
		t.Errorf("expect to have nethttp.Client as netHTTPClient, but not")
	}
	if netHTTPClient.Timeout != 2*time.Second {
		t.Errorf("fail to set timeout")
	}
	if _, err := netHTTPClient.Get("https://postman-echo.com/get"); err != nil {
		t.Errorf("fail to invoke Get")
	}

	c = GetClient(Timeout(time.Second), RawClientType(ClientFastHTTP))
	var fastHTTPClient *fasthttp.Client
	fastHTTPClient, ok = c.GetRawClient().(*fasthttp.Client)
	if !ok {
		t.Errorf("expect to have nethttp.Client as fasthttp client, but not")
	}
	if fastHTTPClient.ReadTimeout != time.Second {
		t.Errorf("fail to set timeout")
	}
	if _, _, err := fastHTTPClient.Get(make([]byte, 10), "https://postman-echo.com/get"); err != nil {
		t.Errorf("fail to invoke Get")
	}
}

func TestFastHTTPClient_RequestJSON(t *testing.T) {
	c := GetClient(RawClientType(ClientFastHTTP))
	resp := map[string]interface{}{}
	var err error
	err = c.RequestJSON(&resp, GET, "https://postman-echo.com/get?data=1", NoData, nil)
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("resp: %v\n", resp)

	listResp := map[string]interface{}{}
	postData := &DataMap{
		"message": []map[string]string{
			{"dada1": "test-1"},
			{"data2": "test-2"},
		},
	}
	err = c.RequestJSON(&listResp, POST, "https://postman-echo.com/post", JSON, postData)
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("resp: %v\n", listResp)
}
