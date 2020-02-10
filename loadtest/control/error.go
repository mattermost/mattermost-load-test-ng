// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package control

// ControlError is custom error type used by a UserController.
type ControlError struct {
	// Err contains the error encountered while performing the action.
	Err error
	// Origin contains information about where the error originated in the
	// controller.
	Origin string
}

func (e *ControlError) Error() string {
	return e.Err.Error()
}
