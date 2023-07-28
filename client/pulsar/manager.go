package pulsar

import (
	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
	"github.com/rs/zerolog"
)

type ClientManager interface {
	GetClient(string) pulsarclient.Client
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
	clients map[string]pulsarclient.Client
}

func (c *clientManagerImpl) GetClient(name string) pulsarclient.Client {
	return c.clients[name]
}

func NewClientManager(configs []Config, opts ...Option) (ClientManager, error) {
	c := &clientManagerImpl{
		configs: configs,
		opts:    opts,
		clients: make(map[string]pulsarclient.Client),
	}
	for _, config := range c.configs {
		client, err := NewClient(&config, c.opts...)
		if err != nil {
			return nil, err
		}
		c.clients[config.Name] = client
	}
	return c, nil
}
