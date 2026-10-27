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

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestSearchGroupChannelsSendsTypedTerm(t *testing.T) {
	var mut sync.Mutex
	var terms []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var search model.ChannelSearch
		require.NoError(t, json.NewDecoder(r.Body).Decode(&search))
		mut.Lock()
		terms = append(terms, search.Term)
		mut.Unlock()
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	store, err := memstore.New(nil)
	require.NoError(t, err)

	const username = "averylongusername"
	self := &model.User{Id: model.NewId(), Username: "self"}
	other := &model.User{Id: model.NewId(), Username: username}
	require.NoError(t, store.SetUser(self))
	require.NoError(t, store.SetUsers([]*model.User{self, other}))

	ue := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    srv.URL,
		WebSocketURL: "ws" + strings.TrimPrefix(srv.URL, "http"),
	})

	resp := searchGroupChannels(ue)
	require.NoError(t, resp.Err)

	mut.Lock()
	defer mut.Unlock()
	require.NotEmpty(t, terms)
	for _, term := range terms {
		// The user is typing, so each request carries a short prefix of the
		// name (possibly followed by a mistyped rune), never the whole name.
		require.NotEqual(t, username, term)
		require.LessOrEqual(t, len(term), 5)
		require.True(t, strings.HasPrefix(username, term[:1]), "term %q does not start like %q", term, username)
	}
}
