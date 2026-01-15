// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"errors"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/performance"

	"github.com/gocolly/colly/v2"
	"github.com/mattermost/mattermost/server/public/model"
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
	dataChan    chan any
	connected   bool
	config      Config
	metrics     *performance.UserEntityMetrics
	wsConnID    string
	wsServerSeq int64
}

const (
	AuthenticationTypeMattermost = "mattermost"
	AuthenticationTypeOpenID     = "openid"
	AuthenticationTypeSAML       = "saml"
)

var stripIDsRE = regexp.MustCompile(`\b\w{26}\b`)

// Config holds necessary information required by a UserEntity.
type Config struct {
	// The URL of the Mattermost web server.
	ServerURL string
	// The URL of the mattermost WebSocket server.
	WebSocketURL string
	// The type of authentication to be used by the entity.
	AuthenticationType string
	// The username to be used by the entity.
	Username string
	// The email to be used by the entity.
	Email string
	// The password to be used by the entity.
	Password string
}

// Setup contains data used to create a new instance of UserEntity.
type Setup struct {
	// The store to be used to save user's data.
	Store store.MutableUserStore
	// The transport to be used to execute API calls.
	Transport http.RoundTripper
	// An optional object used to collect metrics.
	Metrics *performance.UserEntityMetrics
	// The HTTP client timeout to use.
	ClientTimeout time.Duration
}

type userTypingMsg struct {
	channelId string
	parentId  string
}

type channelPresenceMsg struct {
	channelId string
}

type threadPresenceMsg struct {
	channelId  string
	threadView bool
}

type teamPresenceMsg struct {
	teamId string
}

type postedAckMsg struct {
	postId     string
	status     string
	reason     string
	postedData string
}

type ueTransport struct {
	transport http.RoundTripper
	ue        *UserEntity
}

// RoundTrip implements the RoundTripper interface for ueTransport.
// This is used to collect metrics regarding the timing of HTTP calls.
func (t *ueTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()
	resp, err := t.transport.RoundTrip(req)
	t.ue.observeHTTPRequestTimes(time.Since(startTime).Seconds())
	if os.IsTimeout(err) {
		t.ue.incHTTPTimeouts(stripIDs(req.URL.Path), req.Method)
	}
	if resp != nil && resp.StatusCode >= 400 {
		t.ue.incHTTPErrors(stripIDs(req.URL.Path), req.Method, resp.StatusCode)
	}
	return resp, err
}

func (ue *UserEntity) Client() *model.Client4 {
	return ue.client
}

// Store returns the underlying store of the user.
func (ue *UserEntity) Store() store.UserStore {
	return ue.store
}

// New returns a new instance of a UserEntity.
func New(setup Setup, config Config) *UserEntity {
	var ue UserEntity
	ue.config = config
	ue.store = setup.Store
	ue.metrics = setup.Metrics
	ue.client = model.NewAPIv4Client(config.ServerURL)

	if setup.Transport == nil {
		setup.Transport = http.DefaultTransport
	}
	if setup.Metrics != nil {
		setup.Transport = &ueTransport{
			transport: setup.Transport,
			ue:        &ue,
		}
	}
	ue.client.HTTPClient = &http.Client{
		Transport: setup.Transport,
		Timeout:   setup.ClientTimeout,
	}

	err := ue.store.SetUser(&model.User{
		Username: config.Username,
		Email:    config.Email,
		Password: config.Password,
		Id:       ue.store.Id(),
	})
	if err != nil {
		return nil
	}
	ue.store.SetPerformanceReport(&model.PerformanceReport{
		Version: "0.1.0",
		Labels: map[string]string{
			"platform": "other",
			"agent":    "other",
		},
		ClientID: model.NewId(),
		Start:    float64(time.Now().UnixMilli()),
	})

	return &ue
}

// Connect creates a WebSocket connection to the server and starts listening for messages.
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

	ue.wsEventChan = make(chan *model.WebSocketEvent)
	ue.dataChan = make(chan any, 10)
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
	return c.Visit(ue.client.URL)
}

// Disconnect closes the WebSocket connection.
func (ue *UserEntity) Disconnect() error {
	ue.client.HTTPClient.CloseIdleConnections()
	if !ue.connected {
		return errors.New("user is not connected")
	}
	// We exit the listener loop first, and then close the connection.
	// Otherwise, it tries to reconnect first, and then
	// exits, which causes unnecessary delay.
	close(ue.wsClosing)

	<-ue.wsClosed

	close(ue.wsEventChan)
	close(ue.dataChan)
	close(ue.wsErrorChan)
	ue.connected = false
	return nil
}

// Events returns the WebSocket event chan for the controller
// to listen and react to events.
func (ue *UserEntity) Events() <-chan *model.WebSocketEvent {
	return ue.wsEventChan
}

// IsSysAdmin returns whether the user is a system admin or not.
func (ue *UserEntity) IsSysAdmin() (bool, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return false, err
	}

	return user.IsInRole(model.SystemAdminRoleId), nil
}

// IsTeamAdmin returns whether the user is a team admin or not.
func (ue *UserEntity) IsTeamAdmin() (bool, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return false, err
	}

	return user.IsInRole(model.TeamAdminRoleId), nil
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

func stripIDs(path string) string {
	return stripIDsRE.ReplaceAllString(path, "$$ID")
}
