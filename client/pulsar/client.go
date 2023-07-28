package pulsar

import (
	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
)

func NewClient(config *Config, opts ...Option) (pulsarclient.Client, error) {
	var clientOpts options
	for _, opt := range opts {
		opt(&clientOpts)
	}
	pulsarOpts := pulsarclient.ClientOptions{
		URL: config.URL,
	}
	if config.Token != "" {
		pulsarOpts.Authentication = pulsarclient.NewAuthenticationToken(config.Token)
	}
	if clientOpts.logger != nil {
		pulsarOpts.Logger = NewLoggerWithZerolog(clientOpts.logger)
	}
	return pulsarclient.NewClient(pulsarOpts)
}
