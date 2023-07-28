package cache

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"strconv"

	"github.com/frame-go/framego/utils"
)

// Serialize returns a []byte representing the passed value
func Serialize(value any) ([]byte, error) {
	if b, ok := value.([]byte); ok {
		return b, nil
	}

	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return utils.StringToBytes(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return utils.StringToBytes(strconv.FormatUint(v.Uint(), 10)), nil
	case reflect.Float32, reflect.Float64:
		return utils.StringToBytes(strconv.FormatFloat(v.Float(), 'g', -1, 64)), nil
	case reflect.String:
		return utils.StringToBytes(v.String()), nil
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return v.Bytes(), nil
		}
	}

	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Deserialize deserializes the passed []byte into the passed ptr any
func Deserialize(byt []byte, ptr any) (err error) {
	if b, ok := ptr.(*[]byte); ok {
		*b = byt
		return nil
	}

	if v := reflect.ValueOf(ptr); v.Kind() == reflect.Ptr {
		switch p := v.Elem(); p.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var i int64
			i, err = strconv.ParseInt(string(byt), 10, 64)
			if err != nil {
				return err
			}
			p.SetInt(i)
			return nil

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var i uint64
			i, err = strconv.ParseUint(string(byt), 10, 64)
			if err != nil {
				return err
			}
			p.SetUint(i)
			return nil

		case reflect.Float32, reflect.Float64:
			var f float64
			f, err = strconv.ParseFloat(string(byt), 64)
			if err != nil {
				return err
			}
			p.SetFloat(f)
			return nil

		case reflect.String:
			p.SetString(utils.BytesToString(byt))
			return nil

		case reflect.Slice:
			if p.Type().Elem().Kind() == reflect.Uint8 {
				p.SetBytes(byt)
				return nil
			}
		}
	}

	b := bytes.NewBuffer(byt)
	decoder := gob.NewDecoder(b)
	if err = decoder.Decode(ptr); err != nil {
		return err
	}
	return nil
}
