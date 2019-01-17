package spub

import (
	"sync"

	"github.com/xiaoziqingren/spub/subscription"
	"github.com/xiaoziqingren/spub/topic"
)

type MessageCenter struct {
	mu     sync.Mutex
	topics map[string]*topic.Topic
}

var mc = &MessageCenter{topics: make(map[string]*topic.Topic)}

// NewTopic
func NewTopic(name string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if _, ok := mc.topics[name]; ok {
		return topic.ErrUnusableTopic
	}
	mc.topics[name] = new(topic.Topic)
	return nil
}

// Subscribe
func Subscribe(name string, channel interface{}) (subscription.Subscription, error) {
	t, ok := mc.topics[name]
	if !ok {
		return nil, topic.ErrHaveNoTopic
	}
	return t.Subscribe(channel), nil
}

// Publish
func Publish(name string, value interface{}) error {
	t, ok := mc.topics[name]
	if !ok {
		return topic.ErrHaveNoTopic
	}
	go t.Publish(value)
	return nil
}
