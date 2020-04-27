// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package logger

import (
	"strings"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type Settings struct {
	EnableConsole bool   `default:"true"`
	ConsoleJson   bool   `default:"false"`
	ConsoleLevel  string `default:"ERROR" validate:"oneof:{TRACE, INFO, WARN, ERROR}"`
	EnableFile    bool   `default:"true"`
	FileJson      bool   `default:"false"`
	FileLevel     string `default:"ERROR" validate:"oneof:{TRACE, INFO, WARN, ERROR}"`
	FileLocation  string `default:"loadtest.log"`
}

func Init(logSettings *Settings) {
	log := mlog.NewLogger(&mlog.LoggerConfiguration{
		EnableConsole: logSettings.EnableConsole,
		ConsoleJson:   logSettings.ConsoleJson,
		ConsoleLevel:  strings.ToLower(logSettings.ConsoleLevel),
		EnableFile:    logSettings.EnableFile,
		FileJson:      logSettings.FileJson,
		FileLevel:     strings.ToLower(logSettings.FileLevel),
		FileLocation:  logSettings.FileLocation,
	})

	// Redirect default golang logger to this logger
	mlog.RedirectStdLog(log)

	// Use this app logger as the global logger
	mlog.InitGlobalLogger(log)
}
