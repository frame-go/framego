package config

import (
	"fmt"
	"time"

	"github.com/shima-park/agollo"
	remote "github.com/shima-park/agollo/viper-remote"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/frame-go/framego/log"
)

const apolloDefaultNamespace = "config.yaml"
const apolloLongPollerInterval = 10 * time.Second
const apolloHeartBeatInterval = 5 * time.Minute

func bindApolloArgs(cmd *cobra.Command) {
	cmd.Flags().String("apollo-server", "", "apollo server endpoint")
	_ = viper.BindPFlag("apollo_server", cmd.Flags().Lookup("apollo-server"))

	cmd.Flags().String("apollo-app-id", "", "apollo app id")
	_ = viper.BindPFlag("apollo_app_id", cmd.Flags().Lookup("apollo-app-id"))

	cmd.Flags().String("apollo-access-key", "", "apollo app access key secret")
	_ = viper.BindPFlag("apollo_access_key", cmd.Flags().Lookup("apollo-access-key"))

	cmd.Flags().String("apollo-environment", "", "apollo environment")
	_ = viper.BindPFlag("apollo_environment", cmd.Flags().Lookup("apollo-environment"))

	cmd.Flags().String("apollo-cluster", "", "apollo cluster")
	_ = viper.BindPFlag("apollo_cluster", cmd.Flags().Lookup("apollo-cluster"))

	cmd.Flags().String("apollo-namespace", "", "apollo namespace")
	_ = viper.BindPFlag("apollo_namespace", cmd.Flags().Lookup("apollo-namespace"))
}

func setFallbackValue(key string, fallbackKey string) {
	if viper.GetString(key) == "" {
		viper.Set(key, viper.GetString(fallbackKey))
	}
}

func fixApolloArgs() {
	setFallbackValue("apollo_server", "_apollo_server_")
	setFallbackValue("apollo_app_id", "_apollo_app_id_")
	setFallbackValue("apollo_access_key", "_apollo_access_key_")
	setFallbackValue("apollo_environment", "_apollo_environment_")
	setFallbackValue("apollo_cluster", "_apollo_cluster_")
	setFallbackValue("apollo_namespace", "_apollo_namespace_")
}

func initApolloConfig() error {
	fixApolloArgs()
	server := viper.GetString("apollo_server")
	appId := viper.GetString("apollo_app_id")
	if server == "" || appId == "" {
		return nil
	}
	namespace := viper.GetString("apollo_namespace")
	if namespace == "" {
		namespace = apolloDefaultNamespace
	}
	accessKey := viper.GetString("apollo_access_key")
	cluster := viper.GetString("apollo_cluster")
	if cluster == "" {
		cluster = "default"
	}
	logger := &AgolloZerologAdapter{}
	remote.SetAppID(appId)
	remote.SetAgolloOptions(
		agollo.AccessKey(accessKey),
		agollo.Cluster(cluster),
		agollo.AutoFetchOnCacheMiss(),
		agollo.LongPollerInterval(apolloLongPollerInterval),
		agollo.FailTolerantOnBackupExists(),
		agollo.HeartBeatInterval(apolloHeartBeatInterval),
		agollo.WithLogger(logger),
	)
	viper.SetConfigType(getConfigType(namespace))
	err := viper.AddRemoteProvider("apollo", server, namespace)
	if err != nil {
		return err
	}
	err = viper.ReadRemoteConfig()
	if err != nil {
		return err
	}
	err = viper.GetViper().WatchRemoteConfigOnChannel()
	if err != nil {
		return err
	}
	return nil
}

type AgolloZerologAdapter struct {
}

func (l *AgolloZerologAdapter) Log(kvs ...interface{}) {
	if log.Logger == nil {
		fmt.Println(kvs...)
		return
	}
	logger := log.Logger.Info()
	groups := len(kvs) / 2
	for i := 0; i < groups; i++ {
		index := i * 2
		key, ok := kvs[index].(string)
		if ok {
			value := kvs[index+1]
			logger = logger.Interface(key, value)
		}
	}
	logger.Msg("apollo_client")
}
