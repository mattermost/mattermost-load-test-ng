// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package websocket is a tiny websocket client purpose-built for load-testing.
// It does not have any special features other than strictly what is needed.
package websocket

import (
	"bytes"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
)

const avgReadMsgSizeBytes = 1024

// Client is the websocket client to perform all actions.
type Client struct {
	Url          string
	EventChannel chan *model.WebSocketEvent

	conn      *websocket.Conn
	authToken string
	sequence  int64
	readWg    sync.WaitGroup
	writeMut  sync.RWMutex
}

// NewClient4 constructs a new WebSocket client.
func NewClient4(url, authToken string) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url+model.API_URL_SUFFIX+"/websocket", nil)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Url:          url,
		EventChannel: make(chan *model.WebSocketEvent, 100),

		conn:      conn,
		authToken: authToken,
		sequence:  1,
	}

	client.readWg.Add(1)
	go client.reader()

	client.SendMessage(
		model.WEBSOCKET_AUTHENTICATION_CHALLENGE,
		map[string]interface{}{"token": authToken})

	return client, nil
}

// Close closes the client.
func (c *Client) Close() {
	// If Close gets called concurrently during the time
	// a connection-break happens, this will become a no-op.
	c.conn.Close()
	// Wait for reader to return.
	// If the reader has already quit, this will just fall-through.
	c.readWg.Wait()
}

func (c *Client) reader() {
	defer func() {
		close(c.EventChannel)
		// Mark wg as Done.
		c.readWg.Done()
	}()

	var buf bytes.Buffer
	buf.Grow(avgReadMsgSizeBytes)

	for {
		// Reset buffer.
		buf.Reset()
		_, r, err := c.conn.NextReader()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				// log error
				mlog.Debug("error from conn.NextReader", mlog.Err(err))
			}
			return
		}
		// Use pre-allocated buffer.
		_, err = buf.ReadFrom(r)
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				// log error
				mlog.Warn("error from buf.ReadFrom", mlog.Err(err))
			}
			return
		}

		event := model.WebSocketEventFromJson(&buf)
		if event == nil {
			continue
		}
		if event.IsValid() {
			// non-blocking send in case event channel is full.
			select {
			case c.EventChannel <- event:
			default:
			}
		}
	}
}

// SendMessage is the method to write to the websocket.
func (c *Client) SendMessage(action string, data map[string]interface{}) error {
	// It uses a mutex to synchronize writes.
	// Intentionally no atomics are used to perform additional state tracking.
	// Therefore, we let it fail if the user tries to write again on a closed connection.
	c.writeMut.Lock()
	defer c.writeMut.Unlock()

	req := &model.WebSocketRequest{
		Seq:    c.sequence,
		Action: action,
		Data:   data,
	}

	c.sequence++
	return c.conn.WriteJSON(req)
}

// Helper utilities that call SendMessage.

func (c *Client) UserTyping(channelId, parentId string) error {
	data := map[string]interface{}{
		"channel_id": channelId,
		"parent_id":  parentId,
	}

	return c.SendMessage("user_typing", data)
}

func (c *Client) GetStatuses() error {
	return c.SendMessage("get_statuses", nil)
}

func (c *Client) GetStatusesByIds(userIds []string) error {
	data := map[string]interface{}{
		"user_ids": userIds,
	}
	return c.SendMessage("get_statuses_by_ids", data)
}
