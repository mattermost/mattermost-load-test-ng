package simplecontroller

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
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
	v.AddConfigPath("./../config/")
	v.AddConfigPath("./../../../config/")
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
	if _, err := parseActions(&SimpleController{}, c.Actions); err != nil {
		return fmt.Errorf("actions are not valid: %w", err)
	}
	return nil
}

func parseActions(c *SimpleController, definitions []actionDefinition) ([]*UserAction, error) {
	actions := make([]*UserAction, 0)
	for _, def := range definitions {
		s := strings.Split(def.ActionId, ".")
		if len(s) != 2 {
			return nil, fmt.Errorf("invalid action ID: %q", def.ActionId)
		}
		var run control.UserAction
		var ok bool
		switch s[0] {
		case "simplecontroller":
			run = actionByName(c, s[1])
			if run == nil {
				return nil, fmt.Errorf("could not find function %q", s[1])
			}
		case "control":
			run, ok = control.Actions[s[1]]
			if !ok {
				return nil, fmt.Errorf("could not find function %q", s[1])
			}
		default:
			return nil, fmt.Errorf("invalid action package: %q", s[0])
		}
		actions = append(actions, &UserAction{
			run:          run,
			waitAfter:    time.Duration(def.WaitAfterMs),
			runFrequency: def.RunFrequency,
		})
	}
	return actions, nil
}
