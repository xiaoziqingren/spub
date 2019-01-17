package topic

import (
	"reflect"
	"sync"
)

// Topic
type Topic struct {
	name string

	mu   sync.RWMutex
	once sync.Once

	sendBlocker chan struct{}
	subscriber  caseList

	eType reflect.Type
}

func NewTopic(name string) error {
	if Manager.exist(name) {
		return errUnusableTopic
	}

	t := &Topic{name: name}
	t.once.Do(t.init)
	Manager.add(t)
	return nil
}

func (t *Topic) init() {
	t.sendBlocker = make(chan struct{}, 1)
	t.sendBlocker <- struct{}{}
}

func (t *Topic) Subscribe(receiver interface{}) {
	// check chan type
	chanVal := reflect.ValueOf(receiver)
	chanTyp := chanVal.Type()
	if chanTyp.Kind() != reflect.Chan || chanTyp.ChanDir()&reflect.SendDir == 0 {
		panic(errBadChannel)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	// check new chan type if equal to topic type
	if !t.checkType(chanTyp.Elem()) {
		panic(ChanTypeError{got: chanTyp.Elem(), want: reflect.ChanOf(reflect.SendDir, t.eType)})
	}

	t.subscriber = append(t.subscriber, reflect.SelectCase{Dir: reflect.SelectSend, Chan: chanVal})
}

func (t *Topic) UnSubscribe(receiver reflect.Value) {
	ch := receiver.Interface()
	t.mu.Lock()
	index := t.subscriber.find(ch)
	if index != -1 {
		t.subscriber = t.subscriber.delete(index)
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()
}

func (t *Topic) Publish(value interface{}) (sent int) {
	rvalue := reflect.ValueOf(value)

	<-t.sendBlocker

	// Add new cases from the inbox after taking the send lock.
	t.mu.Lock()
	if !t.checkType(rvalue.Type()) {
		t.sendBlocker <- struct{}{}
		panic(ChanTypeError{got: rvalue.Type(), want: t.eType})
	}
	t.mu.Unlock()

	// Set the sent value on all channels.
	for i := 0; i < len(t.subscriber); i++ {
		t.subscriber[i].Send = rvalue
	}

	// Send until all channels except removeSub have been chosen. 'cases' tracks a prefix
	// of sendCases. When a send succeeds, the corresponding case moves to the end of
	// 'cases' and it shrinks by one element.
	cases := t.subscriber
	for {
		// Fast path: try sending without blocking before adding to the select set.
		// This should usually succeed if subscribers are fast enough and have free
		// buffer space.
		for i := 0; i < len(cases); i++ {
			if cases[i].Chan.TrySend(rvalue) {
				sent++
				cases = cases.deactivate(i)
				i--
			}
		}
		if len(cases) == 0 {
			break
		}
		// Select on all the receivers, waiting for them to unblock.
		chosen, recv, _ := reflect.Select(cases)
		if chosen == 0 /* <-f.removeSub */ {
			index := t.subscriber.find(recv.Interface())
			t.subscriber = t.subscriber.delete(index)
			if index >= 0 && index < len(cases) {
				// Shrink 'cases' too because the removed case was still active.
				cases = t.subscriber[:len(cases)-1]
			}
		} else {
			cases = cases.deactivate(chosen)
			sent++
		}
	}

	// Forget about the sent value and hand off the send lock.
	for i := 0; i < len(cases); i++ {
		t.subscriber[i].Send = reflect.Value{}
	}
	t.sendBlocker <- struct{}{}
	return sent
}

func (t *Topic) checkType(value reflect.Type) bool {
	if t.eType == nil {
		t.eType = value
		return true
	}
	return t.eType == value
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
