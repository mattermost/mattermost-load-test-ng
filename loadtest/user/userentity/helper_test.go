// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"

	"github.com/stretchr/testify/require"
)

type config struct {
	ConnectionConfiguration struct {
		ServerURL     string `default:"http://localhost:8065" validate:"url"`
		WebSocketURL  string `default:"ws://localhost:8065" validate:"url"`
		AdminEmail    string `default:"sysadmin@sample.mattermost.com" validate:"email"`
		AdminPassword string `default:"Sys@dmin-sample1" validate:"notempty"`
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
	err := defaults.ReadFromJSON("", "../../../config/config.sample.json", &cfg)
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
