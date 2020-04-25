// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"
)

// UserEntity is an implementation of the User interface
// which provides methods to interact with the Mattermost server.
type UserEntity struct {
	store       store.MutableUserStore
	client      *model.Client4
	wsClosing   chan struct{}
	wsClosed    chan struct{}
	wsErrorChan chan error
	wsEventChan chan *model.WebSocketEvent
	wsTyping    chan userTypingMsg
	connected   bool
	config      Config
}

// Config holds necessary information required by a UserEntity.
type Config struct {
	// The URL of the Mattermost web server.
	ServerURL string
	// The URL of the mattermost WebSocket server.
	WebSocketURL string
	// The username to be used by the entity.
	Username string
	// The email to be used by the entity.
	Email string
	// The password to be used by the entity.
	Password string
}

type userTypingMsg struct {
	channelId string
	parentId  string
}

// Store returns the underlying store of the user.
func (ue *UserEntity) Store() store.UserStore {
	return ue.store
}

// New returns a new instance of a UserEntity.
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
		MaxIdleConns:          10000,
		MaxIdleConnsPerHost:   10000,
		IdleConnTimeout:       5 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	ue.client.HttpClient = &http.Client{Transport: transport}
	err := store.SetUser(&model.User{
		Username: config.Username,
		Email:    config.Email,
		Password: config.Password,
	})
	if err != nil {
		return nil
	}
	ue.store = store
	ue.wsEventChan = make(chan *model.WebSocketEvent)
	ue.wsTyping = make(chan userTypingMsg)
	return &ue
}

// Connect creates a websocket connection to the server and starts listening for messages.
func (ue *UserEntity) Connect() (<-chan error, error) {
	if ue.connected {
		return nil, errors.New("user is already connected")
	}
	ue.wsClosing = make(chan struct{})
	ue.wsClosed = make(chan struct{})
	ue.wsErrorChan = make(chan error, 1)
	if ue.client.AuthToken == "" {
		return nil, errors.New("user is not authenticated")
	}
	if ue.connected {
		return nil, errors.New("user is already connected")
	}

	go ue.listen(ue.wsErrorChan)
	ue.connected = true
	return ue.wsErrorChan, nil
}

// FetchStaticAssets parses index.html and fetches static assets mentioned in link/script tags.
func (ue *UserEntity) FetchStaticAssets() error {
	c := colly.NewCollector(colly.MaxDepth(1))

	c.OnHTML("link[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		c.Visit(e.Request.AbsoluteURL(link))
	})
	c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")
		c.Visit(e.Request.AbsoluteURL(link))
	})
	return c.Visit(ue.client.Url)
}

// Disconnect closes the websocket connection.
func (ue *UserEntity) Disconnect() error {
	ue.client.HttpClient.CloseIdleConnections()
	if !ue.connected {
		return errors.New("user is not connected")
	}
	// We exit the listener loop first, and then close the connection.
	// Otherwise, it tries to reconnect first, and then
	// exits, which causes unnecessary delay.
	close(ue.wsClosing)

	<-ue.wsClosed

	close(ue.wsErrorChan)
	ue.connected = false
	return nil
}

// Events returns the WebSocket event chan for the controller
// to listen and react to events.
func (ue *UserEntity) Events() <-chan *model.WebSocketEvent {
	return ue.wsEventChan
}

// Cleanup is a one time method used to close any open resources
// that the user might have kept open throughout its lifetime.
// After calling cleanup, the user might not be used any more.
// This is different from the Connect/Disconnect methods which
// can be called multiple times.
func (ue *UserEntity) Cleanup() {
	close(ue.wsEventChan)
	close(ue.wsTyping)
}

func (ue *UserEntity) IsSysAdmin() (bool, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return false, err
	}

	return user.IsInRole(model.SYSTEM_ADMIN_ROLE_ID), nil
}

func (ue *UserEntity) IsTeamAdmin() (bool, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return false, err
	}

	return user.IsInRole(model.TEAM_ADMIN_ROLE_ID), nil
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
