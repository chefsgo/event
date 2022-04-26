package event

import (
	. "github.com/chefsgo/base"
)

// Publish
func Publish(name string, values ...Map) error {
	return module.Publish(name, values...)
}
