# Framego UniqueID Module

## Introduction

An ID is a 64-bit unique identifier that is non-sequential and unpredictable. It is held as a `uint64` and rendered as a fixed 16-character hex string for transport or display.

## Package

### ID

```go
// ID is a 64-bit unique identifier.
type ID uint64

// String renders the id as a fixed 16-char hex string (zero -> "0000000000000000").
func (h ID) String() string

// StringOrEmpty is like String but returns "" for the zero id (an absent/optional reference).
func (h ID) StringOrEmpty() string

// Uint64 returns the id as a plain uint64.
func (h ID) Uint64() uint64
```

### Parsing

```go
// ParseID parses a hex id string; empty or malformed input is an error.
func ParseID(id string) (ID, error)

// ParseIDOptional parses an optional id: "" -> zero id (no error), valid -> its id, non-empty malformed -> error.
func ParseIDOptional(id string) (ID, error)

// ParseIDSafe parses a hex id string, coercing empty or any malformed input to the zero id (no error).
func ParseIDSafe(id string) ID
```

### Generator

```go
// Generator mints new unique IDs.
type Generator interface {
	NewID() ID
}

// NewGenerator creates a generator from a secret key and an explicit node id (0..4095).
func NewGenerator(key []byte, nodeID int) (Generator, error)

// NewGeneratorFromHostName derives the node id from serviceID plus the instance index in the
// hostname's trailing "-<n>" suffix; pass debug=true to skip the hostname and use index 0.
func NewGeneratorFromHostName(key []byte, serviceID int, debug bool) (Generator, error)
```

## Generating IDs

Create one generator per process and mint IDs with `NewID`:

```go
gen, err := uniqueid.NewGenerator(key, nodeID)
if err != nil {
	// handle error
}
id := gen.NewID() // uniqueid.ID
```

In multi-instance deployments, `NewGeneratorFromHostName` derives a distinct node id per instance from the hostname, so every instance mints non-colliding IDs.

## Conversion

An id is a `uint64` in memory and a fixed 16-char hex `string` at the boundary; convert between the two through the `ID` type.

**`uint64` -> `string`** -- wrap the value in `ID`, then call `String` (or `StringOrEmpty` for an optional id):

```go
s := uniqueid.ID(v).String()        // v=1 -> "0000000000000001"
s = uniqueid.ID(v).StringOrEmpty()  // optional id: v=0 -> ""
```

**`string` -> `uint64`** -- parse the string, then call `Uint64`:

```go
id, err := uniqueid.ParseID(s)      // required id: empty or invalid is an error
if err != nil {
	// handle invalid id
}
v := id.Uint64()                    // uint64
```

`0` is never a valid generated id, so it represents an **absent / optional** reference; rendered to a string it should be `""`, not `"0000000000000000"`. Pick the variant by how the zero / empty case must behave:

| `uint64` -> `string` | Zero renders as |
|---|---|
| `uniqueid.ID(v).String()` (required / always-set) | `"0000000000000000"` |
| `uniqueid.ID(v).StringOrEmpty()` (optional / nullable) | `""` |

| `string` -> `uint64` | Empty `""` | Invalid non-empty |
|---|---|---|
| `ParseID(s)` (required) | error | error |
| `ParseIDOptional(s)` (optional, empty = absent) | zero id | error |
| `ParseIDSafe(s)` (trusted internal, already validated) | zero id | zero id (silent) |
