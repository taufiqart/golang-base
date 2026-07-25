package event

import (
	"sync"
)

// Payload represents the data passed with an event.
type Payload interface{}

// Listener is a function that handles an event.
type Listener func(payload Payload)

// Dispatcher manages event registration and dispatching.
type Dispatcher struct {
	mu        sync.RWMutex
	listeners map[string][]Listener
}

// NewDispatcher creates a new Event Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		listeners: make(map[string][]Listener),
	}
}

// Subscribe adds a listener to an event.
func (d *Dispatcher) Subscribe(eventName string, listener Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners[eventName] = append(d.listeners[eventName], listener)
}

// Dispatch triggers all listeners associated with an event.
// This runs synchronously. For async execution, listeners should spawn their own goroutines.
func (d *Dispatcher) Dispatch(eventName string, payload Payload) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if handlers, ok := d.listeners[eventName]; ok {
		for _, handler := range handlers {
			handler(payload)
		}
	}
}

// Global default dispatcher
var DefaultDispatcher = NewDispatcher()

// Subscribe registers a listener to the global dispatcher.
func Subscribe(eventName string, listener Listener) {
	DefaultDispatcher.Subscribe(eventName, listener)
}

// Dispatch triggers an event on the global dispatcher.
func Dispatch(eventName string, payload Payload) {
	DefaultDispatcher.Dispatch(eventName, payload)
}
