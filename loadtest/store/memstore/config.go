// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
)

// Config holds information used to create a new MemStore.
type Config struct {
	MaxStoredPosts          int // The maximum number of posts to be stored.
	MaxStoredUsers          int // The maximum number of users to be stored.
	MaxStoredChannelMembers int // The maximum number of channel members to be stored.
	MaxStoredStatuses       int // The maximum number of statuses to be stored.
}

// IsValid checks whether a Config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	if c.MaxStoredPosts <= 0 {
		return errors.New("MaxStoredPosts should be > 0")
	}

	if c.MaxStoredUsers <= 0 {
		return errors.New("MaxStoredUsers should be > 0")
	}

	if c.MaxStoredChannelMembers <= 0 {
		return errors.New("MaxStoredChannelMembers should be > 0")
	}

	if c.MaxStoredStatuses <= 0 {
		return errors.New("MaxStoredStatuses should be > 0")
	}

	return nil
}

// SetDefaults sets default values to the config.
func (c *Config) SetDefaults() {
	c.MaxStoredPosts = 100
	c.MaxStoredUsers = 100
	c.MaxStoredChannelMembers = 100
	c.MaxStoredStatuses = 100
}
