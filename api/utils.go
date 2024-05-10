package api

import (
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
)

// user represents a user in the users.txt file
// The format of each line in the file is:
// (authentication_type:)?email password
func parseUserFromLine(u user) (username, email, password, authenticationType string) {
	username = u.username
	email = u.email
	password = u.password
	authenticationType = userentity.AuthenticationTypeMattermost

	// Check if the user has a custom authentication type. Custom authentication types are
	// specified by prepending the username with the authentication type followed by a colon.
	// Example: "openid:user1@test.mattermost.com user1password"
	if usernameParts := strings.Split(username, ":"); len(usernameParts) > 1 {
		authenticationType = usernameParts[0]
		username = usernameParts[1]

		// Fix the email as well
		if emailParts := strings.Split(email, ":"); len(emailParts) > 1 {
			email = emailParts[1]
		}
	}

	return username, email, password, authenticationType
}
