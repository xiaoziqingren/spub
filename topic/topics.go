package topic

import (
	"sync"

	"github.com/xiaoziqingren/spub/subscription"
)

type topics struct {
	mu        sync.RWMutex
	topicList map[string]*Topic
}

// topics
var Manager *topics

func init() {
	Manager = &topics{
		topicList: make(map[string]*Topic),
	}
}

func (t *topics) add(tp *Topic) {
	t.mu.Lock()
	t.topicList[tp.name] = tp
	t.mu.Unlock()
}

func (t *topics) remove(name string) {
	t.mu.Lock()
	delete(t.topicList, name)
	t.mu.Unlock()
}

func (t *topics) exist(name string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	_, ok := t.topicList[name]
	return ok
}

func (t *topics) Subscribe(name string, channel interface{}) (subscription.Subscription, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if val, ok := t.topicList[name]; ok {
		return val.Subscribe(channel), nil
	}
	return nil, errHaveNoTopic
}

func (t *topics) Publish(name string, value interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if val, ok := t.topicList[name]; ok {
		val.Publish(val)
		return nil
	}
	return errHaveNoTopic
}
