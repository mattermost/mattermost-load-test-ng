// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

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
	StateStopped State = iota
	StateStarting
	StateRunning
	StateStopping
)

// ErrUnknownState is returned when an unknown state variable is encoded/decoded.
var ErrUnknownState = errors.New("unknown state")

// UnmarshalJSON constructs the state from a JSON string.
func (s *State) UnmarshalJSON(b []byte) error {
	var res string
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}

	switch strings.ToLower(res) {
	default:
		return ErrUnknownState
	case "stopped":
		*s = StateStopped
	case "starting":
		*s = StateStarting
	case "running":
		*s = StateRunning
	case "stopping":
		*s = StateStopping
	}

	return nil
}

// MarshalJSON returns a JSON representation from a State variable.
func (s State) MarshalJSON() ([]byte, error) {
	var res string
	switch s {
	default:
		return nil, ErrUnknownState
	case StateStopped:
		res = "stopped"
	case StateStarting:
		res = "starting"
	case StateRunning:
		res = "running"
	case StateStopping:
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
