package handler

import (
	"context"
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
	fmt.Println("inside subscribe")

	_, err := s.BrokerInstance.Subscribe(server.Context(), request.Subject)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (s *Server) Fetch(ctx context.Context, request *proto.FetchRequest) (*proto.MessageResponse, error) {
	log.Println("Getting fetch request")
	defer log.Println("Finish handling fetch request")

	msg, err := s.BrokerInstance.Fetch(ctx, request.Subject, int(request.Id))

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &proto.MessageResponse{Body: []byte(msg.Body)}, nil
}
