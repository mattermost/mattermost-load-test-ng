// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dummyWebsocketHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		upgrader := &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		conn, err := upgrader.Upgrade(w, req, nil)
		require.Nil(t, err)
		var buf []byte
		for {
			_, buf, err = conn.ReadMessage()
			if err != nil {
				break
			}
			t.Logf("%s\n", buf)
			err = conn.WriteMessage(websocket.TextMessage, []byte("hello world"))
			if err != nil {
				break
			}
		}
	}
}

// TestClose verifies that the client is properly and safely closed in all possible ways.
func TestClose(t *testing.T) {
	s := httptest.NewServer(dummyWebsocketHandler(t))
	defer s.Close()

	checkEventChan := func(eventChan chan *model.WebSocketEvent) {
		defer func() {
			if x := recover(); x == nil {
				require.Fail(t, "should have panicked due to closing a closed channel")
			}
		}()
		close(eventChan)
	}

	t.Run("Sudden", func(t *testing.T) {
		url := strings.Replace(s.URL, "http://", "ws://", 1)
		cli, err := NewClient4(url, "authToken")
		require.Nil(t, err)

		go func() {
			// Just drain the event channel
			for range cli.EventChannel {
			}
		}()

		err = cli.UserTyping("channelId", "parentId")
		assert.Nil(t, err)

		err = cli.conn.Close()
		assert.Nil(t, err)

		// wait for a while for reader to exit
		time.Sleep(200 * time.Millisecond)

		// Verify that event channel is closed.
		checkEventChan(cli.EventChannel)
	})

	t.Run("Normal", func(t *testing.T) {
		url := strings.Replace(s.URL, "http://", "ws://", 1)
		cli, err := NewClient4(url, "authToken")
		require.Nil(t, err)

		go func() {
			// Just drain the event channel
			for range cli.EventChannel {
			}
		}()

		err = cli.UserTyping("channelId", "parentId")
		assert.Nil(t, err)

		cli.Close()

		// Verify that event channel is closed.
		checkEventChan(cli.EventChannel)
	})

	t.Run("Concurrent", func(t *testing.T) {
		url := strings.Replace(s.URL, "http://", "ws://", 1)
		cli, err := NewClient4(url, "authToken")
		require.Nil(t, err)

		go func() {
			// Just drain the event channel
			for range cli.EventChannel {
			}
		}()

		err = cli.UserTyping("channelId", "parentId")
		assert.Nil(t, err)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			cli.Close()
		}()

		go func() {
			defer wg.Done()
			cli.conn.Close()
		}()

		wg.Wait()
		// Verify that event channel is closed.
		checkEventChan(cli.EventChannel)
	})
}

// TestSendMessage verifies that there are no races or panics during message send
// in various conditions.
func TestSendMessage(t *testing.T) {
	s := httptest.NewServer(dummyWebsocketHandler(t))
	defer s.Close()

	t.Run("SendAfterSuddenClose", func(t *testing.T) {
		url := strings.Replace(s.URL, "http://", "ws://", 1)
		cli, err := NewClient4(url, "authToken")
		require.Nil(t, err)

		go func() {
			// Just drain the event channel
			for range cli.EventChannel {
			}
		}()

		err = cli.UserTyping("channelId", "parentId")
		assert.Nil(t, err)

		err = cli.conn.Close()
		assert.Nil(t, err)

		err = cli.UserTyping("channelId2", "parentId2")
		assert.NotNil(t, err)
	})

	t.Run("SendAfterClose", func(t *testing.T) {
		url := strings.Replace(s.URL, "http://", "ws://", 1)
		cli, err := NewClient4(url, "authToken")
		require.Nil(t, err)

		go func() {
			// Just drain the event channel
			for range cli.EventChannel {
			}
		}()

		err = cli.UserTyping("channelId", "parentId")
		assert.Nil(t, err)

		cli.Close()

		err = cli.UserTyping("channelId2", "parentId2")
		assert.NotNil(t, err)
	})

	t.Run("SendDuringSuddenClose", func(t *testing.T) {
		url := strings.Replace(s.URL, "http://", "ws://", 1)
		cli, err := NewClient4(url, "authToken")
		require.Nil(t, err)

		go func() {
			// Just drain the event channel
			for range cli.EventChannel {
			}
		}()

		err = cli.UserTyping("channelId", "parentId")
		assert.Nil(t, err)

		go func() {
			cli.UserTyping("channelId2", "parentId2")
		}()

		err = cli.conn.Close()
		assert.Nil(t, err)
	})

	t.Run("SendDuringClose", func(t *testing.T) {
		url := strings.Replace(s.URL, "http://", "ws://", 1)
		cli, err := NewClient4(url, "authToken")
		require.Nil(t, err)

		go func() {
			// Just drain the event channel
			for range cli.EventChannel {
			}
		}()

		err = cli.UserTyping("channelId", "parentId")
		assert.Nil(t, err)

		go func() {
			cli.UserTyping("channelId2", "parentId2")
		}()

		cli.Close()
	})
}
