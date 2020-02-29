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
	default:
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
