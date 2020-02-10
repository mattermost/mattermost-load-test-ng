// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package control

// UserController defines the behavior of a single user in a load test.
// It contains a very simple interface to just start/stop the actions
// performed by a user.
type UserController interface {
	// Run starts the controller to begin performing the
	// user actions.
	Run()
	// SetRate determines the relative speed in which user actions are performed
	// one after the other. A rate of 1.0 will run the actions in their usual
	// speed. A rate of 2.0 will slow down the actions by a factor of 2.
	SetRate(rate float64) error
	// Stop stops the controller.
	Stop()
}
