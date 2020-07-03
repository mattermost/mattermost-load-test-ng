// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		serverURL := "http://apiserver"
		coord, err := New("", serverURL, nil)
		require.Error(t, err)
		require.Nil(t, coord)
		require.EqualError(t, err, "coordinator: id should not be empty")
	})

	t.Run("empty server url", func(t *testing.T) {
		id := "coord0"
		serverURL := ""
		coord, err := New(id, serverURL, nil)
		require.Error(t, err)
		require.Nil(t, coord)
		require.EqualError(t, err, "coordinator: serverURL should not be empty")
	})

	t.Run("successful creation", func(t *testing.T) {
		id := "coord0"
		serverURL := "http://apiserver"
		coord, err := New(id, serverURL, nil)
		require.NoError(t, err)
		require.NotNil(t, coord)
		require.Equal(t, id, coord.id)
		require.Equal(t, http.DefaultClient, coord.client)
	})

	t.Run("successful creation with custom client", func(t *testing.T) {
		id := "coord0"
		serverURL := "http://apiserver"
		httpClient := &http.Client{}
		coord, err := New(id, serverURL, nil)
		require.NoError(t, err)
		require.NotNil(t, coord)
		require.Equal(t, id, coord.id)
		require.Equal(t, httpClient, coord.client)
	})
}
