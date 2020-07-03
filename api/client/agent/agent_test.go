// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package agent

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		serverURL := "http://apiserver"
		agent, err := New("", serverURL, nil)
		require.Error(t, err)
		require.Nil(t, agent)
		require.EqualError(t, err, "agent: id should not be empty")
	})

	t.Run("empty server url", func(t *testing.T) {
		agent, err := New("agent0", "", nil)
		require.Error(t, err)
		require.Nil(t, agent)
		require.EqualError(t, err, "agent: serverURL should not be empty")
	})

	t.Run("successful creation", func(t *testing.T) {
		id := "agent0"
		serverURL := "http://apiserver"
		agent, err := New(id, serverURL, nil)
		require.NoError(t, err)
		require.NotNil(t, agent)
		require.Equal(t, id, agent.id)
		require.Equal(t, http.DefaultClient, agent.client)
	})

	t.Run("successful creation with custom client", func(t *testing.T) {
		id := "agent0"
		serverURL := "http://apiserver"
		httpClient := &http.Client{}
		agent, err := New(id, serverURL, nil)
		require.NoError(t, err)
		require.NotNil(t, agent)
		require.Equal(t, id, agent.id)
		require.Equal(t, httpClient, agent.client)
	})
}
