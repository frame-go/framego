package ginex

import (
	"regexp"
	"slices"

	"github.com/aurowora/compress"
	"github.com/gin-gonic/gin"

	"github.com/frame-go/framego/config"
	"github.com/frame-go/framego/grpcex"
	"github.com/frame-go/framego/log"
)

type compressConfig struct {
	MinLength         int    `json:"min_length"`
	ExcludedPathRegex string `json:"excluded_path_regex"`
}

func CompressMiddleware(options map[string]interface{}) gin.HandlerFunc {
	configMap := config.StringMap(options)
	c := &compressConfig{}
	err := configMap.ToStruct(c)
	if err != nil {
		log.Logger.Fatal().Err(err).Interface("config", options).Msg("parse_compress_config_error")
	}
	opts := []compress.CompressOption{}
	if c.MinLength >= 0 {
		opts = append(opts, compress.WithMinCompressBytes(c.MinLength))
	}
	var excludedRegex *regexp.Regexp
	if c.ExcludedPathRegex != "" {
		excludedRegex, err = regexp.Compile(c.ExcludedPathRegex)
		if err != nil {
			log.Logger.Fatal().Err(err).Interface("regex", c.ExcludedPathRegex).
				Msg("parse_compress_config_excluded_regex_error")
		}
	}
	opts = append(opts, compress.WithExcludeFunc(func(c *gin.Context) bool {
		// compress itself only checks the first Accept header for text/event-stream.
		if slices.Contains(c.Request.Header.Values("Accept"), grpcex.MIMEEventStream) {
			return true
		}
		return excludedRegex != nil && excludedRegex.MatchString(c.Request.URL.Path)
	}))
	return compress.Compress(opts...)
}
