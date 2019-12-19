// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package userentity

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"
)

type UserEntity struct {
	id          int
	store       store.MutableUserStore
	client      *model.Client4
	wsClientMut sync.RWMutex
	wsClient    *model.WebSocketClient
	closing     chan struct{}
	closed      chan struct{}
	config      Config
}

type Config struct {
	ServerURL    string
	WebSocketURL string
}

func (ue *UserEntity) Id() int {
	return ue.id
}

func (ue *UserEntity) Store() store.UserStore {
	return ue.store
}

func New(store store.MutableUserStore, id int, config Config) *UserEntity {
	ue := UserEntity{}
	ue.id = id
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
	ue.closing = make(chan struct{})
	ue.closed = make(chan struct{})
	return &ue
}

// Connect creates a websocket connection to the server and starts listening for messages.
func (ue *UserEntity) Connect() error {
	if ue.client.AuthToken == "" {
		return errors.New("user is not authenticated")
	}
	cli := ue.getWsClient()
	if cli != nil {
		return errors.New("user is already connected")
	}

	go ue.listen()
	return nil
}

// Disconnect closes the websocket connection.
func (ue *UserEntity) Disconnect() error {
	cli := ue.getWsClient()
	if cli == nil {
		return errors.New("user is not connected")
	}
	cli.Close()
	cli = nil
	// Wait for the listener to shut down.
	close(ue.closing)
	<-ue.closed
	return nil
}

// SendWebsocketMessage sends a given action type with data to the websocket.
func (ue *UserEntity) SendWebsocketMessage(action string, data map[string]interface{}) error {
	cli := ue.getWsClient()
	if cli == nil {
		return errors.New("user is not connected")
	}
	cli.SendMessage(action, data)
	return nil
}

// getWsClient is a simple mutex wrapper to access the underlying client object.
func (ue *UserEntity) getWsClient() *model.WebSocketClient {
	ue.wsClientMut.RLock()
	defer ue.wsClientMut.RUnlock()
	return ue.wsClient
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
