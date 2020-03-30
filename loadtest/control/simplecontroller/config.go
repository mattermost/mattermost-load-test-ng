package simplecontroller

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the the rate and user actions definitions that will be run by
// the SimpleController.
type Config struct {
	// Rate is the idle time coefficient for user actions that will be performed
	// sequentially.
	Rate float64
	// Actions are the user action definitions that will be run by the controller.
	Actions []actionDefinition
}

type actionDefinition struct {
	// ActionId is the key of an action which is mapped to a user action
	// implementation.
	ActionId string
	// RunPeriod determines how often the action will be performed.
	RunPeriod int
	// WaitAfterMs is the wait time after the action is performed.
	WaitAfterMs int
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
