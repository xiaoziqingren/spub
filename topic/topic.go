package topic

import (
	"github.com/xiaoziqingren/spub/subscription"
	"reflect"
	"sync"
)

// Topic
type Topic struct {
	name string

	once      sync.Once        // ensures that init only runs once
	sendLock  chan struct{}    // sendLock has a one-element buffer and is empty when held.It protects sendCases.
	removeSub chan interface{} // interrupts Send
	sendCases caseList         // the active set of select cases used by Send

	// The inbox holds newly subscribed channels until they are added to sendCases.
	mu     sync.Mutex
	inbox  caseList
	etype  reflect.Type
	closed bool
}

// This is the index of the first actual subscription channel in sendCases.
// sendCases[0] is a SelectRecv case for the removeSub channel.
const firstSubSendCase = 1

func NewTopic(name string) error {
	if Manager.exist(name) {
		return errUnusableTopic
	}

	t := &Topic{name: name}
	Manager.add(t)
	return nil
}

func (t *Topic) init() {
	t.removeSub = make(chan interface{})
	t.sendLock = make(chan struct{}, 1)
	t.sendLock <- struct{}{}
	t.sendCases = caseList{{Chan: reflect.ValueOf(t.removeSub), Dir: reflect.SelectRecv}}
}

func (t *Topic) Subscribe(channel interface{}) subscription.Subscription {
	t.once.Do(t.init)

	chanval := reflect.ValueOf(channel)
	chantyp := chanval.Type()
	if chantyp.Kind() != reflect.Chan || chantyp.ChanDir()&reflect.SendDir == 0 {
		panic(errBadChannel)
	}
	sub := &subPerformer{topic: t, channel: chanval, err: make(chan error, 1)}

	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.checkType(chantyp.Elem()) {
		panic(ChanTypeError{got: chantyp, want: reflect.ChanOf(reflect.SendDir, t.etype)})
	}
	// Add the select case to the inbox.
	// The next Send will add it to f.sendCases.
	cas := reflect.SelectCase{Dir: reflect.SelectSend, Chan: chanval}
	t.inbox = append(t.inbox, cas)
	return sub
}

func (t *Topic) Unsubscribe(sub *subPerformer) {
	ch := sub.channel.Interface()
	t.mu.Lock()
	index := t.inbox.find(ch)
	if index != -1 {
		t.inbox = t.inbox.delete(index)
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	select {
	case t.removeSub <- ch:
		// Send will remove the channel from f.sendCases.
	case <-t.sendLock:
		// No Send is in progress, delete the channel now that we have the send lock.
		t.sendCases = t.sendCases.delete(t.sendCases.find(ch))
		t.sendLock <- struct{}{}
	}
}

func (t *Topic) Publish(value interface{}) (sent int) {
	rvalue := reflect.ValueOf(value)

	t.once.Do(t.init)
	<-t.sendLock

	// Add new cases from the inbox after taking the send lock.
	t.mu.Lock()
	t.sendCases = append(t.sendCases, t.inbox...)
	t.inbox = nil

	if !t.checkType(rvalue.Type()) {
		t.sendLock <- struct{}{}
		panic(ChanTypeError{got: rvalue.Type(), want: t.etype})
	}
	t.mu.Unlock()

	// Set the sent value on all channels.
	for i := firstSubSendCase; i < len(t.sendCases); i++ {
		t.sendCases[i].Send = rvalue
	}

	// Send until all channels except removeSub have been chosen. 'cases' tracks a prefix
	// of sendCases. When a send succeeds, the corresponding case moves to the end of
	// 'cases' and it shrinks by one element.
	cases := t.sendCases
	for {
		// Fast path: try sending without blocking before adding to the select set.
		// This should usually succeed if subscribers are fast enough and have free
		// buffer space.
		for i := firstSubSendCase; i < len(cases); i++ {
			if cases[i].Chan.TrySend(rvalue) {
				sent++
				cases = cases.deactivate(i)
				i--
			}
		}
		if len(cases) == firstSubSendCase {
			break
		}
		// Select on all the receivers, waiting for them to unblock.
		chosen, recv, _ := reflect.Select(cases)
		if chosen == 0 /* <-f.removeSub */ {
			index := t.sendCases.find(recv.Interface())
			t.sendCases = t.sendCases.delete(index)
			if index >= 0 && index < len(cases) {
				// Shrink 'cases' too because the removed case was still active.
				cases = t.sendCases[:len(cases)-1]
			}
		} else {
			cases = cases.deactivate(chosen)
			sent++
		}
	}

	// Forget about the sent value and hand off the send lock.
	for i := firstSubSendCase; i < len(t.sendCases); i++ {
		t.sendCases[i].Send = reflect.Value{}
	}
	t.sendLock <- struct{}{}
	return sent
}

func (t *Topic) checkType(value reflect.Type) bool {
	if t.etype == nil {
		t.etype = value
		return true
	}
	return t.etype == value
}

// subPerformer
type subPerformer struct {
	topic   *Topic
	channel reflect.Value
	errOnce sync.Once
	err     chan error
}

func (s *subPerformer) Unsubscribe() {
	s.errOnce.Do(func() {
		s.topic.Unsubscribe(s)
		close(s.err)
	})
}

func (s *subPerformer) Err() <-chan error {
	return s.err
}

type caseList []reflect.SelectCase

// find returns the index of a case containing the given channel.
func (cs caseList) find(channel interface{}) int {
	for i, cas := range cs {
		if cas.Chan.Interface() == channel {
			return i
		}
	}
	return -1
}

// delete removes the given case from cs.
func (cs caseList) delete(index int) caseList {
	return append(cs[:index], cs[index+1:]...)
}

// deactivate moves the case at index into the non-accessible portion of the cs slice.
func (cs caseList) deactivate(index int) caseList {
	last := len(cs) - 1
	cs[index], cs[last] = cs[last], cs[index]
	return cs[:last]
}
