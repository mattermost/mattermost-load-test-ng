// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestAddReactionReturnsSaveError(t *testing.T) {
	pinAgentsLoadtestEnv(t)

	config, err := ReadConfig("../../../config/simulcontroller.sample.json")
	require.NoError(t, err)

	store, err := memstore.New(nil)
	require.NoError(t, err)

	user := &model.User{Id: model.NewId(), Username: "user"}
	channel := &model.Channel{Id: model.NewId(), TeamId: model.NewId()}
	post := &model.Post{Id: model.NewId(), ChannelId: channel.Id, UserId: user.Id, CreateAt: 1}

	require.NoError(t, store.SetUser(user))
	require.NoError(t, store.SetChannel(channel))
	require.NoError(t, store.SetCurrentChannel(channel))
	require.NoError(t, store.SetPost(post))
	store.SetClientConfig(map[string]string{"UniqueEmojiReactionLimitPerPost": "0"})

	// Port 1 is never serving the API, so saving the reaction must fail and the
	// action has to report it instead of claiming success.
	ue := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://127.0.0.1:1",
		WebSocketURL: "ws://127.0.0.1:1",
	})

	c, err := New(1, ue, config, make(chan control.UserStatus))
	require.NoError(t, err)

	resp := c.addReaction(ue)
	require.Error(t, resp.Err)
}
