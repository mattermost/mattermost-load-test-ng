package simplecontroller

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the the rate and user actions definitions that will be runned by
// the SimpleController
type Config struct {
	Rate    float64
	Actions []actionDefinition
}

type actionDefinition struct {
	ActionId     string
	RunFrequency int
	WaitAfterMs  int
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will search a config file in predefined folders.
func ReadConfig(configFilePath string) (*Config, error) {
	v := viper.New()

	v.SetConfigName("simplecontroller")
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.AddConfigPath("./../config/")
	v.AddConfigPath("./../../../config/")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configFilePath != "" {
		v.SetConfigFile(configFilePath)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("unable to read configuration file: %w", err)
	}
	var cfg *Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
