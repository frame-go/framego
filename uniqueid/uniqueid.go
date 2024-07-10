package uniqueid

import (
	"fmt"
	"strconv"
)

type ID uint64

func ParseID(id string) (ID, error) {
	v, err := strconv.ParseUint(id, 16, 64)
	return ID(v), err
}

func ParseIDSafe(id string) ID {
	v, _ := strconv.ParseUint(id, 16, 64)
	return ID(v)
}

func (h ID) String() string {
	return fmt.Sprintf("%016x", uint64(h))
}

func (h ID) Uint64() uint64 {
	return uint64(h)
}
