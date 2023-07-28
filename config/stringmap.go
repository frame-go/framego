package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator"
	"github.com/yalp/jsonpath"

	"github.com/frame-go/framego/copy"
	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/utils"
)

// TypeAssertionError describes type assertion error, used to generate type convertion errors
type TypeAssertionError struct {
	Value      interface{}
	TargetType string
}

// Error returns type convertion error
func (e TypeAssertionError) Error() string {
	return fmt.Sprintf("type_assertion_error:%s->%s", utils.GetTypeName(e.Value), e.TargetType)
}

// StringMap defines config map type
type StringMap map[string]interface{}

// ToRawMap converts config to raw map[string]interface{}
func (m StringMap) ToRawMap() map[string]interface{} {
	var target map[string]interface{}
	target = m
	return target
}

// ToStruct converts config to generic interface, copying all config values
func (m StringMap) ToStruct(targetObj interface{}) error {
	return copy.JsonDeepCopy(targetObj, m)
}

// ToStructWithValidation converts config object to config map with field values validation
func (m StringMap) ToStructWithValidation(targetObj interface{}) error {
	err := copy.JsonDeepCopy(targetObj, m)
	if err != nil {
		return errors.Wrap(err, "deep_copy_config_error")
	}
	validate := validator.New()
	err = validate.Struct(targetObj)
	if err != nil {
		return errors.Wrap(err, "config_object_validation_error")
	}
	return nil
}

// GetString returns a string value of config by path
func (m StringMap) GetString(path string) (string, error) {
	path = regularizePath(path)
	rawValue, err := jsonpath.Read(m.ToRawMap(), path)
	if err != nil {
		return "", err
	}
	value, ok := rawValue.(string)
	if !ok {
		return "", TypeAssertionError{rawValue, "string"}
	}
	return value, err
}

// GetInt returns an int value of config by path
func (m StringMap) GetInt(path string) (int, error) {
	path = regularizePath(path)
	rawValue, err := jsonpath.Read(m.ToRawMap(), path)
	if err != nil {
		return 0, err
	}
	value, ok := rawValue.(int)
	if !ok {
		return 0, TypeAssertionError{rawValue, "int"}
	}
	return value, err
}

// GetInt64 returns an int value of config by path
func (m StringMap) GetInt64(path string) (int64, error) {
	path = regularizePath(path)
	rawValue, err := jsonpath.Read(m.ToRawMap(), path)
	if err != nil {
		return 0, err
	}
	value, ok := rawValue.(int64)
	if !ok {
		return 0, TypeAssertionError{rawValue, "int64"}
	}
	return value, err
}

// GetBool returns an bool value of config by path
func (m StringMap) GetBool(path string) (bool, error) {
	path = regularizePath(path)
	rawValue, err := jsonpath.Read(m.ToRawMap(), path)
	if err != nil {
		return false, err
	}
	value, ok := rawValue.(bool)
	if !ok {
		return false, TypeAssertionError{rawValue, "bool"}
	}
	return value, err
}

// GetStringMap returns a map value of config by path
func (m StringMap) GetStringMap(path string) (StringMap, error) {
	path = regularizePath(path)
	rawValue, err := jsonpath.Read(m.ToRawMap(), path)
	if err != nil {
		return nil, err
	}
	value, ok := rawValue.(map[string]interface{})
	if !ok {
		return nil, TypeAssertionError{rawValue, "StringMap"}
	}
	return value, err
}

// GetStringMapList returns a list value of config by path
func (m StringMap) GetStringMapList(path string) ([]StringMap, error) {
	path = regularizePath(path)
	rawValue, err := jsonpath.Read(m.ToRawMap(), path)
	if err != nil {
		return nil, err
	}
	rawList, ok := rawValue.([]interface{})
	if !ok {
		return nil, TypeAssertionError{rawValue, "[]StringMap"}
	}
	configList := make([]StringMap, 0, len(rawList))
	for _, rawItem := range rawList {
		configItem, ok := rawItem.(map[string]interface{})
		if !ok {
			return nil, TypeAssertionError{rawItem, "StringMap"}
		}
		configList = append(configList, configItem)
	}
	return configList, err
}

// GetStruct returns a struct value of config by path
func (m StringMap) GetStruct(path string, value interface{}) error {
	path = regularizePath(path)
	rawValue, err := jsonpath.Read(m.ToRawMap(), path)
	if err != nil {
		return err
	}
	mapValue, ok := rawValue.(map[string]interface{})
	if !ok {
		return TypeAssertionError{rawValue, "StringMap"}
	}
	var configMapValue StringMap = mapValue
	return configMapValue.ToStruct(value)
}

// GetStructWithValidation returns a struct value of config by path with validation
func (m StringMap) GetStructWithValidation(path string, value interface{}) error {
	err := GetStruct(path, value)
	if err != nil {
		return err
	}
	validate := validator.New()
	err = validate.Struct(value)
	if err != nil {
		return errors.Wrap(err, "config_object_validation_error")
	}
	return nil
}

// Mix mixes in src config to dst config, overriding collisions in dst config map
func (m StringMap) Mix(src StringMap) {
	for k, v := range src {
		m[k] = v
	}
}

func regularizePath(path string) string {
	if strings.HasPrefix(path, "$") {
		return path
	}
	return "$." + path
}
