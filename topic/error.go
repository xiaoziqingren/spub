package topic

import (
	"errors"
	"reflect"
)

var (
	ErrBadChannel    = errors.New(" Subscribe argument does not have sendable channel type ")
	ErrUnusableTopic = errors.New(" Unusable topic name ")
	ErrHaveNoTopic   = errors.New(" Have no topic with this name ")
)

type ChanTypeError struct {
	got, want reflect.Type
}

func (c *ChanTypeError) Error() string {
	return "wrong type: " + " got " + c.got.String() + ", want " + c.want.String()
}
