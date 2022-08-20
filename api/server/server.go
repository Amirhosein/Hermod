package server

import (
	"fmt"
	"log"
	"net"
	"therealbroker/api/proto"
	"therealbroker/api/server/handler"
	"therealbroker/internal/broker"
	"therealbroker/internal/telemetry"
	"therealbroker/pkg/metric"

	"google.golang.org/grpc"
)

var (
	Port = 8080
)

func Init() {
	go metric.StartPrometheusServer()
	telemetry.Register()
}

func Run() {
	Init()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", Port))
	if err != nil {
		log.Fatalf("Cant start listener: %v", err)
	}

	server := grpc.NewServer()

	proto.RegisterBrokerServer(
		server,
		&handler.Server{
			BrokerInstance: broker.NewModule(),
		},
	)

	log.Printf("Server starting at: %s", listener.Addr())

	if err := server.Serve(listener); err != nil {
		log.Fatalf("Server serve failed: %v", err)
	}
}
