package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"therealbroker/api/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	Port    = 8080
	subject = "test"
)

func main() {
	connection, err := grpc.Dial(fmt.Sprintf("localhost:%d", Port), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("can't connect to server: %v", err)
	}

	defer func(clientConn *grpc.ClientConn) {
		err := clientConn.Close()
		if err != nil {
			log.Fatalf("can't close server connection: %v", err)
		}
	}(connection)

	client := proto.NewBrokerClient(connection)
	ctx := context.Background()

	// pushToSubject(client, ctx, subject, "some body for testing", int(10*time.Hour))

	var wg sync.WaitGroup

	ticker := time.NewTicker(1000 * time.Millisecond)
	ticker2 := time.NewTicker(2000 * time.Millisecond)
	doneIndicator := make(chan bool)

	channel, _ := client.Subscribe(ctx, &proto.SubscribeRequest{
		Subject: subject,
	})

	wg.Add(1)

	go func() {
		defer wg.Done()

		i := 0

		for {
			select {
			case <-doneIndicator:
				return
			case <-ticker.C:
				i++
				body := fmt.Sprintf("some text for testing %d : %v", i, time.Now())

				go pushToSubject(client, ctx, subject, body, 3600)
			case <-ticker2.C:
				go func() {
					msg, _ := channel.Recv()
					if msg != nil {
						log.Println("received message: ", (string)(msg.Body))
					}
				}()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Minute)
		ticker.Stop()
		doneIndicator <- true
	}()
	wg.Wait()
}

func pushToSubject(client proto.BrokerClient, ctx context.Context, subject string, body string, expire int) {
	response, err := client.Publish(ctx, &proto.PublishRequest{
		Subject:           subject,
		Body:              []byte(body),
		ExpirationSeconds: int32(expire),
	})

	if err != nil {
		log.Printf("publish to subject failed: %s\n", err)
		return
	}

	log.Println("response Id: ", response.Id)
}
