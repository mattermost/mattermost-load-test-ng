// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package control

type Config interface {
	// IsValid reports whether the Config is valid or not.
	// Returns an error if the validation fails.
	IsValid() error
}
