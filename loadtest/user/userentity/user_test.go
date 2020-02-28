// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/stretchr/testify/require"
)

func TestGetUserFromStore(t *testing.T) {
	th := Setup(t).Init()

	user, err := th.User.getUserFromStore()
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Empty(t, user.Id)

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
