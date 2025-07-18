// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// State determines which state a Coordinator is in.
type State int

// Different possible states of a Coordinator.
const (
	Stopped State = iota
	Running
	Done
)

// State related errors.
var (
	ErrNotRunning  = errors.New("coordinator is not running")
	ErrNotStopped  = errors.New("coordinator has not stopped")
	ErrAlreadyDone = errors.New("coordinator is already done")
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
	case "running":
		*s = Running
	case "done":
		*s = Done
	}

	return nil
}

// MarshalJSON returns a JSON representation from a State variable.
func (s State) MarshalJSON() ([]byte, error) {
	val, err := s.stateToString()
	if err != nil {
		return nil, err
	}
	return json.Marshal(val)
}

func (s State) stateToString() (string, error) {
	var res string
	switch s {
	default:
		return "", ErrInvalidState
	case Stopped:
		res = "stopped"
	case Running:
		res = "running"
	case Done:
		res = "done"
	}
	return res, nil
}

func (s State) String() string {
	res, _ := s.stateToString()
	return res
}

// Status contains various information about Coordinator.
type Status struct {
	State              State     // State of Coordinator.
	StartTime          time.Time // Time when Coordinator has started.
	StopTime           time.Time // Time when Coordinator has stopped.
	ActiveUsers        int       // Total number of currently active users across the load-test agents cluster.
	NumErrors          int64     // Total number of errors received from the load-test agents cluster.
	SupportedUsers     int       // Number of supported users.
	ActiveBrowserUsers int       // Total browser users.
	NumBrowserErrors   int64     // Total browser errors.
}
