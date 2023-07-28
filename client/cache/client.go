package cache

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// KeepExpiration keeps current expiration for existing value
const KeepExpiration = -1

// Nil error returned when key does not exist.
const Nil = redis.Nil

// Client is a Cache client representing a connection pool.
// It is safe for concurrent use by multiple goroutines.
type Client interface {
	// GetRawClient gets underlying client object
	GetRawClient() any

	// Add adds key if key exists. When key does not exist, no operation is performed.
	// Zero expiration means the key has no expiration time; KeepExpiration keeps existing expiration.
	// Returns whether the key is added.
	Add(ctx context.Context, key string, value any, expiration time.Duration) (bool, error)

	// Delete removes the keys. When key does not exist, no operation is performed.
	// Returns the number of keys that were removed.
	Delete(ctx context.Context, keys ...string) (int, error)

	// Exists checks if keys exist.
	// Returns the number of keys that exist.
	Exists(ctx context.Context, keys ...string) (int, error)

	// Expire updates the expiration of key. When key does not exist, no operation is performed.
	// Returns whether the expiration is updated.
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)

	// Get gets value of key.
	// `value` should be initialized to a pointer to output data type.
	// If the key does not exist, returns `Nil` error
	Get(ctx context.Context, key string, value any) error

	// IncrBy increments the number stored at key by `value`.
	// If the key does not exist, it is set to 0 before performing the operation.
	// `value` can be either positive or negative integer.
	// Returns the value after increment.
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	// MGet gets values of multiple keys.
	// Map value should be initialized to pointer to output data type.
	// If the key does not exist, the value will be set to nil.
	MGet(ctx context.Context, kv map[string]any) error

	// MSet sets values of multiple keys.
	MSet(ctx context.Context, kv map[string]any) error

	// Set adds or updates key.
	// Zero expiration means the key has no expiration time; KeepExpiration keeps existing expiration.
	Set(ctx context.Context, key string, value any, expiration time.Duration) error

	// Update updates key if key does not exist. When key already holds a value, no operation is performed.
	// Zero expiration means the key has no expiration time; KeepExpiration keeps existing expiration.
	// Returns whether the key is updated.
	Update(ctx context.Context, key string, value any, expiration time.Duration) (bool, error)
}
