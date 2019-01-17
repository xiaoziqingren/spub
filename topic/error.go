package topic

import (
	"errors"
	"reflect"
)

var (
	errBadChannel    = errors.New(" Subscribe argument does not have sendable channel type ")
	errUnusableTopic = errors.New(" Unusable topic name ")
	errHaveNoTopic   = errors.New(" Have no topic with this name ")
)

type ChanTypeError struct {
	got, want reflect.Type
}

func (c *ChanTypeError) Error() string {
	return "wrong type: " + " got " + c.got.String() + ", want " + c.want.String()
}
