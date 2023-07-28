package utils

import (
	"reflect"
	"unsafe"
)

// BytesToString converts byte slice to string without copy.
// Only use this function when you fully understand how it works.
func BytesToString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

// StringToBytes converts string to byte slice without copy.
// Note the byte slice is read only and should not be modified after conversion.
// The lifecycle of returned bytes depends on input string.
// Only use this function when you fully understand how it works.
func StringToBytes(s string) []byte {
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	var b []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh.Data = hdr.Data
	sh.Len = hdr.Len
	sh.Cap = hdr.Len
	return b
}
