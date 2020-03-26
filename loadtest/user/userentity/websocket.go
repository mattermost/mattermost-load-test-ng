// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
)

const (
	// Same as the webapp settings.
	minWebsocketReconnectDuration = 3 * time.Second
	maxWebsocketReconnectDuration = 5 * time.Minute
	maxWebsocketFails             = 7
)

func (ue *UserEntity) handlePostEvent(ev *model.WebSocketEvent) error {
	var data string
	if el, ok := ev.Data["post"]; !ok {
		return fmt.Errorf("post data is missing")
	} else if data, ok = el.(string); !ok {
		return fmt.Errorf("post data not of type string")
	}

	var post *model.Post
	if err := json.Unmarshal([]byte(data), &post); err != nil {
		return err
	}

	switch ev.EventType() {
	case model.WEBSOCKET_EVENT_POSTED, model.WEBSOCKET_EVENT_POST_EDITED:
		return ue.store.SetPost(post)
	case model.WEBSOCKET_EVENT_POST_DELETED:
		return ue.store.DeletePost(post.Id)
	}

	return nil
}

// wsEventHandler handles the given WebSocket event by calling the appropriate
// store methods to make sure the internal user state is kept updated.
// Handling the event at this layer is needed to keep the user state in
// sync with the server. Any response to the event should be made by handling
// the same event at the upper layer (controller).
func (ue *UserEntity) wsEventHandler(ev *model.WebSocketEvent) error {
	switch ev.EventType() {
	case model.WEBSOCKET_EVENT_REACTION_ADDED:
		var reaction *model.Reaction
		if err := json.Unmarshal([]byte(ev.Data["reaction"].(string)), &reaction); err != nil {
			return err
		}
		if err := ue.store.SetReaction(reaction); err != nil {
			return err
		}
	case model.WEBSOCKET_EVENT_REACTION_REMOVED:
		var reaction *model.Reaction
		if err := json.Unmarshal([]byte(ev.Data["reaction"].(string)), &reaction); err != nil {
			return err
		}
		if _, err := ue.store.DeleteReaction(reaction); err != nil {
			return err
		}
	case model.WEBSOCKET_EVENT_POSTED, model.WEBSOCKET_EVENT_POST_EDITED, model.WEBSOCKET_EVENT_POST_DELETED:
		return ue.handlePostEvent(ev)
	}

	return nil
}

// listen starts to listen for messages on various channels.
// It will keep reconnecting if the connection closes.
// Only on calling Disconnect explicitly, it will return.
func (ue *UserEntity) listen(errChan chan error) {
	connectionFailCount := 0
	for {
		client, err := model.NewWebSocketClient4(ue.config.WebSocketURL, ue.client.AuthToken)
		if err != nil {
			errChan <- fmt.Errorf("userentity: websocketClient creation error: %w", err)
			connectionFailCount++
			select {
			case <-ue.wsClosing:
				client.Close()
				// Explicit disconnect. Return.
				close(ue.wsClosed)
				return
			case <-time.After(getWaitTime(connectionFailCount)):
			}
			// Reconnect again.
			continue
		}

		client.Listen()
		chanClosed := false
		for {
			select {
			case ev, ok := <-client.EventChannel:
				if !ok {
					chanClosed = true
					break
				}
				if err := ue.wsEventHandler(ev); err != nil {
					errChan <- fmt.Errorf("userentity: error in wsEventHandler: %w", err)
				}
				ue.wsEventChan <- ev
			case _, ok := <-client.ResponseChannel:
				if !ok {
					chanClosed = true
					break
				}
			case <-ue.wsClosing:
				client.Close()
				// Explicit disconnect. Return.
				close(ue.wsClosed)
				return
			case msg, ok := <-ue.wsTyping:
				if !ok {
					chanClosed = true
					break
				}
				client.UserTyping(msg.channelId, msg.parentId)
			}
			if chanClosed {
				break
			}
		}

		if client.ListenError != nil {
			errChan <- fmt.Errorf("userentity: websocket listen error: %w", client.ListenError)
		}
		connectionFailCount++
		select {
		case <-ue.wsClosing:
			client.Close()
			// Explicit disconnect. Return.
			close(ue.wsClosed)
			return
		case <-time.After(getWaitTime(connectionFailCount)):
		}
		// Reconnect again.
	}
}

// getWaitTime returns the wait time to sleep for.
// This is the same as webapp reconnection logic.
func getWaitTime(failCount int) time.Duration {
	waitTime := minWebsocketReconnectDuration
	if failCount > maxWebsocketFails {
		waitTime *= time.Duration(failCount) * time.Duration(failCount)
		if waitTime > maxWebsocketReconnectDuration {
			waitTime = maxWebsocketReconnectDuration
		}
	}
	return waitTime
}

// SendTypingEvent will push a user_typing event out to all connected users
// who are in the specified channel.
func (ue *UserEntity) SendTypingEvent(channelId, parentId string) {
	ue.wsTyping <- userTypingMsg{
		channelId,
		parentId,
	}
}
