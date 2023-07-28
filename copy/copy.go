package copy

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/pquerna/ffjson/ffjson"
	sMsgpack "github.com/shamaton/msgpack"
	"github.com/vmihailenco/msgpack"

	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/pool"
)

const (
	errorNilPointer = "src_or_dst_cannot_be_nil"
	errorEncoding   = "encoding_failed_during_deep_copy"
	errorDecoding   = "decoding_failed_during_deep_copy"
)

// Notes on deep copy methods:
// 1. always need to pass Ptr for dst, otherwise copy will fail.
// 2. dst need to be a Ptr to be concrete type, instead of interface{}, for example, if dst = &map[string]interface{},
// reflect.DeepEqual(dst, src) will fail, since json serialization does not hold type info.
// 3. support deep copy of slice, map and struct.
// 4. for struct, only supports exported fields.

// DeepCopy deep copies the source to destination
func DeepCopy(dst interface{}, src interface{}) error {
	return GobDeepCopy(dst, src)
}

// JsonDeepCopy default encoding/json deep copy
func JsonDeepCopy(dst interface{}, src interface{}) error {
	if dst == nil || src == nil {
		return errors.New(errorNilPointer)
	}

	encodedBytes, err := json.Marshal(src)
	if err != nil {
		return errors.Wrap(err, errorEncoding)
	}
	if err := json.Unmarshal(encodedBytes, dst); err != nil {
		return errors.Wrap(err, errorDecoding)
	}

	return nil
}

// default encoding/gob deep copy
var gobCodecPool *pool.Pool

type gobCodec struct {
	enc    *gob.Encoder
	dec    *gob.Decoder
	buffer *bytes.Buffer
}

func newGobCodec() *gobCodec {
	c := &gobCodec{
		buffer: &bytes.Buffer{},
	}
	c.enc = gob.NewEncoder(c.buffer)
	c.dec = gob.NewDecoder(c.buffer)
	return c
}

func (c *gobCodec) Close() (err error) {
	return
}

func GobDeepCopy(dst interface{}, src interface{}) error {
	if dst == nil || src == nil {
		return errors.New(errorNilPointer)
	}

	if gobCodecPool == nil {
		gobCodecPool = &pool.Pool{
			Dial: func() (io.Closer, error) {
				c := newGobCodec()
				return c, nil
			},
			MaxIdle:         100,
			MaxActive:       0,
			IdleTimeout:     10 * time.Second,
			MaxConnLifetime: 0,
			Wait:            true,
		}
	}

	gc := gobCodecPool.Get()
	defer gobCodecPool.Put(gc)
	c := gc.(*gobCodec)

	if err := c.enc.Encode(src); err != nil {
		return errors.Wrap(err, errorEncoding)
	}

	if err := c.dec.Decode(dst); err != nil {
		return errors.Wrap(err, errorDecoding)
	}

	return nil
}

// MsgpackDeepCopy vmihailenco/msgpack deep copy
func MsgpackDeepCopy(dst interface{}, src interface{}) error {
	if dst == nil || src == nil {
		return errors.New(errorNilPointer)
	}

	encodedBytes, err := msgpack.Marshal(src)
	if err != nil {
		return errors.Wrap(err, errorEncoding)
	}
	if err := msgpack.Unmarshal(encodedBytes, dst); err != nil {
		return errors.Wrap(err, errorDecoding)
	}

	return nil
}

// json-iterator/go deep copy
var jsoniterJSON = jsoniter.ConfigFastest

func JsoniterDeepCopy(dst interface{}, src interface{}) error {
	if dst == nil || src == nil {
		return errors.New(errorNilPointer)
	}

	encodedBytes, err := jsoniterJSON.Marshal(src)
	if err != nil {
		return errors.Wrap(err, errorEncoding)
	}
	if err := jsoniterJSON.Unmarshal(encodedBytes, dst); err != nil {
		return errors.Wrap(err, errorDecoding)
	}

	return nil
}

// pquerna/ffjson deep copy
func ffjsonDeepCopy(dst interface{}, src interface{}) error {
	if dst == nil || src == nil {
		return errors.New(errorNilPointer)
	}

	encodedBytes, err := ffjson.Marshal(src)
	if err != nil {
		return errors.Wrap(err, errorEncoding)
	}
	if err := ffjson.Unmarshal(encodedBytes, dst); err != nil {
		return errors.Wrap(err, errorDecoding)
	}

	return nil
}

// shamaton/msgpack deep copy
func shamatonMsgpackDeepCopy(dst interface{}, src interface{}) error {
	if dst == nil || src == nil {
		return errors.New(errorNilPointer)
	}

	encodedBytes, err := sMsgpack.Encode(src)
	if err != nil {
		return errors.Wrap(err, errorEncoding)
	}
	if err := sMsgpack.Decode(encodedBytes, dst); err != nil {
		return errors.Wrap(err, errorDecoding)
	}

	return nil
}
