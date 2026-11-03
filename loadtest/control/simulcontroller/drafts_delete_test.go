// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
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

func TestDeleteDraftTargetsTheDraftAndUpdatesTheStore(t *testing.T) {
	pinAgentsLoadtestEnv(t)

	var mut sync.Mutex
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mut.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mut.Unlock()
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	config, err := ReadConfig("../../../config/simulcontroller.sample.json")
	require.NoError(t, err)

	store, err := memstore.New(nil)
	require.NoError(t, err)

	user := &model.User{Id: model.NewId(), Username: "user"}
	team := &model.Team{Id: model.NewId()}
	// The user is looking at a channel that has no draft at all.
	currentChannel := &model.Channel{Id: model.NewId(), TeamId: team.Id}
	channelDraft := &model.Draft{UserId: user.Id, ChannelId: model.NewId()}
	threadDraft := &model.Draft{UserId: user.Id, ChannelId: model.NewId(), RootId: model.NewId()}

	require.NoError(t, store.SetUser(user))
	require.NoError(t, store.SetCurrentTeam(team))
	require.NoError(t, store.SetChannel(currentChannel))
	require.NoError(t, store.SetCurrentChannel(currentChannel))
	require.NoError(t, store.SetDrafts(team.Id, []*model.Draft{channelDraft, threadDraft}))
	store.SetClientConfig(map[string]string{"AllowSyncedDrafts": "true"})

	ue := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    srv.URL,
		WebSocketURL: "ws" + strings.TrimPrefix(srv.URL, "http"),
	})

	c, err := New(1, ue, config, make(chan control.UserStatus))
	require.NoError(t, err)

	// One call per draft, then there is nothing left to delete.
	for i := 0; i < 2; i++ {
		resp := c.deleteDraft(ue)
		require.NoError(t, resp.Err)
	}
	resp := c.deleteDraft(ue)
	require.NoError(t, resp.Err)
	require.Contains(t, resp.Info, "no drafts found")

	mut.Lock()
	defer mut.Unlock()
	require.ElementsMatch(t, []string{
		"DELETE /api/v4/users/" + user.Id + "/channels/" + channelDraft.ChannelId + "/drafts",
		"DELETE /api/v4/users/" + user.Id + "/channels/" + threadDraft.ChannelId + "/drafts/" + threadDraft.RootId,
	}, requests)
}
