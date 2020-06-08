// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"

	"github.com/stretchr/testify/require"
)

type TestHelper struct {
	User   *UserEntity
	config *loadtest.Config
	tb     testing.TB
}

func HelperSetup(tb testing.TB) *TestHelper {
	var th TestHelper
	th.tb = tb
	config, err := loadtest.ReadConfig("../../../config/config.sample.json")
	require.Nil(th.tb, err)
	require.NotNil(th.tb, config)
	th.config = config
	return &th
}

func (th *TestHelper) SetConfig(config *loadtest.Config) *TestHelper {
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
