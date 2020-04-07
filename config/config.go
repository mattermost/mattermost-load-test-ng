// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package config

import (
	"errors"
	"fmt"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/viper"
)

func ReadConfigFile(v *viper.Viper, configName string) error {
	if v == nil {
		return errors.New("config: v should not be nil")
	}
	if configName == "" {
		return errors.New("config: configName should not be empty")
	}

	if err := v.ReadInConfig(); err != nil {
		// If we can't find the config let's rely on the default one.
		// var configErr *viper.ConfigFileNotFoundError
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			mlog.Warn("config: falling back to default configuration file")
			v.SetConfigName(fmt.Sprintf("%s.default", configName))
			if err := v.ReadInConfig(); err != nil {
				return fmt.Errorf("config: unable to read configuration file: %w", err)
			}
		} else {
			return fmt.Errorf("config: unable to read configuration file: %w", err)
		}
	}

	return nil
}
