package config

import (
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"gitlab.com/olaris/olaris-server/helpers"
)

var ConfigDir string

func GetDefaultConfigDir() string {
	defaultConfigDir := path.Join(helpers.GetHome(), ".config", "olaris")
	if configDirEnv := os.Getenv("OLARIS_CONFIG_DIR"); configDirEnv != "" {
		defaultConfigDir = configDirEnv
	}

	return defaultConfigDir
}

func InitViper() {
	viper.SetConfigName("olaris")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("olaris")
	viper.AutomaticEnv()

	viper.AddConfigPath(ConfigDir)
	viper.Set("configdir", ConfigDir)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// the user has no config file
		} else {
			logrus.WithError(err).WithField("configFile", viper.ConfigFileUsed()).Warnln("An error occurred while reading config file, contents are being ignored.")
		}
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		logrus.Debugf("error applying configuration: %s\n", err.Error())
	}
}
