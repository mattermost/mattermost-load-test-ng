package loadtest

import (
	"errors"
)

var (
	ErrNotRunning      = errors.New("LoadTester is not running")
	ErrNotStopped      = errors.New("LoadTester has not stopped")
	ErrNoUsersLeft     = errors.New("No active users left")
	ErrMaxUsersReached = errors.New("Max active users limit reached")
)
