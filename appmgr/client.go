package appmgr

import (
	"context"
	"strings"

	"google.golang.org/grpc"
)

type clientManagerImpl struct {
	ClientManager

	middlewares *middlewareApplier
	grpcClients map[string][]grpc.ClientConnInterface
}

func newClientManager(ctx context.Context, mm *middlewareManager, config *ClientsConfig) (ClientManager, error) {
	c := &clientManagerImpl{
		middlewares: mm.Apply(nil, config.Grpc.Middlewares),
		grpcClients: make(map[string][]grpc.ClientConnInterface),
	}
	for _, server := range config.Grpc.Servers {
		securityConfig := &server.Security
		securityConfig.Resolve()
		tlsConfig, err := newTLSConfig(securityConfig.Cert, securityConfig.Key, securityConfig.Ca)
		if err != nil {
			return nil, err
		}
		var clients []grpc.ClientConnInterface
		endpoints := strings.Split(server.Endpoint, ",")
		for _, endpoint := range endpoints {
			client := newGrpcClient(ctx, server.Name, endpoint, tlsConfig, c.middlewares)
			clients = append(clients, client)
		}
		c.grpcClients[server.Name] = clients
	}
	return c, nil
}

func (c *clientManagerImpl) GetGrpcClientConn(name string) grpc.ClientConnInterface {
	clients, ok := c.grpcClients[name]
	if !ok || len(clients) <= 0 {
		return nil
	}
	return clients[0]
}

func (c *clientManagerImpl) GetGrpcClientConns(name string) []grpc.ClientConnInterface {
	clients, ok := c.grpcClients[name]
	if !ok {
		return nil
	}
	return clients
}
