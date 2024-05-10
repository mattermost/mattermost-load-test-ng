package api

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/stretchr/testify/assert"
)

type expectedParseUserFromLine struct {
	username           string
	email              string
	password           string
	authenticationType string
}

func TestParseUserFromLine(t *testing.T) {
	testCases := []struct {
		name     string
		input    user
		expected expectedParseUserFromLine
	}{
		{
			name: "no custom authentication type",
			input: user{
				username: "user1",
				email:    "user1@sample.mattermost.com",
				password: "user1password",
			},
			expected: expectedParseUserFromLine{
				username:           "user1",
				email:              "user1@sample.mattermost.com",
				password:           "user1password",
				authenticationType: userentity.AuthenticationTypeMattermost,
			},
		},
		{
			name: "custom authentication type",
			input: user{
				username: "openid:user1",
				email:    "openid:user1@sample.mattermost.com",
				password: "user1password",
			},
			expected: expectedParseUserFromLine{
				username:           "user1",
				email:              "user1@sample.mattermost.com",
				password:           "user1password",
				authenticationType: userentity.AuthenticationTypeOpenID,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			username, email, password, authenticationType := parseUserFromLine(tc.input)
			assert.Equal(t, tc.expected.username, username)
			assert.Equal(t, tc.expected.email, email)
			assert.Equal(t, tc.expected.password, password)
			assert.Equal(t, tc.expected.authenticationType, authenticationType)
		})
	}
}
