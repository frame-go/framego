package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	nethttp "net/http"
	neturl "net/url"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	formTagName = "form"
)

// client type
const (
	ClientNetHTTP = iota + 1
	ClientFastHTTP
)

// DefaultClient is the client to handle the various HTTP request, with net/http.Client as internal client
type DefaultClient struct {
	// May add more functions here in the future
	internal *nethttp.Client
}

// FastHTTPClient is the client to handle the various HTTP request, with net/fasthttp.Client as internal client
type FastHTTPClient struct {
	internal *fasthttp.Client
}

// Options are options to create a client
type Options struct {
	Timeout    time.Duration
	ClientType int
}

// Option is modifier to update Options
type Option func(*Options)

// Timeout sets `Options.Timeout`
func Timeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.Timeout = timeout
	}
}

// RawClientType sets `Options.ClientType`
func RawClientType(clientType int) Option {
	return func(options *Options) {
		if clientType < ClientNetHTTP || clientType > ClientFastHTTP {
			return
		}
		options.ClientType = clientType
	}
}

// NewDefaultClient initializes a new DefaultClient
func NewDefaultClient() Client {
	return GetClient()
}

// GetClient creates a `Client` based on `opts` and returns to caller
func GetClient(opts ...Option) Client {
	options := &Options{
		Timeout:    10 * time.Second,
		ClientType: ClientNetHTTP,
	}
	for _, f := range opts {
		f(options)
	}
	if options.ClientType == ClientNetHTTP {
		return &DefaultClient{
			internal: &nethttp.Client{
				Transport: transport,
				Timeout:   options.Timeout,
			},
		}
	} else if options.ClientType == ClientFastHTTP {
		return &FastHTTPClient{
			internal: &fasthttp.Client{
				ReadTimeout:     options.Timeout,
				WriteTimeout:    options.Timeout,
				MaxConnsPerHost: 100,
				Dial: func(addr string) (conn net.Conn, e error) {
					return transport.DialContext(context.Background(), "tcp", addr)
				},
			},
		}
	}
	return nil
}

func isPublicField(fieldType reflect.StructField) bool {
	return unicode.IsUpper(rune(fieldType.Name[0]))
}

func setFormValue(form *neturl.Values, key string, value interface{}) {
	obj := reflect.ValueOf(value)
	switch obj.Kind() {
	case reflect.Ptr:
		form.Set(key, fmt.Sprintf("%v", obj.Elem()))
	case reflect.Slice, reflect.Array:
		for i := 0; i < obj.Len(); i++ {
			item := obj.Index(i)
			if item.Kind() == reflect.Ptr {
				form.Add(key, fmt.Sprintf("%v", item.Elem()))
			} else {
				form.Add(key, fmt.Sprintf("%v", item))
			}
		}
	default:
		form.Set(key, fmt.Sprintf("%v", value))
	}
}

type headerSetter interface {
	Add(key string, value string)
}

func setHeader(headerSetter headerSetter, value interface{}, key string) (err error) {
	obj := reflect.ValueOf(value)
	switch obj.Kind() {
	case reflect.Ptr:
		if obj.Elem().Kind() != reflect.Struct {
			return setHeader(headerSetter, obj.Elem(), key)
		}
		s := ""
		s, err = json.MarshalToString(value)
		if err != nil {
			return
		}
		headerSetter.Add(key, s)
	case reflect.Slice, reflect.Array:
		for i := 0; i < obj.Len(); i++ {
			err = setHeader(headerSetter, obj.Index(i).Interface(), key)
			if err != nil {
				return
			}
		}
	case reflect.Struct:
		s := ""
		s, err = json.MarshalToString(value)
		if err != nil {
			return
		}
		headerSetter.Add(key, s)
	case reflect.Map:
		s := ""
		s, err = json.MarshalToString(value)
		if err != nil {
			return
		}
		headerSetter.Add(key, s)
	default:
		headerSetter.Add(key, fmt.Sprintf("%v", value))
	}
	return
}

func getBodyReaderAndForm(format requestDataFormat, data interface{}) (io.Reader, *neturl.Values, error) {
	var reqBody io.Reader
	var s string
	var ok bool
	var err error
	var form *neturl.Values
	if data != nil {
		if format == JSON {
			s, err = json.MarshalToString(data)
			if err != nil {
				return nil, nil, err
			}
			reqBody = strings.NewReader(s)
		} else if format == QueryString || format == Form {
			var urlValues neturl.Values
			var pDataMap *DataMap
			var dataMap DataMap
			if data == nil {
				// Nil
			} else if form, ok = data.(*neturl.Values); ok {
				// data is *url.Values
			} else if urlValues, ok = data.(neturl.Values); ok {
				// data is url.Values
				form = &urlValues
			} else if pDataMap, ok = data.(*DataMap); ok {
				// data is *DataMap
				form = &neturl.Values{}
				for k, v := range *pDataMap {
					setFormValue(form, k, v)
				}
			} else if dataMap, ok = data.(DataMap); ok {
				// data is DataMap
				form = &neturl.Values{}
				for k, v := range dataMap {
					setFormValue(form, k, v)
				}
			} else {
				// data is struct
				form = &neturl.Values{}
				obj := reflect.ValueOf(data)
				if obj.Kind() == reflect.Ptr {
					obj = obj.Elem()
				}
				if obj.Kind() != reflect.Struct {
					return nil, nil, fmt.Errorf("unsupport_data_type:%v", obj.Kind())
				}
				t := obj.Type()
				fn := obj.NumField()
				var fieldName string
				for i := 0; i < fn; i++ {
					ft := t.Field(i)
					if isPublicField(ft) {
						fieldName, ok = ft.Tag.Lookup(formTagName)
						if !ok {
							fieldName = ft.Name
						}
						setFormValue(form, fieldName, obj.Field(i).Interface())
					}
				}
			}
			if form != nil {
				if format == Form {
					reqBody = strings.NewReader(form.Encode())
				}
			}
		} else if format == Raw {
			if s, ok := data.(string); ok {
				reqBody = strings.NewReader(s)
			} else if s, ok := data.(*string); ok {
				if s != nil {
					reqBody = strings.NewReader(*s)
				}
			} else if b, ok := data.([]byte); ok {
				reqBody = bytes.NewBuffer(b)
			} else if b, ok := data.(*[]byte); ok {
				if b != nil {
					reqBody = bytes.NewBuffer(*b)
				}
			} else {
				obj := reflect.ValueOf(data)
				return nil, nil, fmt.Errorf("unsupported_data_type_for_Raw:%v", obj.Kind())
			}
		}
	}
	return reqBody, form, nil
}

