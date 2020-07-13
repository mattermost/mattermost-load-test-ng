// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

type AppInstanceConfig struct {
	Id           string `default:"app0" validate:"notempty"`
	ServerURL    string `default:"http://localhost:8065" validate:"url"`
	WebSocketURL string `default:"ws://localhost:8065" validate:"url"`
}

type AgentInstanceConfig struct {
	Id     string `default:"lt0" validate:"notempty"`
	ApiURL string `default:"http://localhost:4000" validate:"url"`
}

type Config struct {
	AppInstances   []AppInstanceConfig   `validate:"notempty"`
	AgentInstances []AgentInstanceConfig `validate:"notempty"`
}
