package appmgr

import (
	"path"

	"github.com/spf13/viper"

	"github.com/frame-go/framego/client/cache"
	"github.com/frame-go/framego/client/database"
	"github.com/frame-go/framego/client/pulsar"
	"github.com/frame-go/framego/config"
	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/utils"
)

type EndpointsConfig struct {
	Grpc string `json:"grpc" mapstructure:"grpc"`
	Http string `json:"http" mapstructure:"http"`
}

type GrpcSecurityConfig struct {
	Cert string `json:"cert" mapstructure:"cert"`
	Key  string `json:"key" mapstructure:"key"`
	Ca   string `json:"ca" mapstructure:"ca"`
}

type ServiceSecurityConfig struct {
	Grpc GrpcSecurityConfig `json:"grpc" mapstructure:"grpc"`
}

type ObservableConfig struct {
	Endpoints EndpointsConfig `json:"endpoints" mapstructure:"endpoints" validate:"required"`
	Modules   []string        `json:"modules" mapstructure:"modules"`
}

type ServiceConfig struct {
	Name        string                `json:"name" mapstructure:"name" validate:"required"`
	Endpoints   EndpointsConfig       `json:"endpoints" mapstructure:"endpoints" validate:"required"`
	Security    ServiceSecurityConfig `json:"security" mapstructure:"security"`
	Middlewares []interface{}         `json:"middlewares" mapstructure:"middlewares"`
}

type GrpcServerConfig struct {
	Name     string             `json:"name" mapstructure:"name" validate:"required"`
	Endpoint string             `json:"endpoint" mapstructure:"endpoint" validate:"required"`
	Security GrpcSecurityConfig `json:"security" mapstructure:"security"`
}

type GrpcConfig struct {
	Middlewares []interface{}      `json:"middlewares" mapstructure:"middlewares"`
	Servers     []GrpcServerConfig `json:"servers" mapstructure:"servers"`
}

type ClientsConfig struct {
	Grpc GrpcConfig `json:"grpc" mapstructure:"grpc"`
}

type IDGeneratorConfig struct {
	ServiceID int    `json:"service_id" mapstructure:"service_id"`
	Key       string `json:"key" mapstructure:"key"`
}

type AppConfig struct {
	Name        string            `json:"name" mapstructure:"name"`
	Observable  ObservableConfig  `json:"observable" mapstructure:"observable"`
	Services    []ServiceConfig   `json:"services" mapstructure:"services"`
	Jobs        []string          `json:"jobs" mapstructure:"jobs"`
	Clients     ClientsConfig     `json:"clients" mapstructure:"clients"`
	Databases   []database.Config `json:"databases" mapstructure:"databases"`
	Caches      []cache.Config    `json:"caches" mapstructure:"caches"`
	Pulsars     []pulsar.Config   `json:"pulsar" mapstructure:"pulsars"`
	IDGenerator IDGeneratorConfig `json:"id_generator" mapstructure:"id_generator"`
}

func (c *GrpcSecurityConfig) Resolve() {
	c.Cert = resolvePathInConfig(c.Cert)
	c.Key = resolvePathInConfig(c.Key)
	c.Ca = resolvePathInConfig(c.Ca)
}

func parseMiddlewareConfig(middlewareConfig interface{}) (name string, options map[string]interface{}, err error) {
	ok := false
	name, ok = middlewareConfig.(string)
	if ok {
		return
	}
	options, ok = middlewareConfig.(map[string]interface{})
	if !ok {
		err = errors.New("middleware_config_error_unknown_type").With("config", middlewareConfig).
			With("type", utils.GetTypeName(middlewareConfig))
		return
	}
	name, err = config.StringMap(options).GetString("name")
	if err != nil {
		err = errors.Wrap(err, "middleware_config_error_missed_name").With("config", middlewareConfig)
		return
	}
	return
}

func resolvePathInConfig(filePath string) string {
	if filePath == "" || path.IsAbs(filePath) {
		return filePath
	}
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		return filePath
	}
	configDir := path.Dir(configPath)
	return path.Join(configDir, filePath)
}
