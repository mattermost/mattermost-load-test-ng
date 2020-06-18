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
)

// State related errors.
var (
	ErrNotRunning = errors.New("coordinator is not running")
	ErrNotStopped = errors.New("coordinator has not stopped")
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
	case Running:
		res = "running"
	}

	return json.Marshal(res)
}

// Status contains various information about Coordinator.
type Status struct {
	State       State     // State of Coordinator.
	StartTime   time.Time // Time when Coordinator was started.
	ActiveUsers int       // Total number of currently active users across the load-test agents cluster.
	NumErrors   int64     // Total number of errors received from the load-test agents cluster.
}
