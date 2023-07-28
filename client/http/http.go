package http

import (
	"net"
	nethttp "net/http"
	"time"
)

// Method represents http method in string
type Method string

// http methods
const (
	GET  Method = "GET"
	POST Method = "POST"
)

type requestDataFormat int

// request data formats
const (
	NoData      requestDataFormat = 0
	QueryString requestDataFormat = 1
	Form        requestDataFormat = 2
	JSON        requestDataFormat = 3
	Raw         requestDataFormat = 4
)

// DataMap is a map of key to any data
type DataMap map[string]interface{}

// Client is the http client standard interface.
type Client interface {
	// RequestJSON supports different request format with JSON Response
	RequestJSON(respData interface{}, method Method, url string, format requestDataFormat, data interface{}, headers ...*DataMap) (err error)

	// GetRawClient returns the internal http client
	GetRawClient() interface{}
}

var transport *nethttp.Transport
var c Client

// SetClient sets http client
func SetClient(client Client) {
	c = client
}

func init() {
	transport = &nethttp.Transport{
		Proxy: nethttp.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	c = NewDefaultClient()
}

// GetJSON send GET request with data in JSON format, using `DefaultClient`
func GetJSON(respData interface{}, url string, params interface{}, headers ...*DataMap) error {
	return c.RequestJSON(respData, GET, url, QueryString, params, headers...)
}

// PostForm send POST request with data in Form format, using `DefaultClient`
func PostForm(respData interface{}, url string, form interface{}, headers ...*DataMap) error {
	return c.RequestJSON(respData, POST, url, Form, form, headers...)
}

// PostJSON send POST request with data in JSON format, using `DefaultClient`
func PostJSON(respData interface{}, url string, data interface{}, headers ...*DataMap) error {
	return c.RequestJSON(respData, POST, url, JSON, data, headers...)
}

// RequestJSON send http request with JSON response, support both GET and POST, different body format and headers, using `DefaultClient`
func RequestJSON(respData interface{}, method Method, url string, format requestDataFormat, data interface{}, headers ...*DataMap) error {
	return c.RequestJSON(respData, method, url, format, data, headers...)
}
