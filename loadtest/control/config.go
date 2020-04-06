// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package control

// Config is an abstraction over Config structures provided by
// UserController implementations.
type Config interface {
	// IsValid reports whether the Config is valid or not.
	// Returns an error if the validation fails.
	IsValid() error
}
