// Package utils provides a set of common utilities, used thoughout the project
package utils

import "reflect"

// GetTypeName returns type's name as string
func GetTypeName(v interface{}) string {
	return reflect.TypeOf(v).String()
}
