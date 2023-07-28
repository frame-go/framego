package json

import (
	"encoding/json"

	"github.com/frame-go/framego/utils"
)

var Marshal = json.Marshal
var Unmarshal = json.Unmarshal

func MarshalString(v interface{}) (string, error) {
	bs, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return utils.BytesToString(bs), err
}

func UnmarshalString(data string, v interface{}) error {
	return json.Unmarshal(utils.StringToBytes(data), v)
}
