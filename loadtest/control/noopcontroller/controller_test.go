// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package noopcontroller

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	ch := make(chan control.UserStatus)
	c, err := New(77, &userentity.UserEntity{}, ch)
	require.Nil(t, err)

	require.Equal(t, len(c.actionList), len(c.actionMap))
}
