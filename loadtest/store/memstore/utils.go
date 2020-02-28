// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

func restorePrivateData(old, new *model.User) {
	if old == nil {
		return
	}

	if new.Password == "" {
		new.Password = old.Password
	}

	if new.LastPasswordUpdate == 0 {
		new.LastPasswordUpdate = old.LastPasswordUpdate
	}

	if new.Email == "" {
		new.Email = old.Email
	}

	if new.FirstName == "" {
		new.FirstName = old.FirstName
	}

	if new.LastName == "" {
		new.LastName = old.LastName
	}

	if new.AuthService == "" {
		new.AuthService = old.AuthService
	}

	if new.AuthData == nil {
		new.AuthData = old.AuthData
	}

	if new.MfaSecret == "" {
		new.MfaSecret = old.MfaSecret
	}
}
