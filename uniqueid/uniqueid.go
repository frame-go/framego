package uniqueid

import (
	"fmt"
	"strconv"
)

// ID is a 64-bit unique identifier, stored as uint64 and rendered as a 16-char hex string.
type ID uint64

// ParseID parses a hex id string into an ID; empty or malformed input returns an error.
func ParseID(id string) (ID, error) {
	v, err := strconv.ParseUint(id, 16, 64)
	return ID(v), err
}

// ParseIDOptional parses an optional id: "" yields the zero ID, a non-empty malformed string returns an error.
func ParseIDOptional(id string) (ID, error) {
	if id == "" {
		return 0, nil
	}
	return ParseID(id)
}

// ParseIDSafe parses a hex id string, coercing empty or any malformed input to the zero ID (no error).
func ParseIDSafe(id string) ID {
	v, _ := strconv.ParseUint(id, 16, 64)
	return ID(v)
}

// String renders the id as a fixed 16-char hex string.
func (h ID) String() string {
	return fmt.Sprintf("%016x", uint64(h))
}

// StringOrEmpty is like String but returns "" for the zero ID, which represents an absent/optional reference.
func (h ID) StringOrEmpty() string {
	if h == 0 {
		return ""
	}
	return h.String()
}

// Uint64 returns the id as a plain uint64.
func (h ID) Uint64() uint64 {
	return uint64(h)
}
