// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/stretchr/testify/require"
)

func TestSetRate(t *testing.T) {
	c, err := New(1, &userentity.UserEntity{}, make(chan control.UserStatus))
	require.Nil(t, err)
	require.Equal(t, 1.0, c.rate)

	err = c.SetRate(-1.0)
	require.NotNil(t, err)
	require.Equal(t, 1.0, c.rate)

	err = c.SetRate(0.0)
	require.Nil(t, err)
	require.Equal(t, 0.0, c.rate)

	err = c.SetRate(1.5)
	require.Nil(t, err)
	require.Equal(t, 1.5, c.rate)
}

func TestRunStop(t *testing.T) {
	store := memstore.New()

	user := userentity.New(store, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
	})
	statusChan := make(chan control.UserStatus)
	c, err := New(1, user, statusChan)
	require.Nil(t, err)

	doneRunning := make(chan struct{})
	go func() {
		c.Run()
		close(doneRunning)
	}()

	status := <-statusChan
	require.NoError(t, status.Err)
	require.Equal(t, "user started", status.Info)

	doneHandlingStatus := make(chan struct{})
	go func() {
		var last control.UserStatus
		for {
			status, ok := <-statusChan
			if !ok {
				require.Equal(t, "user stopped", last.Info)
				break
			}
			last = status
		}
		close(doneHandlingStatus)
	}()

	c.Stop()
	<-doneRunning
	close(statusChan)
	<-doneHandlingStatus
}
