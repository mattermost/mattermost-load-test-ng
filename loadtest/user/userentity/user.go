// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package userentity

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/model"
)

type UserEntity struct {
	id       int
	store    store.MutableUserStore
	client   *model.Client4
	wsClient *model.WebSocketClient
	config   Config
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
	return &ue
}

func (ue *UserEntity) Connect() error {
	if ue.client.AuthToken == "" {
		return errors.New("user is not authenticated")
	}
	if ue.wsClient != nil {
		return errors.New("user is already connected")
	}
	client, err := model.NewWebSocketClient4(ue.config.WebSocketURL, ue.client.AuthToken)
	if err != nil {
		return err
	}
	ue.wsClient = client
	ue.wsClient.Listen()
	// TODO: implement listener function
	return nil
}

func (ue *UserEntity) Disconnect() error {
	if ue.wsClient == nil {
		return errors.New("user is not connected")
	}
	ue.wsClient.Close()
	ue.wsClient = nil
	return nil
}
