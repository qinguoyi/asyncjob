package event

import (
	"context"
	"sync"
	"time"
)

var (
	once          sync.Once
	eventsHandler *EventsHandler
)

type EventsHandler struct {
	mux sync.RWMutex
	// ctx 信号
	// t 超时
	// s 终止
	handlers map[string]func(i interface{}, c context.Context, t <-chan time.Time, s chan struct{}) error
}

func NewEventsHandler() *EventsHandler {
	once.Do(func() {
		eventsHandler = &EventsHandler{
			mux:      sync.RWMutex{},
			handlers: map[string]func(i interface{}, c context.Context, t <-chan time.Time, s chan struct{}) error{},
		}
	})
	return eventsHandler
}

// RegHandler 注册handler
func (e *EventsHandler) RegHandler(t string, handler func(i interface{}, c context.Context, t <-chan time.Time, s chan struct{}) error) {
	e.mux.Lock()
	defer e.mux.Unlock()
	_, ok := e.handlers[t]
	if !ok {
		e.handlers[t] = handler
	}
}

// GetHandler 获取handler
func (e *EventsHandler) GetHandler(t string) func(i interface{}, c context.Context, t <-chan time.Time, s chan struct{}) error {
	e.mux.RLock()
	defer e.mux.RUnlock()
	handler, ok := e.handlers[t]
	if !ok {
		return nil
	} else {
		return handler
	}
}
