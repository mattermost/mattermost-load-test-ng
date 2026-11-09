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
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestScrollChannelBackPagesBackwards(t *testing.T) {
	pinAgentsLoadtestEnv(t)

	channel := &model.Channel{Id: model.NewId(), TeamId: model.NewId()}
	userId := model.NewId()
	newPost := func(createAt int64) *model.Post {
		return &model.Post{Id: model.NewId(), ChannelId: channel.Id, UserId: userId, CreateAt: createAt}
	}

	// Two posts are loaded when the user starts scrolling. The server has two
	// older pages, the second one being the last one.
	loaded := []*model.Post{newPost(1000), newPost(2000)}
	page1 := []*model.Post{newPost(500), newPost(600)}
	page2 := []*model.Post{newPost(100), newPost(200)}
	pages := map[string][]*model.Post{
		loaded[0].Id: page1,
		page1[0].Id:  page2,
	}

	var mut sync.Mutex
	var cursors []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before := r.URL.Query().Get("before")
		if !strings.HasSuffix(r.URL.Path, "/posts") || before == "" {
			_, _ = w.Write([]byte("[]"))
			return
		}

		mut.Lock()
		cursors = append(cursors, before)
		mut.Unlock()

		list := model.NewPostList()
		for _, p := range pages[before] {
			list.AddPost(p)
			list.AddOrder(p.Id)
		}
		require.NoError(t, json.NewEncoder(w).Encode(list))
	}))
	defer srv.Close()

	config, err := ReadConfig("../../../config/simulcontroller.sample.json")
	require.NoError(t, err)

	store, err := memstore.New(nil)
	require.NoError(t, err)
	require.NoError(t, store.SetUser(&model.User{Id: userId, Username: "user"}))
	require.NoError(t, store.SetChannel(channel))
	require.NoError(t, store.SetCurrentChannel(channel))
	require.NoError(t, store.SetPosts(loaded))

	ue := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    srv.URL,
		WebSocketURL: "ws" + strings.TrimPrefix(srv.URL, "http"),
	})

	c, err := New(1, ue, config, make(chan control.UserStatus))
	require.NoError(t, err)

	// Ask for more scrolls than there are pages: the action has to stop on its
	// own once the server has nothing older to return.
	resp := c.scrollChannelBack(ue, channel.Id, false, 5, func() time.Duration { return 0 })
	require.NoError(t, resp.Err)

	mut.Lock()
	defer mut.Unlock()
	// Each scroll starts from the oldest post known so far, and the third
	// request, which comes back empty, ends the scrolling.
	require.Equal(t, []string{loaded[0].Id, page1[0].Id, page2[0].Id}, cursors)
	require.Equal(t, "scrolled channel "+channel.Id+" 3 times", resp.Info)
}
