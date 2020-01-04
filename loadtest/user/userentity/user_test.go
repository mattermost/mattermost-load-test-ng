package userentity

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/stretchr/testify/require"
)

func TestGetUserFromStore(t *testing.T) {
	th := Setup(t).Init()

	user, err := th.User.getUserFromStore()
	require.Nil(t, user)
	require.Error(t, err)
	require.EqualError(t, err, "user was not initialized")

	err = th.User.store.SetUser(&model.User{
		Id: "someid",
	})
	require.NoError(t, err)
	user, err = th.User.getUserFromStore()
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, "someid", user.Id)
}

func TestIsSysAdmin(t *testing.T) {
	th := Setup(t).Init()

	err := th.User.store.SetUser(&model.User{
		Id:    "someid",
		Roles: "system_user",
	})
	require.NoError(t, err)

	user, err := th.User.getUserFromStore()
	require.NoError(t, err)

	ok, err := th.User.IsSysAdmin()
	require.NoError(t, err)
	require.False(t, ok)

	user.Roles = "system_user system_admin"
	ok, err = th.User.IsSysAdmin()
	require.NoError(t, err)
	require.True(t, ok)
}
