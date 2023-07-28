package cache

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

type ClientManager interface {
	GetClient(name string) Client
}

type options struct {
	logger *zerolog.Logger
}

type Option func(*options)

func WithLogger(logger *zerolog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

type clientManagerImpl struct {
	configs []Config
	opts    []Option
	clients map[string]Client
}

func NewClientManager(configs []Config, opts ...Option) (ClientManager, error) {
	c := &clientManagerImpl{
		configs: configs,
		opts:    opts,
		clients: make(map[string]Client),
	}
	initRedis(opts...)
	for _, config := range c.configs {
		cacheType := strings.ToLower(config.Type)
		newClientFunc := NewRedisClient
		switch cacheType {
		case "", "redis":
			// Redis
		default:
			return nil, fmt.Errorf("unknown_cache_client_type:%s", cacheType)
		}
		client, err := newClientFunc(&config, c.opts...)
		if err != nil {
			return nil, err
		}
		c.clients[config.Name] = client
	}
	return c, nil
}

func (c *clientManagerImpl) GetClient(name string) Client {
	return c.clients[name]
}
