package event

import "errors"

const (
	NAME = "event"
)

var (
	errInvalidConnection = errors.New("Invalid event connection.")
)
