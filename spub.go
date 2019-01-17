package spub

import (
	"github.com/xiaoziqingren/spub/subscription"
	"github.com/xiaoziqingren/spub/topic"
)

// NewTopic
func NewTopic(name string) error {
	return topic.NewTopic(name)
}

// Subscribe
func Subscribe(name string, channel chan interface{}) (subscription.Subscription, error) {
	return topic.Manager.Subscribe(name, channel)
}

// Publish
func Publish(name string, value interface{}) error {
	return topic.Manager.Publish(name, value)
}
