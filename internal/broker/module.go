package broker

import (
	"context"
	"sync"
	"therealbroker/pkg/broker"
	"time"
)

type Module struct {
	subscribers   map[string][]chan broker.Message
	messages      []broker.Message
	IsClosed      bool
	ListenersLock sync.Mutex
}

func NewModule() broker.Broker {
	return &Module{
		subscribers:   make(map[string][]chan broker.Message),
		IsClosed:      false,
		ListenersLock: sync.Mutex{},
	}
}

func (m *Module) Close() error {
	if m.IsClosed {
		return broker.ErrUnavailable
	}

	m.IsClosed = true
	return nil
}

func (m *Module) Publish(ctx context.Context, subject string, msg broker.Message) (int, error) {
	if m.IsClosed {
		return -1, broker.ErrUnavailable
	}

	for _, listener := range m.subscribers[subject] {
		if cap(listener) != len(listener) {

			listener <- msg
		}
	}

	msg.Id = len(m.messages)
	// msg.IsExpired = false
	m.messages = append(m.messages, msg)

	if msg.Expiration != 0 {
		go func(msg *broker.Message) {
			time.Sleep(msg.Expiration)
			m.ListenersLock.Lock()
			defer m.ListenersLock.Unlock()
			for i, msg := range m.messages {
				if msg.Id == i {
					m.messages = append(m.messages[:i], m.messages[i+1:]...)
					break
				}
			}
			// msg.IsExpired = true
		}(&msg)
	}

	return msg.Id, nil
}

func (m *Module) Subscribe(ctx context.Context, subject string) (<-chan broker.Message, error) {
	if m.IsClosed {
		return nil, broker.ErrUnavailable
	}

	select {
	case <-ctx.Done():
		return nil, broker.ErrExpiredID
	default:
		newChannel := make(chan broker.Message, 100)
		m.ListenersLock.Lock()
		m.subscribers[subject] = append(m.subscribers[subject], newChannel)
		m.ListenersLock.Unlock()

		return newChannel, nil
	}
}

func (m *Module) Fetch(ctx context.Context, subject string, id int) (broker.Message, error) {
	if m.IsClosed {
		return broker.Message{}, broker.ErrUnavailable
	}

	for _, msg := range m.messages {
		if msg.Id == id {
			return msg, nil
		}
	}

	return broker.Message{}, broker.ErrExpiredID
}
