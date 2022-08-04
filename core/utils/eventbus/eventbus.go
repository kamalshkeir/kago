package eventbus

import (
	"fmt"
	"sync"
)

var mTopicBus = map[string]any{}

type Bus[T any] struct {
	Topic string
	Subs  map[string][]chan T
	mu    sync.RWMutex
}

func Subscribe[T any](topic string, fn func(data T)) {
	var b *Bus[T]
	if topicbus, ok := mTopicBus[topic]; ok {
		if bb, ok := topicbus.(*Bus[T]); ok {
			b = bb
		}
	} else {
		b = &Bus[T]{
			Topic: topic,
			Subs:  make(map[string][]chan T),
		}
		mTopicBus[topic] = b
	}
	ch := make(chan T)
	b.mu.Lock()
	if subs, found := b.Subs[topic]; found {
		b.Subs[topic] = append(subs, ch)
	} else {
		b.Subs[topic] = append([]chan T{}, ch)
	}
	b.mu.Unlock()

	go func() {
		for v := range ch {
			fn(v)
		}
	}()
}

func Publish[T any](topic string, data T) {
	var b *Bus[T]
	if topicbus, ok := mTopicBus[topic]; ok {
		if bb, ok := topicbus.(*Bus[T]); ok {
			b = bb
		} else {
			fmt.Printf("eventbus SendTo topic doesn't match data type of bus: want %T got %T\n", topicbus, bb)
			return
		}
	} else {
		b = &Bus[T]{
			Topic: topic,
			Subs:  make(map[string][]chan T),
		}
		mTopicBus[topic] = b
	}
	b.mu.RLock()
	if chans, found := b.Subs[topic]; found {
		// create copy of channels to avoid copy reference
		channels := append([]chan T{}, chans...)
		go func(data T, dataChannels []chan T) {
			for _, ch := range dataChannels {
				ch <- data
			}
		}(data, channels)
	}
	b.mu.RUnlock()
}
