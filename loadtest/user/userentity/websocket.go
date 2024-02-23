// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/websocket"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const (
	// Same as the webapp settings.
	minWebsocketReconnectDuration = 3 * time.Second
	maxWebsocketReconnectDuration = 5 * time.Minute
	maxWebsocketFails             = 7
)

var errSeqMismatch = errors.New("mismatch in server sequence number")

func (ue *UserEntity) handleReactionEvent(ev *model.WebSocketEvent) error {
	var data string
	if el, ok := ev.GetData()["reaction"]; !ok {
		return errors.New("reaction data is missing")
	} else if data, ok = el.(string); !ok {
		return fmt.Errorf("type of the reaction data should be a string, but it is %T", el)
	}

	var reaction *model.Reaction
	if err := json.Unmarshal([]byte(data), &reaction); err != nil {
		return err
	}

	currentChannel, err := ue.store.CurrentChannel()
	if err != nil && !errors.Is(err, memstore.ErrChannelNotFound) {
		return fmt.Errorf("failed to get current channel from store: %w", err)
	} else if currentChannel == nil {
		return nil
	}

	post, err := ue.store.Post(reaction.PostId)
	if errors.Is(err, memstore.ErrPostNotFound) || post.ChannelId != currentChannel.Id {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get post from store: %w", err)
	}

	switch ev.EventType() {
	case model.WebsocketEventReactionAdded:
		return ue.store.SetReaction(reaction)
	case model.WebsocketEventReactionRemoved:
		if ok, err := ue.store.DeleteReaction(reaction); err != nil {
			return err
		} else if !ok {
			return errors.New("could not find reaction in the store")
		}
	}

	return nil
}

