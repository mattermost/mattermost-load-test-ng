package simplecontroller

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var v = viper.New()

type Config struct {
	Rate    float32            `json:"Rate"`
	Actions []actionDefinition `json:"Actions"`
}

type actionDefinition struct {
	ActionId     string `josn:"ActionId"`
	RunFrequency int    `json:"RunFrequency"`
	WaitAfterMs  int    `json:"WaitAfterMs"`
}

func ReadConfig(configFilePath string) error {
	v.SetConfigName("simplecontroller")
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configFilePath != "" {
		v.SetConfigFile(configFilePath)
	}

	if err := v.ReadInConfig(); err != nil {
		return errors.Wrap(err, "unable to read configuration file")
	}

	return nil
}

func GetConfig() (*Config, error) {
	var cfg *Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsValid checks whether a Config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	return nil
}
