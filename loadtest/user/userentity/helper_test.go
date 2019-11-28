package userentity

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/stretchr/testify/require"
)

type TestHelper struct {
	User   user.User
	config *config.LoadTestConfig
	tb     testing.TB
}

func Setup(tb testing.TB) *TestHelper {
	var th TestHelper
	th.tb = tb
	config, err := config.GetConfig()
	require.Nil(th.tb, err)
	require.NotNil(th.tb, config)
	th.config = config
	return &th
}

func (th *TestHelper) Init() *TestHelper {
	th.User = th.CreateUser()
	return th
}

func (th *TestHelper) CreateUser() user.User {
	s := memstore.New()
	u := New(s, 1, Config{
		th.config.ConnectionConfiguration.ServerURL,
		th.config.ConnectionConfiguration.WebSocketURL,
	})
	require.NotNil(th.tb, u)
	return u
}

func TestSetup(t *testing.T) {
	th := Setup(t)
	require.NotNil(t, th)
}

func TestInit(t *testing.T) {
	th := Setup(t).Init()
	require.NotNil(t, th)
}
