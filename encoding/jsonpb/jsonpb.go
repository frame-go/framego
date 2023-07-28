package jsonpb

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/frame-go/framego/utils"
)

var jpb *jsonpbCodec

func initJsonpb() {
	c := &jsonpbCodec{}
	c.Marshaler.AllowPartial = true
	c.Marshaler.UseProtoNames = true
	c.Marshaler.UseEnumNumbers = true
	c.Marshaler.EmitUnpopulated = true
	c.Unmarshaler.AllowPartial = true
	c.Unmarshaler.DiscardUnknown = true
	jpb = c
}

func Marshal(v interface{}) ([]byte, error) {
	return jpb.Marshal(v)
}

func MarshalString(v interface{}) (string, error) {
	return jpb.MarshalString(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return jpb.Unmarshal(data, v)
}

func UnmarshalString(data string, v interface{}) error {
	return jpb.UnmarshalString(data, v)
}

type jsonpbCodec struct {
	Marshaler   protojson.MarshalOptions
	Unmarshaler protojson.UnmarshalOptions
}

func (c *jsonpbCodec) Marshal(v interface{}) ([]byte, error) {
	if pm, ok := v.(proto.Message); ok {
		return c.Marshaler.Marshal(pm)
	}
	return json.Marshal(v)
}

func (c *jsonpbCodec) MarshalString(v interface{}) (string, error) {
	if pm, ok := v.(proto.Message); ok {
		b, err := c.Marshaler.Marshal(pm)
		if err != nil {
			return "", err
		}
		return utils.BytesToString(b), nil
	}
	bs, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return utils.BytesToString(bs), err
}

func (c *jsonpbCodec) Unmarshal(data []byte, v interface{}) error {
	if pm, ok := v.(proto.Message); ok {
		return c.Unmarshaler.Unmarshal(data, pm)
	}
	return json.Unmarshal(data, v)
}

func (c *jsonpbCodec) UnmarshalString(data string, v interface{}) error {
	if pm, ok := v.(proto.Message); ok {
		return c.Unmarshaler.Unmarshal(utils.StringToBytes(data), pm)
	}
	return json.Unmarshal(utils.StringToBytes(data), v)
}

func init() {
	initJsonpb()
}
