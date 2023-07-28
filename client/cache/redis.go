package cache

import (
	"context"
	"runtime"
	"time"

	redis "github.com/redis/go-redis/v9"

	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/utils"
)

const (
	poolSizePerProc = 10
	connMaxIdleTime = 60 * time.Second
	connMaxLifeTime = 10 * time.Minute
)

func initRedis(opts ...Option) {
	var clientOpts options
	for _, opt := range opts {
		opt(&clientOpts)
	}
	if clientOpts.logger != nil {
		redis.SetLogger(NewLoggerWithZerolog(clientOpts.logger))
	}
}

type redisClient struct {
	client *redis.Client
}

func NewRedisClient(config *Config, opts ...Option) (Client, error) {
	option := &redis.Options{
		Addr:                  config.Address,
		Username:              config.Username,
		Password:              config.Password,
		DB:                    int(config.DB),
		MaxRetries:            1,
		ContextTimeoutEnabled: false,
		PoolFIFO:              false,
		PoolSize:              poolSizePerProc * runtime.GOMAXPROCS(0),
		ConnMaxIdleTime:       connMaxIdleTime,
		ConnMaxLifetime:       connMaxLifeTime,
	}
	c := redis.NewClient(option)
	client := &redisClient{
		client: c,
	}
	return client, nil
}

func (c *redisClient) GetRawClient() any {
	return c.client
}

func (c *redisClient) Add(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	b, err := Serialize(value)
	if err != nil {
		return false, errors.Wrap(err, "redis_add_serialize_value_error").With("key", key).With("value", value)
	}
	result, err := c.client.SetNX(ctx, key, b, expiration).Result()
	if err != nil {
		return false, errors.Wrap(err, "redis_add_request_error").With("key", key)
	}
	return result, nil
}

func (c *redisClient) Delete(ctx context.Context, keys ...string) (int, error) {
	result, err := c.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, errors.Wrap(err, "redis_delete_request_error")
	}
	return int(result), nil
}

func (c *redisClient) Exists(ctx context.Context, keys ...string) (int, error) {
	result, err := c.client.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, errors.Wrap(err, "redis_exists_request_error").With("keys", keys)
	}
	return int(result), nil
}

func (c *redisClient) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	result, err := c.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return false, errors.Wrap(err, "redis_expire_request_error").With("key", key)
	}
	return result, nil
}

func (c *redisClient) Get(ctx context.Context, key string, value any) error {
	v, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return errors.Wrap(err, "redis_get_request_error").With("key", key)
	}
	err = Deserialize(v, value)
	if err != nil {
		return errors.Wrap(err, "redis_get_deserialize_value_error").With("key", key).With("value", value)
	}
	return nil
}

func (c *redisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	result, err := c.client.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, errors.Wrap(err, "redis_incrby_request_error").With("key", key)
	}
	return result, nil
}

func (c *redisClient) MGet(ctx context.Context, kv map[string]any) error {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	result, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return errors.Wrap(err, "redis_mget_request_error")
	}
	for i, k := range keys {
		keyResult := result[i]
		if keyResult == nil {
			kv[k] = nil
		} else {
			sv, ok := keyResult.(string)
			if !ok {
				return errors.Wrap(err, "redis_mget_convert_value_error").With("key", k).With("value", keyResult)
			}
			err = Deserialize(utils.StringToBytes(sv), kv[k])
			if err != nil {
				return errors.Wrap(err, "redis_mget_deserialize_value_error").With("key", k).With("value", sv)
			}
		}
	}
	return nil
}

func (c *redisClient) MSet(ctx context.Context, kv map[string]any) error {
	skv := make(map[string]any)
	for k, v := range kv {
		sv, err := Serialize(v)
		if err != nil {
			return errors.Wrap(err, "redis_mset_serialize_value_error").With("key", k).With("value", v)
		}
		skv[k] = sv
	}
	err := c.client.MSet(ctx, skv).Err()
	if err != nil {
		return errors.Wrap(err, "redis_mset_request_error")
	}
	return nil
}

func (c *redisClient) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	b, err := Serialize(value)
	if err != nil {
		return errors.Wrap(err, "redis_set_serialize_value_error").With("key", key).With("value", value)
	}
	err = c.client.Set(ctx, key, b, expiration).Err()
	if err != nil {
		return errors.Wrap(err, "redis_set_request_error").With("key", key)
	}
	return nil
}

func (c *redisClient) Update(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	b, err := Serialize(value)
	if err != nil {
		return false, errors.Wrap(err, "redis_update_serialize_value_error").With("key", key).With("value", value)
	}
	result, err := c.client.SetXX(ctx, key, b, expiration).Result()
	if err != nil {
		return false, errors.Wrap(err, "redis_update_request_error").With("key", key)
	}
	return result, nil
}
