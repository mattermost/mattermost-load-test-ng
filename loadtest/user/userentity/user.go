// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"
	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
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
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
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

// DownloadFile will download a url into a buffer
func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("File not found: " + url)
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func extractLinks(data []byte) []string {
	tokenizer := html.NewTokenizer(strings.NewReader(string(data)))
	links := []string{}
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			// End of the document, we're done
			return links
		case html.StartTagToken:
			tag, _ := tokenizer.TagName()
			nodeType := ""
			nodeValue := ""
			if string(tag) != "script" {
				continue
			}
			for {
				key, val, more := tokenizer.TagAttr()
				if string(key) == "src" {
					nodeValue = string(val)
				}
				if string(key) == "type" {
					nodeType = string(val)
				}
				if !more {
					break
				}
			}
			if nodeType == "text/javascript" {
				links = append(links, nodeValue)
			}
		}
	}
}

// FetchStaticAssets parses index.html and fetches static assets mentioned in it
func (ue *UserEntity) FetchStaticAssets() error {
	indexHTML, err := downloadFile(ue.client.Url)
	if err != nil {
		return err
	}

	var g errgroup.Group

	for _, url := range extractLinks(indexHTML) {
		// Launch a goroutine to fetch the URL.
		url := url // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			_, err := downloadFile(ue.client.Url + "/" + url)
			return err
		})
	}
	// Wait for all HTTP fetches to complete.
	return g.Wait()
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
}

func (ue *UserEntity) IsSysAdmin() (bool, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return false, err
	}

	return user.IsInRole(model.SYSTEM_ADMIN_ROLE_ID), nil
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
