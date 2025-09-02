// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/stretchr/testify/require"
)

type ratesDistribution struct {
	Rate       float64 `default:"1.0" validate:"range:[0,)"`
	Percentage float64 `default:"1.0" validate:"range:[0,1]"`
}
type userControllerType string

type config struct {
	ConnectionConfiguration struct {
		ServerURL     string `default:"http://localhost:8065" validate:"url"`
		WebSocketURL  string `default:"ws://localhost:8065" validate:"url"`
		AdminEmail    string `default:"sysadmin@sample.mattermost.com" validate:"email"`
		AdminPassword string `default:"Sys@dmin-sample1" validate:"notempty"`
	}
	UserControllerConfiguration struct {
		Type              userControllerType  `default:"simulative" validate:"oneof:{simple,simulative,noop,cluster,generative}"`
		RatesDistribution []ratesDistribution `default_len:"1"`
		ServerVersion     string
	}
	InstanceConfiguration struct {
		NumTeams                    int64   `default:"2" validate:"range:[0,]"`
		NumChannels                 int64   `default:"10" validate:"range:[0,]"`
		NumPosts                    int64   `default:"0" validate:"range:[0,]"`
		NumReactions                int64   `default:"0" validate:"range:[0,]"`
		NumAdmins                   int64   `default:"0" validate:"range:[0,]"`
		PercentReplies              float64 `default:"0.5" validate:"range:[0,1]"`
		PercentRepliesInLongThreads float64 `default:"0.05" validate:"range:[0,1]"`
		PercentUrgentPosts          float64 `default:"0.001" validate:"range:[0,1]"`
		PercentPublicChannels       float64 `default:"0.2" validate:"range:[0,1]"`
		PercentPrivateChannels      float64 `default:"0.1" validate:"range:[0,1]"`
		PercentDirectChannels       float64 `default:"0.6" validate:"range:[0,1]"`
		PercentGroupChannels        float64 `default:"0.1" validate:"range:[0,1]"`
	}
	UsersConfiguration struct {
		UsersFilePath          string
		InitialActiveUsers     int     `default:"0" validate:"range:[0,$MaxActiveUsers]"`
		MaxActiveUsers         int     `default:"2000" validate:"range:(0,]"`
		MaxActiveBrowserUsers  int     `default:"0" validate:"range:[0,]"`
		AvgSessionsPerUser     int     `default:"1" validate:"range:[1,]"`
		PercentOfUsersAreAdmin float64 `default:"0.0005" validate:"range:[0,1]"`
	}
	BrowserConfiguration struct {
		Headless            bool `default:"true"`
		SimulationTimeoutMs int  `default:"60000" validate:"range:[0,]"`
	}
	LogSettings        logger.Settings
	BrowserLogSettings struct {
		EnableConsole bool   `default:"false"`
		ConsoleLevel  string `default:"error" validate:"oneof:{trace, debug, info, warn, error, fatal}"`
		EnableFile    bool   `default:"true"`
		FileLevel     string `default:"error" validate:"oneof:{trace, debug, info, warn, error, fatal}"`
		FileLocation  string `default:"browseragent.log"`
	}
}

type TestHelper struct {
	User   *UserEntity
	config config
	tb     testing.TB
}

func HelperSetup(tb testing.TB) *TestHelper {
	var th TestHelper
	th.tb = tb
	var cfg config
	err := defaults.ReadFrom("", "../../../config/config.sample.json", &cfg)
	require.Nil(th.tb, err)
	require.NotNil(th.tb, cfg)
	th.config = cfg
	return &th
}

func (th *TestHelper) SetConfig(config config) *TestHelper {
	th.config = config
	return th
}

func (th *TestHelper) Init() *TestHelper {
	th.User = th.CreateUser()
	return th
}

func (th *TestHelper) CreateUser() *UserEntity {
	s, err := memstore.New(nil)
	require.NotNil(th.tb, s)
	require.NoError(th.tb, err)
	u := New(Setup{Store: s}, Config{
		th.config.ConnectionConfiguration.ServerURL,
		th.config.ConnectionConfiguration.WebSocketURL,
		AuthenticationTypeMattermost,
		"testuser",
		"testuser@example.com",
		"testpassword",
	})
	require.NotNil(th.tb, u)
	return u
}

func TestHelperSetup(t *testing.T) {
	th := HelperSetup(t)
	require.NotNil(t, th)
}

func TestInit(t *testing.T) {
	th := HelperSetup(t).Init()
	require.NotNil(t, th)
}
