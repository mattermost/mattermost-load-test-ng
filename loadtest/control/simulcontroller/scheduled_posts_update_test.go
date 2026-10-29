// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestUpdateScheduledPostUsesItsOwnChannel(t *testing.T) {
	pinAgentsLoadtestEnv(t)

	const serverMessage = "message returned by the server"
	var mut sync.Mutex
	var updatedChannelId string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This runs in the server goroutine, so it must not use require, which
		// calls FailNow. The test body still fails through resp.Err.
		sp := &model.ScheduledPost{}
		if err := json.NewDecoder(r.Body).Decode(sp); err != nil {
			t.Errorf("decoding scheduled post: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mut.Lock()
		updatedChannelId = sp.ChannelId
		mut.Unlock()
		sp.Message = serverMessage
		if err := json.NewEncoder(w).Encode(sp); err != nil {
			t.Errorf("encoding scheduled post: %v", err)
		}
	}))
	defer srv.Close()

	config, err := ReadConfig("../../../config/simulcontroller.sample.json")
	require.NoError(t, err)

	store, err := memstore.New(nil)
	require.NoError(t, err)

	user := &model.User{Id: model.NewId(), Username: "user"}
	// The scheduled post lives in a channel of one team, while the user is
	// looking at a channel of another team.
	scheduledChannel := &model.Channel{Id: model.NewId(), TeamId: model.NewId()}
	currentChannel := &model.Channel{Id: model.NewId(), TeamId: model.NewId()}
	scheduledPost := &model.ScheduledPost{
		Draft: model.Draft{
			UserId:    user.Id,
			ChannelId: scheduledChannel.Id,
			Message:   "original",
		},
		Id: model.NewId(),
	}

	require.NoError(t, store.SetUser(user))
	require.NoError(t, store.SetChannel(scheduledChannel))
	require.NoError(t, store.SetChannel(currentChannel))
	require.NoError(t, store.SetCurrentChannel(currentChannel))
	require.NoError(t, store.SetScheduledPost(scheduledChannel.TeamId, scheduledPost))
	store.SetClientConfig(map[string]string{"ScheduledPosts": "true"})

	ue := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    srv.URL,
		WebSocketURL: "ws" + strings.TrimPrefix(srv.URL, "http"),
	})

	c, err := New(1, ue, config, make(chan control.UserStatus))
	require.NoError(t, err)

	resp := c.updateScheduledPost(ue)
	require.NoError(t, resp.Err)
	mut.Lock()
	require.Equal(t, scheduledChannel.Id, updatedChannelId)
	mut.Unlock()

	// The scheduled post is still stored once, under its own team, with the
	// version the server returned. Drawing many times makes sure that no
	// second copy was filed under the other team.
	distinct := map[*model.ScheduledPost]bool{}
	for i := 0; i < 100; i++ {
		sp, err := store.GetRandomScheduledPost()
		require.NoError(t, err)
		distinct[sp] = true
		require.Equal(t, serverMessage, sp.Message)
	}
	require.Len(t, distinct, 1)
}
