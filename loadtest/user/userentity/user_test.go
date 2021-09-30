// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/require"
)

func TestGetUserFromStore(t *testing.T) {
	th := HelperSetup(t).Init()

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

func TestFetchStaticAssets(t *testing.T) {
	var cfg config
	err := defaults.ReadFromJSON("", "../../../config/config.sample.json", &cfg)
	require.Nil(t, err)
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()
	indexFetched := false
	jsFetched := false
	cssFetched := false
	indexHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `
		<html>
		<head>
			<script src="/static/test.js" type="text/javascript">
			 // stuff
			</script>
			<link href="/static/test.css" rel="stylesheet" />
		</head>
		<body>
		yo
		</body>
		</html>`)
		indexFetched = true
	}
	jsHandler := func(w http.ResponseWriter, r *http.Request) {
		jsFetched = true
	}
	cssHandler := func(w http.ResponseWriter, r *http.Request) {
		cssFetched = true
	}
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/static/test.js", jsHandler)
	mux.HandleFunc("/static/test.css", cssHandler)
	cfg.ConnectionConfiguration.ServerURL = ts.URL
	th := HelperSetup(t).SetConfig(cfg).Init()
	err = th.User.FetchStaticAssets()
	require.NoError(t, err)
	require.True(t, indexFetched)
	require.True(t, jsFetched)
	require.True(t, cssFetched)
}

func TestIsSysAdmin(t *testing.T) {
	th := HelperSetup(t).Init()

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
