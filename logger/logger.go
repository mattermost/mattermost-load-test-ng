// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package logger

import (
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/server/config"
	"github.com/mattermost/mattermost-server/v6/server/platform/shared/mlog"
)

// Settings holds information used to initialize a new logger.
type Settings struct {
	EnableConsole bool   `default:"true"`
	ConsoleJson   bool   `default:"false"`
	ConsoleLevel  string `default:"ERROR" validate:"oneof:{TRACE, DEBUG, INFO, WARN, ERROR}"`
	EnableFile    bool   `default:"true"`
	FileJson      bool   `default:"false"`
	FileLevel     string `default:"ERROR" validate:"oneof:{TRACE, DEBUG, INFO, WARN, ERROR}"`
	FileLocation  string `default:"loadtest.log"`
	EnableColor   bool   `default:"false"`
}

// New returns a newly created and initialized logger with the given settings.
func New(logSettings *Settings) *mlog.Logger {
	logger, _ := mlog.NewLogger()
	cfg, _ := config.MloggerConfigFromLoggerConfig(&model.LogSettings{
		EnableConsole: &logSettings.EnableConsole,
		ConsoleJson:   &logSettings.ConsoleJson,
		ConsoleLevel:  model.NewString(strings.ToLower(logSettings.ConsoleLevel)),
		EnableFile:    &logSettings.EnableFile,
		FileJson:      &logSettings.FileJson,
		FileLevel:     model.NewString(strings.ToLower(logSettings.FileLevel)),
		FileLocation:  &logSettings.FileLocation,
		EnableColor:   &logSettings.EnableColor,
	}, nil, func(filename string) string {
		return logSettings.FileLocation
	})
	logger.ConfigureTargets(cfg, nil)
	return logger
}

// Init initializes the global logger with the given settings.
func Init(logSettings *Settings) {
	logger, _ := mlog.NewLogger()
	cfg, _ := config.MloggerConfigFromLoggerConfig(&model.LogSettings{
		EnableConsole: &logSettings.EnableConsole,
		ConsoleJson:   &logSettings.ConsoleJson,
		ConsoleLevel:  model.NewString(strings.ToLower(logSettings.ConsoleLevel)),
		EnableFile:    &logSettings.EnableFile,
		FileJson:      &logSettings.FileJson,
		FileLevel:     model.NewString(strings.ToLower(logSettings.FileLevel)),
		FileLocation:  &logSettings.FileLocation,
		EnableColor:   &logSettings.EnableColor,
	}, nil, func(filename string) string {
		return logSettings.FileLocation
	})
	logger.ConfigureTargets(cfg, nil)
	// Redirect default golang logger to this logger
	logger.RedirectStdLog(mlog.LvlStdLog)
	// Use this app logger as the global logger
	mlog.InitGlobalLogger(logger)
}
