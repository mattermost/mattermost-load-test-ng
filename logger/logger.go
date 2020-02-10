// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package logger

import (
	"strings"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type LoggerSettings struct {
	EnableConsole bool
	ConsoleJson   bool
	ConsoleLevel  string
	EnableFile    bool
	FileJson      bool
	FileLevel     string
	FileLocation  string
}

func Init(logSettings *LoggerSettings) {
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