func (ue *UserEntity) handlePostEvent(ev *model.WebSocketEvent) error {
	var data string
	if el, ok := ev.GetData()["post"]; !ok {
		return errors.New("post data is missing")
	} else if data, ok = el.(string); !ok {
		return fmt.Errorf("type of the post data should be a string, but it is %T", el)
	}

	var post *model.Post
	if err := json.Unmarshal([]byte(data), &post); err != nil {
		return err
	}

	switch ev.EventType() {
	case model.WebsocketEventPosted, model.WebsocketEventPostEdited:
		currentChannel, err := ue.store.CurrentChannel()
		if err == nil && currentChannel.Id == post.ChannelId {
			return ue.store.SetPost(post)
		} else if err != nil && !errors.Is(err, memstore.ErrChannelNotFound) {
			return fmt.Errorf("failed to get current channel from store: %w", err)
		}
	case model.WebsocketEventPostDeleted:
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
	if ev.EventType() == model.WebsocketEventHello {
		if connID, ok := ev.GetData()["connection_id"].(string); ok {
			// If we already have a connectionId present, and server sends a different one,
			// that means it's either a long timeout, or server restart, or sequence number is not found.
			// Then we reset sequence number to 0.
			if ue.wsConnID != "" && ue.wsConnID != connID {
				mlog.Debug("Long timeout, or server restart, or sequence number not found")
				// In future, we can add the missed event callback here.
				ue.wsServerSeq = 0
			}
			ue.wsConnID = connID
		}
	}

	// Now we check for sequence number, and if it does not match,
	// we just disconnect and reconnect.
	if ev.GetSequence() != ue.wsServerSeq {
		mlog.Warn("Missed websocket event", mlog.Int("got", ev.GetSequence()), mlog.Int("expected", ue.wsServerSeq))
		return errSeqMismatch
	}

	ue.wsServerSeq = ev.GetSequence() + 1

	switch ev.EventType() {
	case model.WebsocketEventReactionAdded, model.WebsocketEventReactionRemoved:
		return ue.handleReactionEvent(ev)
	case model.WebsocketEventPosted, model.WebsocketEventPostEdited, model.WebsocketEventPostDeleted:
		return ue.handlePostEvent(ev)
	}

	return nil
}

// listen starts to listen for messages on various channels.
// It will keep reconnecting if the connection closes.
// Only on calling Disconnect explicitly, it will return.
func (ue *UserEntity) listen(errChan chan error) {
	connectionFailCount := 0
start:
	for {
		client, err := websocket.NewClient4(&websocket.ClientParams{
			WsURL:          ue.config.WebSocketURL,
			AuthToken:      ue.client.AuthToken,
			ConnID:         ue.wsConnID,
			ServerSequence: ue.wsServerSeq,
		})
		if err != nil {
			errChan <- fmt.Errorf("userentity: websocketClient creation error: %w", err)
			connectionFailCount++
			select {
			// Draining the channel to avoid blocking the sender.
			case <-ue.dataChan:
			case <-ue.wsClosing:
				// Explicit disconnect. Return.
				close(ue.wsClosed)
				return
			case <-time.After(getWaitTime(connectionFailCount)):
			}
			// Reconnect again.
			continue
		}

		ue.incWebSocketConnections()

		var chanClosed bool
		for {
			select {
			case ev, ok := <-client.EventChannel:
				if !ok {
					chanClosed = true
					break
				}
				if err := ue.wsEventHandler(ev); err != nil {
					if err == errSeqMismatch {
						// Disconnect and reconnect.
						client.Close()
						ue.decWebSocketConnections()
						continue start
					}
					errChan <- fmt.Errorf("userentity: error in wsEventHandler: %w", err)
				}
				ue.wsEventChan <- ev
			case <-ue.wsClosing:
				client.Close()
				ue.decWebSocketConnections()
				// Explicit disconnect. Return.
				close(ue.wsClosed)
				return
			case msg, ok := <-ue.dataChan:
				if !ok {
					chanClosed = true
					break
				}
				switch v := msg.(type) {
				case userTypingMsg:
					if err := client.UserTyping(v.channelId, v.parentId); err != nil {
						errChan <- fmt.Errorf("userentity: error in client.UserTyping: %w", err)
					}
				case threadPresenceMsg:
					if err := client.UpdateActiveThread(v.channelId, v.threadView); err != nil {
						errChan <- fmt.Errorf("userentity: error in client.UpdateActiveThread: %w", err)
					}
				case channelPresenceMsg:
					if err := client.UpdateActiveChannel(v.channelId); err != nil {
						errChan <- fmt.Errorf("userentity: error in client.UpdateActiveChannel: %w", err)
					}
				case teamPresenceMsg:
					if err := client.UpdateActiveTeam(v.teamId); err != nil {
						errChan <- fmt.Errorf("userentity: error in client.UpdateActiveTeam: %w", err)
					}
				}
			}
			if chanClosed {
				client.Close()
				break
			}
		}

		ue.decWebSocketConnections()

		connectionFailCount++
		select {
		// Draining the channel to avoid blocking the sender.
		case <-ue.dataChan:
		case <-ue.wsClosing:
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
func (ue *UserEntity) SendTypingEvent(channelId, parentId string) error {
	if !ue.connected {
		return errors.New("user is not connected")
	}
	ue.dataChan <- userTypingMsg{
		channelId: channelId,
		parentId:  parentId,
	}
	return nil
}

func (ue *UserEntity) UpdateActiveChannel(channelId string) error {
	if !ue.connected {
		return errors.New("user is not connected")
	}
	ue.dataChan <- channelPresenceMsg{
		channelId: channelId,
	}
	return nil
}

func (ue *UserEntity) UpdateActiveThread(channelId string) error {
	if !ue.connected {
		return errors.New("user is not connected")
	}
	// We don't really have a notion of RHS thread vs global thread in the load test.
	// We either load a channel or load a thread. For now, we just set `is_thread_view`
	// as both true and false to set the scope.
	ue.dataChan <- threadPresenceMsg{
		channelId:  channelId,
		threadView: true,
	}
	ue.dataChan <- threadPresenceMsg{
		channelId:  channelId,
		threadView: false,
	}
	return nil
}

func (ue *UserEntity) UpdateActiveTeam(teamId string) error {
	if !ue.connected {
		return errors.New("user is not connected")
	}
	ue.dataChan <- teamPresenceMsg{
		teamId: teamId,
	}
	return nil
}
