// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
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

// SetRandomSeed sets the global random seed and returns it's value.
func SetRandomSeed() int64 {
	s := os.Getenv("MM_LOADTEST_SEED")
	var seed int64
	if s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			panic(fmt.Sprintf("could not convert %q to a numeric value", s))
		}
		seed = int64(v)
	} else {
		seed = time.Now().Unix()
	}
	rand.Seed(seed)
	return seed
}
