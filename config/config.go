package config

import (
	common "github.com/edwinavalos/common/config"
	"github.com/edwinavalos/common/logger"
	"github.com/spf13/viper"
)

func NewConfig() *common.Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./resources")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Error("unable to read configuration file, exiting. %s", err)
		panic(err)
	}
	logger.Info("Config file used: %s", viper.ConfigFileUsed())
	conf := common.NewConfig()
	conf.ReadConfig()
	return conf
}
