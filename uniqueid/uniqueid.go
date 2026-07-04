package uniqueid

import (
	"database/sql/driver"
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

// Scan implements sql.Scanner. Ids are stored in signed bigint columns via
// two's-complement bit-cast, so negative values map back to the upper uint64 range.
func (h *ID) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*h = 0
	case int64:
		*h = ID(v)
	case uint64:
		*h = ID(v)
	case []byte:
		parsed, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("uniqueid: scan %q: %w", v, err)
		}
		*h = ID(parsed)
	default:
		return fmt.Errorf("uniqueid: unsupported scan type %T", src)
	}
	return nil
}

// Value implements driver.Valuer, returning the raw uint64 so the driver layer picks
// the representation: postgres bit-casts to bigint via the framego codec, mysql binds
// unsigned natively. Returning int64 here would break BIGINT UNSIGNED columns on mysql.
func (h ID) Value() (driver.Value, error) {
	return uint64(h), nil
}
