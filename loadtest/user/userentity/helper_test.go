package userentity

import (
	config2 "github.com/mattermost/mattermost-load-test-ng/cmd/loadtest/config"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/stretchr/testify/require"
)

type TestHelper struct {
	User   *UserEntity
	config *config2.LoadTestConfig
	tb     testing.TB
}

func Setup(tb testing.TB) *TestHelper {
	var th TestHelper
	th.tb = tb
	config, err := config2.GetConfig()
	require.Nil(th.tb, err)
	require.NotNil(th.tb, config)
	th.config = config
	return &th
}

func (th *TestHelper) Init() *TestHelper {
	th.User = th.CreateUser()
	return th
}

func (th *TestHelper) CreateUser() *UserEntity {
	s := memstore.New()
	u := New(s, Config{
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
