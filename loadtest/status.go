// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// State determines which state a loadtester is in.
type State int

// Different possible states of a loadtester.
const (
	Stopped State = iota
	Starting
	Running
	Stopping
)

// ErrInvalidState is returned when an unknown state variable is encoded/decoded.
var ErrInvalidState = errors.New("unknown state")

// UnmarshalJSON constructs the state from a JSON string.
func (s *State) UnmarshalJSON(b []byte) error {
	var res string
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}

	switch strings.ToLower(res) {
	default:
		return ErrInvalidState
	case "stopped":
		*s = Stopped
	case "starting":
		*s = Starting
	case "running":
		*s = Running
	case "stopping":
		*s = Stopping
	}

	return nil
}

// MarshalJSON returns a JSON representation from a State variable.
func (s State) MarshalJSON() ([]byte, error) {
	var res string
	switch s {
	default:
		return nil, ErrInvalidState
	case Stopped:
		res = "stopped"
	case Starting:
		res = "starting"
	case Running:
		res = "running"
	case Stopping:
		res = "stopping"
	}

	return json.Marshal(res)
}

// Status contains various information about the load test.
type Status struct {
	State           State     // State of the load test.
	NumUsers        int       // Number of active users.
	NumUsersAdded   int       // Number of users added since the start of the test.
	NumUsersRemoved int       // Number of users removed since the start of the test.
	NumErrors       int64     // Number of errors that have occurred.
	StartTime       time.Time // Time when the load test was started. This only logs the time when the load test was first started, and does not get reset if it was subsequently restarted.
}