// RequestJSON Send http request to query JSON data
// `data` can be `*url.Values`, `*DataMap`, or struct, for any requestDataFormat
//
// Note:
// 1. if pass `url.Values` or `DataMap` as data, should pass the pointer
// 2. For struct data, can specify the field name by `form` tag. E.g. struct { Name string `form:"name"` }
//
// Examples:
//
//  0. Response struct
//
//     ```
//     type ApiResponse struct {
//     Error string    `json:"error"`
//     }
//     resp := ApiResponse{}
//     ```
//
//  1. Pass *url.Values as parameters
//
//     ```
//     v := url.Values{}
//     v.Set("name", "abc")
//     v.Add("values", "1")
//     v.Add("values", "2")
//     http.HTTPRequestJSON(resp, "GET", "http://localhost/api", http.QueryString, &v)
//     ```
//
// The request will be:
//
//		```
//		GET /api?name=abc&values=1&values=2 HTTP/1.1
//		Host: localhost
//		```
//
//	 2. Pass DataMap as parameters
//
//	    ```
//	    http.HTTPRequestJSON(resp, "POST", "http://localhost/api", http.Form,
//	    &http.DataMap{
//	    	"name": "abc",
//	    	"values": []int{1, 2},
//	    })
//	    ```
//
// The request will be:
//
//		```
//		POST /api HTTP/1.1
//		Host: localhost
//		Content-Type: application/x-www-form-urlencoded
//
//		name=abc&values=1&values=2
//		```
//
//	 3. Pass Struct as parameters
//
//	    ```
//	    type ApiRequest struct {
//	    	Name string `json:"name" form:"name"`
//	    	Values []int `json:"values" form:"values"`
//	    }
//	    http.HTTPRequestJSON(resp, "POST", "http://localhost/api", http.JSON,
//	    &ApiRequest{
//	    	Name: "abc",
//	    	Values: []int{1, 2},
//	    })
//	    ```
//
// The request will be:
//
//	```
//	POST /api HTTP/1.1
//	Host: localhost
//	Content-Type: application/json
//
//	{"name":"abc","values":[1,2]}
//	```
func (c *DefaultClient) RequestJSON(respData interface{}, method Method, url string, format requestDataFormat, data interface{}, headers ...*DataMap) (err error) {
	reqBody, form, err := getBodyReaderAndForm(format, data)
	if err != nil {
		return err
	}
	var req *nethttp.Request
	req, err = nethttp.NewRequest(string(method), url, reqBody)
	if format == QueryString {
		if form != nil {
			req.URL.RawQuery = form.Encode()
		}
	} else if format == Form {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if format == JSON {
		req.Header.Set("Content-Type", "application/json")
	}

	// add custom headers
	for _, h := range headers {
		for k, v := range *h {
			err := setHeader(req.Header, v, k)
			if err != nil {
				return err
			}
		}
	}

	var resp *nethttp.Response
	resp, err = c.internal.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if resp.StatusCode != nethttp.StatusOK {
		return fmt.Errorf("http_error_status_code:%d", resp.StatusCode)
	}
	if respData == nil {
		return nil
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, respData)
	if err != nil {
		return err
	}
	return nil
}

// RequestJSON Send http request to query JSON data
func (c *FastHTTPClient) RequestJSON(respData interface{}, method Method, url string, format requestDataFormat, data interface{}, headers ...*DataMap) (err error) {
	reqBody, form, err := getBodyReaderAndForm(format, data)
	if err != nil {
		return err
	}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(string(method))
	req.SetRequestURI(url)
	if format == QueryString {
		if form != nil {
			req.SetRequestURI(fmt.Sprintf("%s?%s", url, form.Encode()))
		}
	}
	if string(method) != "GET" && string(method) != "HEAD" {
		req.SetBodyStream(reqBody, -1)
	}
	u, _ := neturl.Parse(url)
	req.SetHost(u.Host)

	if format == QueryString {
		if form != nil {
			req.SetRequestURI(fmt.Sprintf("%s?%s", url, form.Encode()))
		}
	} else if format == Form {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if format == JSON {
		req.Header.Set("Content-Type", "application/json")
	}

	// add custom headers
	for _, h := range headers {
		for k, v := range *h {
			err := setHeader(&req.Header, v, k)
			if err != nil {
				return err
			}
		}
	}

	err = c.internal.DoTimeout(req, resp, c.internal.ReadTimeout)
	if err != nil {
		return err
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("http_error_status_code:%d", resp.StatusCode())
	}
	body := resp.Body()
	err = json.Unmarshal(body, respData)
	if err != nil {
		return err
	}
	return nil
}

// GetRawClient gets the internal client
func (c *DefaultClient) GetRawClient() interface{} {
	return c.internal
}

// GetRawClient gets the internal client
func (c *FastHTTPClient) GetRawClient() interface{} {
	return c.internal
}
