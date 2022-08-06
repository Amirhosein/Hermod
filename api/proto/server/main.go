package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"therealbroker/api/proto"
	"therealbroker/api/proto/server/handler"
	"therealbroker/internal/broker"

	"google.golang.org/grpc"
)

var (
	Port = 8080
)

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", Port))
	if err != nil {
		log.Fatalf("Cant start listener: %v", err)
	}

	server := grpc.NewServer()

	proto.RegisterBrokerServer(
		server,
		&handler.Server{
			BrokerInstance:  broker.NewModule(),
			LastPublishLock: &sync.Mutex{},
			LastTopicLock:   &sync.Mutex{},
			LastPublishId:   0,
		},
	)

	log.Printf("Server starting at: %s", listener.Addr())

	if err := server.Serve(listener); err != nil {
		log.Fatalf("Server serve failed: %v", err)
	}
}
