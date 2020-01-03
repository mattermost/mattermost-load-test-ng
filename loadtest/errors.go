package loadtest

import (
	"errors"
)

var (
	ErrNotRunning      = errors.New("LoadTester is not running")
	ErrAlreadyRunning  = errors.New("LoadTester is already running")
	ErrNoUsersLeft     = errors.New("No active users left")
	ErrMaxUsersReached = errors.New("Max active users limit reached")
	ErrStopping        = errors.New("LoadTester is stopping")
)
