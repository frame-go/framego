package grpcex

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	// JsonCodecName is the name registered for the json encoder.
	JsonCodecName = "json"
)

// codec implements encoding.Codec to encode messages into JSON.
type codec struct {
	Marshaler   protojson.MarshalOptions
	Unmarshaler protojson.UnmarshalOptions
}

// Marshal marshals "v" into JSON.
func (c *codec) Marshal(v interface{}) ([]byte, error) {
	if pm, ok := v.(proto.Message); ok {
		return c.Marshaler.Marshal(pm)
	}
	return json.Marshal(v)
}

// Unmarshal unmarshals JSON-encoded data into "v".
func (c *codec) Unmarshal(data []byte, v interface{}) error {
	if pm, ok := v.(proto.Message); ok {
		return c.Unmarshaler.Unmarshal(data, pm)
	}
	return json.Unmarshal(data, v)
}

// Name returns the identifier of the codec.
func (c *codec) Name() string {
	return JsonCodecName
}

func RegisterJsonCodec() {
	c := &codec{}
	c.Marshaler.AllowPartial = true
	c.Marshaler.UseProtoNames = true
	c.Marshaler.UseEnumNumbers = true
	c.Marshaler.EmitUnpopulated = true
	c.Unmarshaler.AllowPartial = true
	c.Unmarshaler.DiscardUnknown = true
	encoding.RegisterCodec(c)
}
