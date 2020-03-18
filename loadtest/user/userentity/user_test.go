// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
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

func TestFetchStaticAssets(t *testing.T) {
	config, err := loadtest.GetConfig()
	require.Nil(t, err)
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()
	indexFetched := false
	assetFetched := false
	indexHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `
		<html>
		<head>
			<script src="./test.js" type="text/javascript">
			 // stuff
			</script>
		</head>
		<body>
		yo
		</body>
		</html>`)
		indexFetched = true
	}
	assetHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `javascript!`)
		assetFetched = true
	}
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/test.js", assetHandler)

	config.ConnectionConfiguration.ServerURL = ts.URL
	th := Setup(t).SetConfig(config).Init()
	err = th.User.FetchStaticAssets()
	require.NoError(t, err)
	require.True(t, indexFetched)
	require.True(t, assetFetched)
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
