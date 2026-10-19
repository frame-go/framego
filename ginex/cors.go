package ginex

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/frame-go/framego/config"
	"github.com/frame-go/framego/log"
)

type corsConfig struct {
	AllowAllOrigins bool `json:"allow_all_origins"`

	// AllowOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// Default value is []
	AllowOrigins []string `json:"allow_origins"`

	// AllowMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is [], which sends no Access-Control-Allow-Methods
	// header, so browsers allow only simple methods (GET, HEAD and POST)
	AllowMethods []string `json:"allow_methods"`

	// AllowHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	AllowHeaders []string `json:"allow_headers"`

	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool `json:"allow_credentials"`

	// ExposeHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposeHeaders []string `json:"expose_headers"`

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge uint32 `json:"max_age"`

	// Allows to add origins like http://some-domain/*, https://api.* or http://some.*.subdomain.com
	AllowWildcard bool `json:"allow_wildcard"`

	// Allows usage of popular browser extensions schemas
	AllowBrowserExtensions bool `json:"allow_browser_extensions"`

	// Allows to add custom schema like tauri://
	CustomSchemas []string `json:"custom_schemas"`

	// Allows usage of WebSocket protocol
	AllowWebSockets bool `json:"allow_web_sockets"`
}

func CorsMiddleware(options map[string]interface{}) gin.HandlerFunc {
	configMap := config.StringMap(options)
	c := &corsConfig{}
	err := configMap.ToStruct(c)
	if err != nil {
		log.Logger.Fatal().Err(err).Interface("config", options).Msg("parse_cors_config_error")
	}
	return cors.New(cors.Config{
		AllowAllOrigins:        c.AllowAllOrigins,
		AllowOrigins:           c.AllowOrigins,
		AllowOriginFunc:        nil,
		AllowMethods:           c.AllowMethods,
		AllowHeaders:           c.AllowHeaders,
		AllowCredentials:       c.AllowCredentials,
		ExposeHeaders:          c.ExposeHeaders,
		MaxAge:                 time.Duration(c.MaxAge) * time.Second,
		AllowWildcard:          c.AllowWildcard,
		AllowBrowserExtensions: c.AllowBrowserExtensions,
		CustomSchemas:          c.CustomSchemas,
		AllowWebSockets:        c.AllowWebSockets,
		AllowFiles:             false,
	})
}
