// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package userentity

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"
)

type UserEntity struct {
	store       store.MutableUserStore
	client      *model.Client4
	wsClosing   chan struct{}
	wsClosed    chan struct{}
	wsErrorChan chan error
	connected   bool
	config      Config
}

type Config struct {
	ServerURL    string
	WebSocketURL string
}

func (ue *UserEntity) Store() store.UserStore {
	return ue.store
}

func New(store store.MutableUserStore, config Config) *UserEntity {
	ue := UserEntity{}
	ue.config = config
	ue.client = model.NewAPIv4Client(ue.config.ServerURL)
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	ue.client.HttpClient = &http.Client{Transport: transport}
	ue.store = store
	return &ue
}

// Connect creates a websocket connection to the server and starts listening for messages.
func (ue *UserEntity) Connect() <-chan error {
	ue.wsClosing = make(chan struct{})
	ue.wsClosed = make(chan struct{})
	ue.wsErrorChan = make(chan error, 1)
	if ue.client.AuthToken == "" {
		ue.wsErrorChan <- errors.New("user is not authenticated")
		return ue.wsErrorChan
	}
	if ue.connected {
		ue.wsErrorChan <- errors.New("user is already connected")
		return ue.wsErrorChan
	}

	go ue.listen(ue.wsErrorChan)
	ue.connected = true
	return ue.wsErrorChan
}

// Disconnect closes the websocket connection.
func (ue *UserEntity) Disconnect() error {
	if !ue.connected {
		return errors.New("user is not connected")
	}
	// We exit the listener loop first, and then close the connection.
	// Otherwise, it tries to reconnect first, and then
	// exits, which causes unnecessary delay.
	close(ue.wsClosing)

	// We wait to get the response with a timeout, because
	// the loop may be sleeping on a reconnect cycle.
	select {
	case <-ue.wsClosed:
	case <-time.After(minWebsocketReconnectDuration * 2):
	}

	close(ue.wsErrorChan)
	ue.connected = false
	return nil
}

func (ue *UserEntity) getUserFromStore() (*model.User, error) {
	user, err := ue.store.User()

	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("user was not initialized")
	}

	return user, nil
}
