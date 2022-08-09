package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"therealbroker/api/proto"
	"therealbroker/pkg/broker"
	"time"

	"go.opentelemetry.io/otel"
)

type Server struct {
	proto.UnimplementedBrokerServer
	BrokerInstance broker.Broker

	LastPublishLock *sync.Mutex
	LastTopicLock   *sync.Mutex
	LastPublishId   int
}

func (s *Server) Publish(globalContext context.Context, request *proto.PublishRequest) (*proto.PublishResponse, error) {
	fmt.Println("inside publish")

	globalContext, globalSpan := otel.Tracer("Server").Start(globalContext, "publish method")
	publishStartTime := time.Now()

	msg := broker.Message{
		Body:       (string)(request.Body),
		Expiration: time.Duration(request.ExpirationSeconds) * time.Second,
	}

	publishId, err := s.BrokerInstance.Publish(globalContext, request.Subject, msg)
	publishDuration := time.Since(publishStartTime)
	MethodDuration.WithLabelValues("publish_duration").Observe(float64(publishDuration) / float64(time.Nanosecond))

	if err != nil {
		MethodCount.WithLabelValues("publish", "failed").Inc()

		return nil, err
	}

	MethodCount.WithLabelValues("publish", "successful").Inc()

	globalSpan.End()

	return &proto.PublishResponse{Id: int32(publishId)}, nil
}

func (s *Server) Subscribe(request *proto.SubscribeRequest, server proto.Broker_SubscribeServer) error {
	fmt.Println("Subscriber request received.")
	var subscribeError error

	SubscribedChannel, err := s.BrokerInstance.Subscribe(
		server.Context(),
		request.Subject,
	)

	if err != nil {
		MethodCount.WithLabelValues("subscribe", "failed").Inc()
		return err
	}

	ActiveSubscribersGauge.Inc()

	ctx := server.Context()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		for {
			select {
			case msg, ok := <-SubscribedChannel:
				if !ok {
					ActiveSubscribersGauge.Dec()
					wg.Done()

					return
				}

				if err := server.Send(&(proto.MessageResponse{Body: []byte(msg.Body)})); err != nil {
					subscribeError = err
				}
			case <-ctx.Done():
				subscribeError = errors.New("context timeout reached")

				ActiveSubscribersGauge.Dec()
				wg.Done()

				return
			}
		}
	}()

	wg.Wait()

	if subscribeError == nil {
		MethodCount.WithLabelValues("subscribe", "successful").Inc()
	} else {
		MethodCount.WithLabelValues("subscribe", "failed").Inc()
	}

	return subscribeError
}

func (s *Server) Fetch(ctx context.Context, request *proto.FetchRequest) (*proto.MessageResponse, error) {
	fetchStartTime := time.Now()

	log.Println("Getting fetch request")
	defer log.Println("Finish handling fetch request")

	msg, err := s.BrokerInstance.Fetch(ctx, request.Subject, int(request.Id))

	if err != nil {
		MethodCount.WithLabelValues("fetch", "failed").Inc()

		return nil, err
	}

	fetchDuration := time.Since(fetchStartTime)
	MethodDuration.WithLabelValues("fetch_duration").Observe(float64(fetchDuration) / float64(time.Nanosecond))
	MethodCount.WithLabelValues("fetch", "successful").Inc()

	return &proto.MessageResponse{Body: []byte(msg.Body)}, nil
}
