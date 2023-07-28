package http

import (
	"io/ioutil"
)

// GetMockFileNameFunc returns a file name, of which contains mock data for test
type GetMockFileNameFunc func(url string, method Method, data interface{}) string

// DefaultMockClient read json content from mock file and unmarshal it to response.
type DefaultMockClient struct {
	getMockData GetMockFileNameFunc
}

// RequestJSON mocks RequestJSON in http but returns mock data
func (hcm DefaultMockClient) RequestJSON(respData interface{}, method Method, url string, format requestDataFormat, data interface{}, headers ...*DataMap) (err error) {
	mockDataFileName := hcm.getMockData(url, method, data)
	file, _ := ioutil.ReadFile(mockDataFileName)
	err = json.Unmarshal([]byte(file), respData)
	return err
}

// GetRawClient gets the internal client
func (hcm DefaultMockClient) GetRawClient() interface{} {
	return nil
}

// NewDefaultHTTPMockClient creates a http client for testing
func NewDefaultHTTPMockClient(getFileFunc GetMockFileNameFunc) Client {
	return DefaultMockClient{
		getMockData: getFileFunc,
	}
}
