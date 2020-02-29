// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package control

// UserError is a custom error type used to report user errors.
type UserError struct {
	// Err contains the error encountered while performing the action.
	Err error
	// Origin contains information about where the error originated.
	Origin string
}

func (e *UserError) Error() string {
	return e.Origin + " " + e.Err.Error()
}

// NewUserError returns a new UserError object with the given error
// including location information.
func NewUserError(err error) *UserError {
	origin := getErrOrigin()
	return &UserError{
		Err:    err,
		Origin: origin,
	}
}
