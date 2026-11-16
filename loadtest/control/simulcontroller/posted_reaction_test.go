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

func TestPostedBurst(t *testing.T) {
	var b postedBurst
	now := time.Now()

	for i := 0; i < postedImmediateLimit; i++ {
		require.False(t, b.batched(now.Add(time.Duration(i)*10*time.Millisecond)), "event %d", i)
	}
	last := now.Add(time.Duration(postedImmediateLimit) * 10 * time.Millisecond)
	require.True(t, b.batched(last))
	require.True(t, b.batched(last.Add(10*time.Millisecond)))

	// A pause longer than the debounce window starts a new burst.
	require.False(t, b.batched(last.Add(time.Second)))
}

type postedFixture struct {
	c       *SimulController
	store   *memstore.MemStore
	me      *model.User
	author  *model.User
	channel *model.Channel
	mut     sync.Mutex
	paths   []string
}

func (f *postedFixture) count(substr string) int {
	f.mut.Lock()
	defer f.mut.Unlock()
	n := 0
	for _, p := range f.paths {
		if strings.Contains(p, substr) {
			n++
		}
	}
	return n
}

func newPostedFixture(t *testing.T) *postedFixture {
	t.Helper()
	pinAgentsLoadtestEnv(t)

	f := &postedFixture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mut.Lock()
		f.paths = append(f.paths, r.Method+" "+r.URL.Path)
		f.mut.Unlock()

		switch {
		case strings.HasSuffix(r.URL.Path, "/thread"):
			root := strings.Split(r.URL.Path, "/")[4]
			pl := model.NewPostList()
			pl.AddPost(&model.Post{Id: root, ChannelId: f.channel.Id, UserId: f.author.Id})
			pl.AddOrder(root)
			_ = json.NewEncoder(w).Encode(pl)
		case strings.HasSuffix(r.URL.Path, "/view"):
			_ = json.NewEncoder(w).Encode(&model.ChannelViewResponse{Status: "OK"})
		default:
			_, _ = w.Write([]byte("[]"))
		}
	}))
	t.Cleanup(srv.Close)

	config, err := ReadConfig("../../../config/simulcontroller.sample.json")
	require.NoError(t, err)
	config.ReactToPostedEvents = true

	f.store, err = memstore.New(nil)
	require.NoError(t, err)

	f.me = &model.User{Id: model.NewId(), Username: "me"}
	f.author = &model.User{Id: model.NewId(), Username: "author"}
	f.channel = &model.Channel{Id: model.NewId(), TeamId: model.NewId()}
	require.NoError(t, f.store.SetUser(f.me))
	require.NoError(t, f.store.SetUsers([]*model.User{f.author}))
	require.NoError(t, f.store.SetStatus(f.author.Id, &model.Status{UserId: f.author.Id, Status: "online"}))
	require.NoError(t, f.store.SetChannel(f.channel))
	require.NoError(t, f.store.SetCurrentChannel(f.channel))
	f.store.SetClientConfig(map[string]string{"CollapsedThreads": model.CollapsedThreadsDisabled})

	ue := userentity.New(userentity.Setup{Store: f.store}, userentity.Config{
		ServerURL:    srv.URL,
		WebSocketURL: "ws" + strings.TrimPrefix(srv.URL, "http"),
	})

	f.c, err = New(1, ue, config, make(chan control.UserStatus))
	require.NoError(t, err)
	return f
}

func (f *postedFixture) post(userId, channelId string) *model.Post {
	p := &model.Post{Id: model.NewId(), UserId: userId, ChannelId: channelId, Message: "hello"}
	// The user entity stores the posts of the current channel before the
	// controller sees the event.
	if channelId == f.channel.Id {
		_ = f.store.SetPost(p)
	}
	return p
}

func TestReactToPostedEventMarksChannelRead(t *testing.T) {
	f := newPostedFixture(t)

	require.NoError(t, f.c.reactToPostedEvent(f.post(f.author.Id, f.channel.Id), false))
	require.Equal(t, 1, f.count("/view"))
	require.Equal(t, 0, f.count("/users/ids"), "known author must not be fetched again")
	require.Equal(t, 0, f.count("/image"), "known author picture must not be fetched again")
}

func TestReactToPostedEventSkips(t *testing.T) {
	f := newPostedFixture(t)

	// Batched events do not mark the channel as read.
	require.NoError(t, f.c.reactToPostedEvent(f.post(f.author.Id, f.channel.Id), true))
	// Own posts are marked as read locally only.
	require.NoError(t, f.c.reactToPostedEvent(f.post(f.me.Id, f.channel.Id), false))
	// System messages are ignored.
	sys := f.post(f.author.Id, f.channel.Id)
	sys.Type = model.PostTypeJoinChannel
	require.NoError(t, f.c.reactToPostedEvent(sys, false))
	// Posts in other channels do nothing.
	require.NoError(t, f.c.reactToPostedEvent(f.post(f.author.Id, model.NewId()), false))

	require.Empty(t, f.paths)
}

func TestReactToPostedEventFetchesMissingRootAndAuthor(t *testing.T) {
	f := newPostedFixture(t)

	stranger := model.NewId()
	reply := f.post(stranger, f.channel.Id)
	reply.RootId = model.NewId()

	// The mocked users endpoints return no users, which is fine: we only
	// check which calls are made.
	require.NoError(t, f.c.reactToPostedEvent(reply, false))
	require.Equal(t, 1, f.count("GET /api/v4/posts/"+reply.RootId+"/thread"))
	require.Equal(t, 1, f.count("POST /api/v4/users/ids"))
	require.Equal(t, 1, f.count("/image"))
	require.Equal(t, 1, f.count("POST /api/v4/users/status/ids"))
	require.Equal(t, 1, f.count("/view"))

	// A mention of an unknown user is looked up by username.
	f.mut.Lock()
	f.paths = nil
	f.mut.Unlock()
	mention := f.post(f.author.Id, f.channel.Id)
	mention.Message = "hi @someoneelse"
	require.NoError(t, f.c.reactToPostedEvent(mention, true))
	require.Equal(t, 1, f.count("POST /api/v4/users/usernames"))

	// With the root already loaded the thread is not fetched.
	f.mut.Lock()
	f.paths = nil
	f.mut.Unlock()
	require.NoError(t, f.store.SetPost(&model.Post{Id: reply.RootId, ChannelId: f.channel.Id, UserId: f.author.Id}))
	require.NoError(t, f.c.reactToPostedEvent(reply, false))
	require.Equal(t, 0, f.count("/thread"))
}
