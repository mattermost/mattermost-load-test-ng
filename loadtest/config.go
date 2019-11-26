// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type LoadTestConfig struct {
	ConnectionConfiguration   ConnectionConfiguration
	UserEntitiesConfiguration UserEntitiesConfiguration
	ResultsConfiguration      ResultsConfiguration
	LogSettings               LoggerSettings
}

type UserEntitiesConfiguration struct {
	TestLengthMinutes                 int
	NumActiveEntities                 int
	ActionRateMilliseconds            int
	ActionRateMaxVarianceMilliseconds int
	EnableRequestTiming               bool
	ChannelLinkChance                 float64
	UploadImageChance                 float64
	LinkPreviewChance                 float64
	CustomEmojiChance                 float64
	CustomEmojiReactionChance         float64
	SystemEmojiReactionChance         float64
	NeedsProfilesByIdChance           float64
	NeedsProfilesByUsernameChance     float64
	NeedsProfileStatusChance          float64
	DoStatusPolling                   bool
	RandomizeEntitySelection          bool
	UserProfileUpdateFullnameChance   float64
	UserProfileUpdateUsernameChance   float64
	UserProfileUpdateNicknameChance   float64
	UserProfileUpdatePositionChance   float64
	UserProfileUpdateEmailChance      float64
	UserProfileUpdateImageChance      float64
	PublicChannelCreationChance       float64
	PrivateChannelCreationChance      float64
	DirectChannelCreationChance       float64
	GroupChannelCreationChance        float64
	NumPostReactionsPerUser           int
	PostReactionsRateMilliseconds     int
	NumPostsGetBeforeAfter            int
	GetPostsAroundLastUnreadChance    float64
	NumGetPostsAroundLastUnread       int
}

type ConnectionConfiguration struct {
	ServerURL                   string
	WebsocketURL                string
	PProfURL                    string
	DriverName                  string
	DataSource                  string
	DBEndpoint                  string // deprecated
	LocalCommands               bool
	SSHHostnamePort             string
	SSHUsername                 string
	SSHPassword                 string
	SSHKey                      string
	MattermostInstallDir        string
	ConfigFileLoc               string
	AdminEmail                  string
	AdminPassword               string
	SkipBulkload                bool
	MaxIdleConns                int
	MaxIdleConnsPerHost         int
	IdleConnTimeoutMilliseconds int
}

type ResultsConfiguration struct {
	PProfDelayMinutes int
	PProfLength       int
}

type LoggerSettings struct {
	EnableConsole bool
	ConsoleJson   bool
	ConsoleLevel  string
	EnableFile    bool
	FileJson      bool
	FileLevel     string
	FileLocation  string
}

func ReadConfig() error {
	viper.SetConfigName("loadtestconfig")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")
	viper.SetEnvPrefix("mmloadtest")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("LogSettings.EnableConsole", true)
	viper.SetDefault("LogSettings.ConsoleLevel", "INFO")
	viper.SetDefault("LogSettings.ConsoleJson", true)
	viper.SetDefault("LogSettings.EnableFile", true)
	viper.SetDefault("LogSettings.FileLevel", "INFO")
	viper.SetDefault("LogSettings.FileJson", true)
	viper.SetDefault("LogSettings.FileLocation", "loadtest.log")
	viper.SetDefault("ConnectionConfiguration.MaxIdleConns", 100)
	viper.SetDefault("ConnectionConfiguration.MaxIdleConnsPerHost", 128)
	viper.SetDefault("ConnectionConfiguration.IdleConnTimeoutMilliseconds", 90000)

	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "unable to read configuration file")
	}

	return nil
}

func GetConfig() (*LoadTestConfig, error) {
	var cfg *LoadTestConfig

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
