package config

import (
	"fmt"
	"path/filepath"

	"github.com/go-playground/validator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/frame-go/framego/errors"
)

// BindArgs binds command line arguments to config
func BindArgs(cmd *cobra.Command) {
	viper.AutomaticEnv()

	cmd.Flags().StringP("config-path", "c", "", "config file path")
	_ = viper.BindPFlag("config_path", cmd.Flags().Lookup("config-path"))

	cmd.Flags().BoolP("debug", "d", false, "enable debug mode")
	_ = viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))

	cmd.Flags().StringP("log-level", "l", "", "log level: trace, debug, info, warn, error")
	_ = viper.BindPFlag("log_level", cmd.Flags().Lookup("log-level"))

	cmd.Flags().BoolP("beautify-log", "b", false, "enable human-friendly, colorized log")
	_ = viper.BindPFlag("beautify_log", cmd.Flags().Lookup("beautify-log"))

	cmd.Flags().StringP("job", "j", "", "run job")
	_ = viper.BindPFlag("job", cmd.Flags().Lookup("job"))

	bindApolloArgs(cmd)
}

// InitConfig loads config from config path
func InitConfig() error {
	err := initApolloConfig()
	if err != nil {
		return errors.Wrap(err, "ReadConfigError(SOURCE=Apollo)")
	}
	configPath := viper.GetString("config_path")
	if configPath != "" {
		viper.SetConfigFile(configPath)
		viper.SetConfigType(getConfigType(configPath))
		err = viper.ReadInConfig()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("ReadConfigError(CONFIG_PATH=\"%s\")", configPath))
		}
	}
	return nil
}

// GetStringMap gets StringMap object by path in config
func GetStringMap(path string) StringMap {
	return viper.GetStringMap(path)
}

// GetStruct gets Struct object by path in config
func GetStruct(path string, value interface{}) error {
	return viper.UnmarshalKey(path, value)
}

// GetStructWithValidation gets Struct object by path in config with validation
func GetStructWithValidation(path string, value interface{}) error {
	err := viper.UnmarshalKey(path, value)
	if err != nil {
		return err
	}
	validate := validator.New()
	err = validate.Struct(value)
	if err != nil {
		return errors.Wrap(err, "config_object_validation_error")
	}
	return nil
}

func getConfigType(name string) string {
	ext := filepath.Ext(name)
	if len(ext) > 1 {
		return ext[1:]
	}
	return ""
}
