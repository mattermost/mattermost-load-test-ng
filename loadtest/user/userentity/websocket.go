// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package userentity

import (
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

// listen starts to listen for messages on various channels.
// It will keep reconnecting if the connection closes.
// Only on calling Disconnect explicitly, it will return.
func (ue *UserEntity) listen(errChan chan error) {
	connectionFailCount := 0
	for {
		client, err := model.NewWebSocketClient4(ue.config.WebSocketURL, ue.client.AuthToken)
		if err != nil {
			errChan <- fmt.Errorf("websocketClient creation error: %w", err)
			connectionFailCount++
			time.Sleep(getWaitTime(connectionFailCount))
			// Reconnect again.
			continue
		}
		ue.wsClientMut.Lock()
		ue.wsClient = client
		ue.wsClientMut.Unlock()

		ue.wsClient.Listen()
		chanClosed := false
		for {
			select {
			case ev, ok := <-ue.wsClient.EventChannel:
				if !ok {
					chanClosed = true
					break
				}
				_ = ev // TODO: handle event
			case resp, ok := <-ue.wsClient.ResponseChannel:
				if !ok {
					chanClosed = true
					break
				}
				_ = resp // TODO: handle response
			case <-ue.wsClosing:
				// Explicit disconnect. Return.
				close(ue.wsClosed)
				return
			}
			if chanClosed {
				break
			}
		}

		if ue.wsClient.ListenError != nil {
			errChan <- fmt.Errorf("websocket listen error: %w", ue.wsClient.ListenError)
		}
		connectionFailCount++
		time.Sleep(getWaitTime(connectionFailCount))
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
