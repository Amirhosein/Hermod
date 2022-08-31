package broker

import (
	"context"
	"sync"
	"therealbroker/internal/app/model"
	"therealbroker/internal/app/repository"
	"therealbroker/internal/pkg/broker"
	"time"

	"go.opentelemetry.io/otel"
)

type Module struct {
	subscribers   map[string]model.Subject
	messages      map[int]broker.Message
	db            repository.Database
	IsClosed      bool
	ListenersLock sync.RWMutex
}

func NewModule() broker.Broker {
	db, err := repository.GetPostgre()
	// db, err := repository.GetCassandra()
	if err != nil {
		panic(err)
	}

	return &Module{
		subscribers:   make(map[string]model.Subject),
		messages:      make(map[int]broker.Message),
		db:            db,
		IsClosed:      false,
		ListenersLock: sync.RWMutex{},
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
	select {
	case <-ctx.Done():
		return -1, ctx.Err()
	default:
		_, subSpan := otel.Tracer("Server").Start(ctx, "Module.putChannel")
		for _, listener := range m.subscribers[subject].Subscribers {
			if cap(listener.Channel) != len(listener.Channel) {
				listener.Channel <- msg
			}
		}
		subSpan.End()

		_, saveSpan := otel.Tracer("Server").Start(ctx, "Module.saveDB")
		msg.Id = m.db.SaveMessage(msg, subject)
		// msg.Id = len(m.messages)

		// m.ListenersLock.Lock()
		// m.messages[msg.Id] = msg
		// m.ListenersLock.Unlock()
		saveSpan.End()

		if msg.Expiration != 0 {
			go func(msg *broker.Message) {
				ticker := time.NewTicker(msg.Expiration)
				defer ticker.Stop()

				<-ticker.C
				m.ListenersLock.Lock()
				// delete(m.messages, msg.Id)
				m.db.DeleteMessage(msg.Id, subject)
				m.ListenersLock.Unlock()
			}(&msg)
		}

		return msg.Id, nil
	}
}

func (m *Module) Subscribe(ctx context.Context, subject string) (<-chan broker.Message, error) {
	if m.IsClosed {
		return nil, broker.ErrUnavailable
	}

	select {
	case <-ctx.Done():
		return nil, broker.ErrInvalidID
	default:
		newChannel := make(chan broker.Message, 100)

		m.ListenersLock.Lock()
		m.subscribers[subject] = model.Subject{
			Subscribers: append(m.subscribers[subject].Subscribers, model.Subscriber{Channel: newChannel}),
			Name:        subject,
		}
		m.ListenersLock.Unlock()

		return newChannel, nil
	}
}

func (m *Module) Fetch(ctx context.Context, subject string, id int) (broker.Message, error) {
	if m.IsClosed {
		return broker.Message{}, broker.ErrUnavailable
	}

	select {
	case <-ctx.Done():
		return broker.Message{}, broker.ErrInvalidID
	default:
		// msg, ok := m.messages[id]
		// if ok {
		// 	return msg, nil
		// }
		msg, err := m.db.FetchMessage(id)
		if err != nil {
			return broker.Message{}, broker.ErrExpiredID
		}

		return msg, nil
	}
}
