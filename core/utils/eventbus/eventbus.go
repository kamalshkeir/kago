package eventbus

import (
	"sync"
)


type DataEvent[T map[string]string] struct {
	Data  T
	Topic string
}

// DataChannel is a channel which can accept an DataEvent
type DataChannel[T map[string]string] chan DataEvent[T]


// EventBus stores the information about subscribers interested for a particular topic
type EventBus struct {
	subscribers map[string][]DataChannel[map[string]string]
	rm          sync.RWMutex
}

var eb = new()

func new() *EventBus {
	return &EventBus{
		subscribers: map[string][]DataChannel[map[string]string]{},
	}
}

func Publish(topic string, data map[string]string) {
	eb.rm.RLock()
	if chans, found := eb.subscribers[topic]; found {
		// create copy of channels to avoid copy reference
		channels := append([]DataChannel[map[string]string]{}, chans...)
		go func(data DataEvent[map[string]string], dataChannels []DataChannel[map[string]string]) {
			for _, ch := range dataChannels {
				ch <- data
			}
		}(DataEvent[map[string]string]{Data: data, Topic: topic}, channels)
	}
	eb.rm.RUnlock()
}

func Subscribe(topic string,fn func (data map[string]string)) {
	ch := make(chan DataEvent[map[string]string])
	eb.rm.Lock()
	if subs, found := eb.subscribers[topic]; found {
		eb.subscribers[topic] = append(subs, ch)
	} else {
		eb.subscribers[topic] = append([]DataChannel[map[string]string]{}, ch)
	}
	eb.rm.Unlock()

	go func() {
		for v := range ch {
			fn(v.Data)
		}
	}()
}

